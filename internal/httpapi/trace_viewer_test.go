package httpapi_test

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/traceviewer"
)

func viewerBundle(t *testing.T) *traceviewer.Bundle {
	t.Helper()
	root := t.TempDir()
	hashes := map[string]string{}
	for _, name := range []string{"index.html", "snapshot.html", "sw.bundle.js", "sw.vendor.js", "teststudio-session.js", "LICENSE"} {
		body := []byte("trusted viewer " + name)
		digest := sha256.Sum256(body)
		hashes[name] = hex.EncodeToString(digest[:])
		if err := os.WriteFile(filepath.Join(root, name), body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	data, _ := json.Marshal(map[string]any{"schemaVersion": 1, "playwrightVersion": traceviewer.Version, "files": hashes})
	if err := os.WriteFile(filepath.Join(root, "teststudio-viewer.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	bundle, err := traceviewer.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return bundle
}

func newTrace(t *testing.T, c *client, run string) evidence.Artifact {
	t.Helper()
	var data bytes.Buffer
	writer := zip.NewWriter(&data)
	entry, err := writer.Create("trace.trace")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = entry.Write([]byte("{}\n"))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	a, err := c.api.Evidence.SaveBytes(t.Context(), evidence.Artifact{Project: "example", Run: run, Kind: "trace", Filename: "trace.zip", ContentType: "application/zip"}, data.Bytes(), 100<<20)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func getStatus(t *testing.T, address string) (int, http.Header) {
	t.Helper()
	response, err := http.Get(address)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	return response.StatusCode, response.Header
}

func TestPrivateTraceViewerBindsGrantArtifactSessionAndProject(t *testing.T) {
	c, id := preparedRun(t)
	c.api.TraceViewer = viewerBundle(t)
	a := newTrace(t, c, id)
	base := "/api/v1/projects/example/runs/" + id + "/artifacts/" + a.ID + "/access"
	status, grant := c.call("POST", base, map[string]string{"action": "view"}, nil)
	if status != 200 || grant["traceViewerUrl"] == nil {
		t.Fatal(status, grant)
	}
	viewer := grant["traceViewerUrl"].(string)
	u, _ := url.Parse(viewer)
	if status, headers := getStatus(t, viewer); status != 200 || !strings.Contains(headers.Get("Content-Security-Policy"), "connect-src 'self'") || !strings.Contains(headers.Get("Content-Security-Policy"), c.origin) || headers.Get("Cache-Control") != "no-store" || headers.Get("Referrer-Policy") != "no-referrer" {
		t.Fatal("viewer origin policy", status, headers)
	}
	scope := viewer[:strings.LastIndex(u.Scheme+"://"+u.Host+u.Path, "/")+1]
	for _, name := range []string{"sw.bundle.js", "snapshot.html", "teststudio-session.js", "authorization"} {
		if status, _ := getStatus(t, scope+name); status != 200 && status != 204 {
			t.Fatal(name, status)
		}
	}
	foreign := *u
	q := foreign.Query()
	q.Set("trace", c.api.ArtifactOrigin+"/access/"+strings.Repeat("x", 43))
	foreign.RawQuery = q.Encode()
	if status, _ := getStatus(t, foreign.String()); status != 422 {
		t.Fatal("viewer accepted another trace", status)
	}
	if status, data := c.call("POST", "/api/v1/projects/foreign/runs/"+id+"/artifacts/"+a.ID+"/access", map[string]string{"action": "view"}, nil); status != 404 {
		t.Fatal("trace crossed its project namespace", status, data)
	}
	_, download := c.call("POST", base, map[string]string{"action": "download"}, nil)
	token := strings.TrimPrefix(download["url"].(string), c.api.ArtifactOrigin+"/access/")
	if status, _ := getStatus(t, c.api.ArtifactOrigin+"/trace/"+token+"/sw.bundle.js"); status != 404 {
		t.Fatal("download grant became a viewer", status)
	}
	if err := c.api.Auth.Engine.UnassignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "admin"}, "builtin.teststudio.installation-admin", nil); err != nil {
		t.Fatal(err)
	}
	for _, address := range []string{viewer, scope + "authorization", scope + "sw.bundle.js", grant["url"].(string)} {
		if status, _ := getStatus(t, address); status != 403 {
			t.Fatal("viewer survived permission revocation", status)
		}
	}
}

func TestPrivateTraceViewerExpiryUnsupportedFilesAndOptionalPackaging(t *testing.T) {
	c, id := preparedRun(t)
	a := newTrace(t, c, id)
	base := "/api/v1/projects/example/runs/" + id + "/artifacts/" + a.ID + "/access"
	status, grant := c.call("POST", base, map[string]string{"action": "view"}, nil)
	if status != 200 || grant["traceViewerUrl"] != nil || grant["traceViewerUnavailable"] == nil {
		t.Fatal("missing bundle has no recovery", status, grant)
	}
	c.api.TraceViewer = viewerBundle(t)
	_, grant = c.call("POST", base, map[string]string{"action": "view"}, nil)
	token := strings.TrimPrefix(grant["url"].(string), c.api.ArtifactOrigin+"/access/")
	row, err := c.api.Store.Get(t.Context(), "artifact-access", sessions.Digest(token))
	if err != nil {
		t.Fatal(err)
	}
	value, err := storage.Decode[map[string]any](row)
	if err != nil {
		t.Fatal(err)
	}
	value["expiresAt"] = time.Now().Add(-time.Second)
	change, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, value)
	if err := c.api.Store.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	if status, _ := getStatus(t, grant["traceViewerUrl"].(string)); status != 401 {
		t.Fatal("expired viewer remained available", status)
	}
	bad, err := c.api.Evidence.SaveBytes(t.Context(), evidence.Artifact{Project: "example", Run: id, Kind: "trace", Filename: "invalid.zip", ContentType: "application/zip"}, []byte("not a zip"), 100<<20)
	if err != nil {
		t.Fatal(err)
	}
	_, invalid := c.call("POST", "/api/v1/projects/example/runs/"+id+"/artifacts/"+bad.ID+"/access", map[string]string{"action": "view"}, nil)
	if invalid["traceViewerUrl"] != nil || invalid["traceViewerUnavailable"] == nil {
		t.Fatal("unsupported archive was presented as viewable", invalid)
	}
	badToken := strings.TrimPrefix(invalid["url"].(string), c.api.ArtifactOrigin+"/access/")
	if status, _ := getStatus(t, c.api.ArtifactOrigin+"/trace/"+badToken+"/sw.bundle.js"); status != 404 {
		t.Fatal("unsupported archive bypassed its grant validation", status)
	}
}
