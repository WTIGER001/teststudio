package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

func login(ctx context.Context, args []string, writer io.Writer) error {
	flags := flag.NewFlagSet("login", flag.ContinueOnError)
	flags.SetOutput(writer)
	server := flags.String("server", "", "Exact TestStudio public origin")
	local := flags.Bool("local", false, "Use the server's enabled loopback administrator sign-in")
	user := flags.String("user", "", "Local account identifier")
	passwordEnv := flags.String("password-env", "", "Environment variable containing the password")
	filename := flags.String("session-file", sessionPath(), "Private managed-session file")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 || *server == "" || (*local && (*user != "" || *passwordEnv != "")) || (!*local && (*user == "" || *passwordEnv == "")) {
		return errors.New("choose --local or --user with --password-env")
	}
	origin, err := serverOrigin(*server)
	if err != nil {
		return err
	}
	if *local && !loopback(origin.Hostname()) {
		return errors.New("local administrator sign-in requires a loopback server")
	}
	client := newClient(loginFile{Origin: origin.String()})
	if previous, err := readClient(*filename); err == nil && previous.session.Origin == origin.String() {
		client = previous
	}
	endpoint := "/auth/local"
	body := map[string]string{}
	if !*local {
		password, ok := os.LookupEnv(*passwordEnv)
		if !ok || password == "" {
			return errors.New("password environment variable is empty")
		}
		endpoint = "/auth/password"
		body = map[string]string{"userId": *user, "password": password}
	}
	if err := client.request(ctx, "POST", endpoint, body, 0, "", nil); err != nil {
		return err
	}
	var authenticated struct {
		Session *struct {
			CSRF       string    `json:"csrfToken"`
			ExpiresAt  time.Time `json:"expiresAt"`
			MustChange bool      `json:"mustChangePassword"`
		} `json:"session"`
	}
	if err := client.request(ctx, "GET", "/auth/session", nil, 0, "", &authenticated); err != nil {
		return err
	}
	if authenticated.Session == nil {
		return errors.New("server did not establish a managed session")
	}
	if authenticated.Session.MustChange {
		client.session.CSRF = authenticated.Session.CSRF
		_ = client.request(ctx, "POST", "/auth/logout", map[string]any{}, 0, "", nil)
		return errors.New("complete the required password change in the browser before CLI sign-in")
	}
	token := ""
	for _, cookie := range client.client.Jar.Cookies(origin) {
		if cookie.Name == "teststudio_session" {
			token = cookie.Value
		}
	}
	if !sessionToken.MatchString(token) {
		return errors.New("server did not issue a valid managed session cookie")
	}
	session := loginFile{Schema: "teststudio.publisher-session/v1", Origin: origin.String(), Cookie: token, CSRF: authenticated.Session.CSRF, ExpiresAt: authenticated.Session.ExpiresAt}
	if err := saveLogin(*filename, session); err != nil {
		return err
	}
	fmt.Fprintln(writer, "CLI session saved privately:", *filename)
	return nil
}
func logout(ctx context.Context, args []string, writer io.Writer) error {
	flags := flag.NewFlagSet("logout", flag.ContinueOnError)
	flags.SetOutput(writer)
	filename := flags.String("session-file", sessionPath(), "Private managed-session file")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected logout arguments")
	}
	client, err := readClient(*filename)
	if err != nil {
		return err
	}
	if err := client.request(ctx, "POST", "/auth/logout", map[string]any{}, 0, "", nil); err != nil {
		return err
	}
	if err := os.Remove(*filename); err != nil {
		return err
	}
	fmt.Fprintln(writer, "CLI session revoked.")
	return nil
}
