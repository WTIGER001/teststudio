package httpapi_test

import (
	"github.com/wtiger001/teststudio/internal/statuspages"
	"net/http"
	"testing"
)

func TestStatusPageLogoHeightValidationAndPublishedSnapshot(t *testing.T) {
	c, _, probe := probeFixture(t)
	d := statuspages.Definition{ID: "logo-height", Title: "Health", Background: "#ffffff", Window: "24h", MaxAge: 900, Logo: "https://example.test/logo.png", LogoHeight: 96, Probes: []statuspages.Entry{{Probe: probe.ID, Title: "Website"}}}
	path := "/api/v1/projects/example/status-pages/logo-height"
	row := accessCall(t, c, "PUT", path, d, "0", 200)
	if row["data"].(map[string]any)["logoHeight"] != float64(96) {
		t.Fatal("height was not saved")
	}
	anon := *c
	anon.http = &http.Client{}
	public := "/api/v1/public/status-pages/" + row["data"].(map[string]any)["publicToken"].(string)
	accessCall(t, c, "POST", path+"/publish", nil, "1", 200)
	if accessCall(t, &anon, "GET", public, nil, "", 200)["logoHeight"] != float64(96) {
		t.Fatal("published height missing")
	}
	d.LogoHeight = 160
	accessCall(t, c, "PUT", path, d, "2", 200)
	if accessCall(t, &anon, "GET", public, nil, "", 200)["logoHeight"] != float64(96) {
		t.Fatal("draft height leaked")
	}
	accessCall(t, c, "POST", path+"/publish", nil, "3", 200)
	if accessCall(t, &anon, "GET", public, nil, "", 200)["logoHeight"] != float64(160) {
		t.Fatal("new height was not published")
	}
	for _, height := range []int{-1, 1025} {
		d.LogoHeight = height
		accessCall(t, c, "PUT", path, d, "4", 422)
	}
	for _, height := range []int{1, 1024, 0} {
		d.ID = "height-boundary"
		d.LogoHeight = height
		if height == 1 {
			accessCall(t, c, "PUT", "/api/v1/projects/example/status-pages/height-boundary", d, "0", 200)
		} else if height == 1024 {
			accessCall(t, c, "PUT", "/api/v1/projects/example/status-pages/height-boundary", d, "1", 200)
		} else {
			saved := accessCall(t, c, "PUT", "/api/v1/projects/example/status-pages/height-boundary", d, "2", 200)
			if saved["data"].(map[string]any)["logoHeight"] != nil {
				t.Fatal("automatic height should be omitted")
			}
		}
	}
	d.ID = "logo-height"
	d.Logo = ""
	d.LogoHeight = 160
	saved := accessCall(t, c, "PUT", path, d, "4", 200)
	if saved["data"].(map[string]any)["logoHeight"] != nil {
		t.Fatal("height remains without a logo")
	}
}
