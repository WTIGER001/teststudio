package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"time"

	"github.com/wtiger001/teststudio/internal/cliauth"
)

func browserLogin(ctx context.Context, origin *url.URL, name, filename string, noOpen bool, writer io.Writer) error {
	client := newClient(loginFile{Origin: origin.String()})
	previous, _ := readClient(filename)
	if previous != nil && previous.session.Origin != origin.String() {
		previous = nil
	}
	var options cliauth.Options
	if err := client.request(ctx, "GET", "/auth/cli/options", nil, 0, "", &options); err != nil {
		return err
	}
	if options.LifetimeSeconds != int(cliauth.Lifetime/time.Second) || options.PollSeconds < 1 || options.PollSeconds > 10 {
		return errors.New("server CLI sign-in options are unsupported")
	}
	id, secret, code, err := cliauth.NewRequest(options.ServerTime, name)
	if err != nil {
		return err
	}
	proof := cliauth.Proof{Request: id, Client: name, Verifier: secret}
	browser := origin.String() + "/cli-sign-in/" + id + "?clientName=" + url.QueryEscape(name)
	fmt.Fprintln(writer, "Open this address to confirm CLI sign-in:", browser)
	fmt.Fprintln(writer, "Terminal code:", code)
	fmt.Fprintln(writer, "Client:", name)
	fmt.Fprintln(writer, "Sign in with your configured account, compare the terminal code, and approve this request. The request expires in five minutes.")
	if !noOpen {
		if err := openBrowser(ctx, browser); err != nil {
			fmt.Fprintln(writer, "The browser could not open automatically. Open the address above manually.")
		}
	}
	saved := false
	defer func() {
		if !saved {
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			_ = client.request(cleanup, "POST", "/auth/cli/cancel", proof, 0, "", nil)
		}
	}()
	waiting, cancel := context.WithTimeout(ctx, cliauth.Lifetime)
	defer cancel()
	timer := time.NewTicker(time.Duration(options.PollSeconds) * time.Second)
	defer timer.Stop()
	for {
		var exchanged cliauth.Exchange
		err := client.request(waiting, "POST", "/auth/cli/exchange", proof, 0, "", &exchanged)
		var response responseError
		if err != nil && !(errors.As(err, &response) && response.status == http.StatusConflict) {
			return err
		}
		if err == nil {
			switch exchanged.State {
			case "authorized":
				if exchanged.Session == nil || !sessionToken.MatchString(exchanged.Session.Cookie) || exchanged.Session.CSRF == "" || !time.Now().Before(exchanged.Session.ExpiresAt) {
					return errors.New("server did not issue a valid CLI session")
				}
				session := loginFile{Schema: "teststudio.publisher-session/v1", Origin: origin.String(), Cookie: exchanged.Session.Cookie, CSRF: exchanged.Session.CSRF, ExpiresAt: exchanged.Session.ExpiresAt}
				if err := saveLogin(filename, session); err != nil {
					return err
				}
				saved = true
				if previous != nil {
					err := previous.request(ctx, "POST", "/auth/logout", map[string]any{}, 0, "", nil)
					var rejected responseError
					if err != nil && !(errors.As(err, &rejected) && rejected.status == http.StatusUnauthorized) {
						fmt.Fprintln(writer, "The earlier CLI session could not be revoked. Remove it through user session administration.")
					}
				}
				fmt.Fprintln(writer, "CLI session saved privately:", filename)
				return nil
			case "pending":
			case "denied":
				return errors.New("CLI sign-in was declined in the browser")
			case "cancelled", "consumed":
				return errors.New("CLI sign-in is no longer available; start a new request")
			default:
				return errors.New("server returned an invalid CLI sign-in state")
			}
		}
		select {
		case <-waiting.Done():
			return errors.New("CLI sign-in stopped or expired; start it again when ready")
		case <-timer.C:
		}
	}
}
func openBrowser(ctx context.Context, address string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.CommandContext(ctx, "open", address)
	case "windows":
		command = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", address)
	default:
		command = exec.CommandContext(ctx, "xdg-open", address)
	}
	return command.Run()
}
