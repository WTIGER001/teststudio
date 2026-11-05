package browsergateway

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/domain"
)

const RemoteProtocol = "tls-execution/v1"
const executionHeader = "X-TestStudio-Gateway-Execution"

var ErrRemoteConfiguration = errors.New("remote gateway requires a valid scoped execution, private TLS connection, and unexpired deadline")

type RemoteSession struct {
	Execution string    `json:"executionId"`
	Token     string    `json:"token"`
	Deadline  time.Time `json:"deadline"`
}

type RemoteConnection struct {
	Origin               string        `json:"origin"`
	CertificateAuthority []byte        `json:"certificateAuthority,omitempty"`
	Session              RemoteSession `json:"session"`
}

func (s RemoteSession) validate() error {
	token, err := base64.RawURLEncoding.Strict().DecodeString(s.Token)
	if err != nil || len(token) != 32 || !domain.ValidID(s.Execution) || !s.Deadline.After(time.Now()) || s.Deadline.After(time.Now().Add(2*time.Hour)) {
		return ErrRemoteConfiguration
	}
	return nil
}

func (c RemoteConnection) validate() error {
	origin, err := browserpolicy.Origin(c.Origin)
	if err != nil || origin != c.Origin || !strings.HasPrefix(origin, "https://") || len(c.CertificateAuthority) > 64<<10 {
		return ErrRemoteConfiguration
	}
	return c.Session.validate()
}

func (s RemoteSession) String() string      { return "gateway session: " + s.Execution }
func (s RemoteSession) GoString() string    { return s.String() }
func (c RemoteConnection) String() string   { return "gateway connection: " + c.Session.Execution }
func (c RemoteConnection) GoString() string { return c.String() }

type Remote struct {
	gateway *Gateway
	session RemoteSession
}

func (g *Remote) String() string   { return "trusted remote gateway" }
func (g *Remote) GoString() string { return g.String() }

func NewRemote(configuration Configuration) (*Remote, error) {
	if configuration.Remote == nil || configuration.Remote.validate() != nil {
		return nil, ErrRemoteConfiguration
	}
	gateway, err := NewAuthenticated(configuration.Target, configuration.Credential)
	if err != nil {
		return nil, err
	}
	return &Remote{gateway: gateway, session: *configuration.Remote}, nil
}

func (g *Remote) Close() { g.gateway.Close() }

func (g *Remote) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.TLS == nil || !r.TLS.HandshakeComplete || r.ProtoMajor != 1 {
		http.Error(w, "Gateway transport requires authenticated TLS", http.StatusForbidden)
		return
	}
	authorization := r.Header.Values("Proxy-Authorization")
	execution := r.Header.Values(executionHeader)
	expected := "Bearer " + g.session.Token
	if len(authorization) != 1 || len(execution) != 1 || execution[0] != g.session.Execution || subtle.ConstantTimeCompare([]byte(authorization[0]), []byte(expected)) != 1 || !time.Now().Before(g.session.Deadline) {
		w.Header().Set("Proxy-Authenticate", `Bearer realm="teststudio-gateway"`)
		http.Error(w, "Gateway authorization is unavailable", http.StatusProxyAuthRequired)
		return
	}
	request := r.Clone(r.Context())
	request.Header.Del("Proxy-Authorization")
	request.Header.Del(executionHeader)
	g.gateway.ServeHTTP(w, request)
}
