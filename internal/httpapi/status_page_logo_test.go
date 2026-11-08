package httpapi_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/wtiger001/teststudio/internal/statuspages"
)

func TestStatusPageLogosFollowPublicationAndRejectInvalidImages(t *testing.T) {
	c, _, binding := probeFixture(t)
	var imageData bytes.Buffer
	if err := png.Encode(&imageData, image.NewNRGBA(image.Rect(0, 0, 24, 12))); err != nil {
		t.Fatal(err)
	}
	logo := "data:image/png;base64," + base64.StdEncoding.EncodeToString(imageData.Bytes())
	definition := statuspages.Definition{ID: "logo-page", Title: "Public health", Background: "#ffffff", Window: "24h", MaxAge: 900, Logo: logo, LogoAlt: "Example brand", Probes: []statuspages.Entry{{Probe: binding.ID, Title: "Website"}}}

	var jpegData bytes.Buffer
	if err := jpeg.Encode(&jpegData, image.NewNRGBA(image.Rect(0, 0, 24, 12)), nil); err != nil {
		t.Fatal(err)
	}
	jpegPage := definition
	jpegPage.ID = "jpeg-logo"
	jpegPage.Logo = "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(jpegData.Bytes())
	jpegRow := accessCall(t, c, "PUT", "/api/v1/projects/example/status-pages/jpeg-logo", jpegPage, "0", 200)
	if jpegRow["data"].(map[string]any)["logo"] != jpegPage.Logo {
		t.Fatal("valid JPEG logo rejected")
	}
	base := "/api/v1/projects/example/status-pages/logo-page"
	row := accessCall(t, c, "PUT", base, definition, "0", 200)
	if row["data"].(map[string]any)["logo"] != logo {
		t.Fatal("uploaded logo was not retained")
	}
	token := row["data"].(map[string]any)["publicToken"].(string)
	public := "/api/v1/public/status-pages/" + token
	anon := *c
	anon.http = &http.Client{}
	accessCall(t, &anon, "GET", public, nil, "", 404)
	accessCall(t, c, "POST", base+"/publish", nil, "1", 200)
	report := accessCall(t, &anon, "GET", public, nil, "", 200)
	if report["logo"] != logo || report["logoAlt"] != "Example brand" {
		t.Fatal("published logo or alternative text missing")
	}
	var requests atomic.Int32
	host := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1); w.WriteHeader(200) }))
	defer host.Close()
	definition.Logo = host.URL + "/public-logo.svg"
	accessCall(t, c, "PUT", base, definition, "2", 200)
	if accessCall(t, &anon, "GET", public, nil, "", 200)["logo"] != logo {
		t.Fatal("draft logo leaked")
	}
	accessCall(t, c, "POST", base+"/publish", nil, "3", 200)
	if accessCall(t, &anon, "GET", public, nil, "", 200)["logo"] != definition.Logo {
		t.Fatal("referenced logo missing")
	}
	if requests.Load() != 0 {
		t.Fatal("the service fetched an image URL")
	}
	invalid := []string{"http://example.com/logo.png", "javascript:alert(1)", "/private-logo", "https://user:secret@example.com/logo.png", "data:image/svg+xml,<svg/>", "data:image/png;base64,bad", "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(imageData.Bytes()), "data:image/png;base64," + base64.StdEncoding.EncodeToString(imageData.Bytes()[:len(imageData.Bytes())-12]), strings.Repeat("x", 256*1024+1)}
	var large bytes.Buffer
	if err := png.Encode(&large, image.NewNRGBA(image.Rect(0, 0, 1601, 1))); err != nil {
		t.Fatal(err)
	}
	invalid = append(invalid, "data:image/png;base64,"+base64.StdEncoding.EncodeToString(large.Bytes()))
	for index, value := range invalid {
		t.Run(fmt.Sprint(index), func(t *testing.T) { d := definition; d.Logo = value; accessCall(t, c, "PUT", base, d, "4", 422) })
	}
	definition.LogoAlt = strings.Repeat("a", 257)
	accessCall(t, c, "PUT", base, definition, "4", 422)
	definition.Logo = ""
	definition.LogoAlt = "Cleared with logo"
	saved := accessCall(t, c, "PUT", base, definition, "4", 200)
	if saved["data"].(map[string]any)["logoAlt"] != nil {
		t.Fatal("unused alternative text retained")
	}
	if accessCall(t, &anon, "GET", public, nil, "", 200)["logo"] == nil {
		t.Fatal("draft removal affected publication")
	}
	accessCall(t, c, "POST", base+"/publish", nil, "5", 200)
	if accessCall(t, &anon, "GET", public, nil, "", 200)["logo"] != nil {
		t.Fatal("logo removal was not published")
	}
	accessCall(t, c, "POST", base+"/unpublish", nil, "6", 200)
	accessCall(t, &anon, "GET", public, nil, "", 404)
}
