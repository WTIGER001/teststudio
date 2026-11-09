package httpapi

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"time"

	authoidc "github.com/wtiger001/teststudio/adapters/auth/oidc"
	"github.com/wtiger001/teststudio/internal/sessions"
)

func (s *Server) authRoutes(m *http.ServeMux) {
	s.cliAuthenticationRoutes(m)
	m.HandleFunc("GET /auth/providers", s.signInOptions)
	m.HandleFunc("POST /auth/local", func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() || s.LocalUser == "" || r.Header.Get("Origin") != s.Sessions.Origin {
			JSON(w, 403, map[string]string{"message": "local sign-in is unavailable"})
			return
		}
		if _, err := s.Sessions.CreateReplacing(r.Context(), w, s.LocalUser, "local", time.Time{}, sessionFromCookie(r)); err != nil {
			problem(w, err)
			return
		}
		clearLoginTransaction(w, s.Sessions.Secure)
		JSON(w, 200, map[string]bool{"signedIn": true})
	})
	m.HandleFunc("GET /auth/session", func(w http.ResponseWriter, r *http.Request) {
		v, err := s.Sessions.Read(r.Context(), r)
		if err != nil {
			if errors.Is(err, sessions.ErrUnauthenticated) {
				JSON(w, 200, map[string]any{"user": nil, "session": nil})
				return
			}
			problem(w, err)
			return
		}
		u, err := s.Auth.Directory.User(r.Context(), v.UserID)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, map[string]any{"user": u, "session": v})
	})
	m.HandleFunc("POST /auth/logout", s.authenticated(func(w http.ResponseWriter, r *http.Request) {
		v := r.Context().Value(sessionKey{}).(sessions.Session)
		if err := s.Sessions.Revoke(r.Context(), v.ID); err != nil {
			problem(w, err)
			return
		}
		s.Sessions.Clear(w)
		clearLoginTransaction(w, s.Sessions.Secure)
		result := authoidc.LogoutResponse{SignedOut: true}
		if s.OIDC != nil {
			offer, cookie, err := s.OIDC.OfferLogout(r.Context(), v)
			if err != nil {
				result.ProviderLogoutUnavailable = true
			} else if offer != nil {
				result.ProviderLogout = offer
				http.SetCookie(w, cookie)
			}
		}
		JSON(w, 200, result)
	}, true))
	m.HandleFunc("GET /auth/logout/{offer}", func(w http.ResponseWriter, r *http.Request) {
		if s.OIDC == nil || s.OIDC.LaunchLogout(w, r, r.PathValue("offer")) != nil {
			providerLogoutFailure(w, r)
		}
	})
	m.HandleFunc("GET /auth/logout/callback/{provider}", func(w http.ResponseWriter, r *http.Request) {
		if s.OIDC == nil || s.OIDC.CompleteLogout(w, r, r.PathValue("provider")) != nil {
			providerLogoutFailure(w, r)
		}
	})
	m.HandleFunc("GET /auth/login/{provider}", func(w http.ResponseWriter, r *http.Request) {
		if err := s.OIDC.Login(w, r, r.PathValue("provider")); err != nil {
			http.Redirect(w, r, "/projects?signIn=unavailable", http.StatusFound)
		}
	})
	m.HandleFunc("GET /auth/callback/{provider}", func(w http.ResponseWriter, r *http.Request) {
		if err := s.OIDC.Callback(w, r, r.PathValue("provider")); err != nil {
			outcome := "failed"
			if errors.Is(err, authoidc.ErrSignInCancelled) {
				outcome = "cancelled"
			}
			if errors.Is(err, authoidc.ErrSignInExpired) {
				outcome = "expired"
			}
			w.Header().Set("Cache-Control", "no-store")
			target := "/projects"
			var failure *authoidc.SignInError
			if errors.As(err, &failure) && failure.ReturnTo != "/" {
				target = failure.ReturnTo
			}
			destination, parseErr := url.Parse(target)
			if parseErr != nil {
				destination = &url.URL{Path: "/projects"}
			}
			query := destination.Query()
			query.Set("signIn", outcome)
			destination.RawQuery = query.Encode()
			http.Redirect(w, r, destination.String(), http.StatusFound)
		}
	})
}

func sessionFromCookie(r *http.Request) string {
	if cookie, err := r.Cookie("teststudio_session"); err == nil && len(cookie.Value) == 43 {
		return sessions.Digest(cookie.Value)
	}
	return ""
}

func clearLoginTransaction(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{Name: "teststudio_login", Value: "", Path: "/auth", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
}

func providerLogoutFailure(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	http.Redirect(w, r, "/projects?providerLogout=unconfirmed", http.StatusFound)
}
