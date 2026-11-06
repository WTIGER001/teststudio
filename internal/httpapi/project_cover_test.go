package httpapi_test

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/jpeg"
	"net/http"
	"testing"

	"github.com/wtiger001/teststudio/internal/projects"
)

func TestProjectCoverHTTPContentAndRevisions(t *testing.T) {
	c := fixture(t)
	c.signIn()
	if status, data := c.call("POST", "/api/v1/projects", projects.Project{ID: "alpha", Name: "Alpha"}, nil); status != 200 {
		t.Fatal(status, data)
	}
	base := "/api/v1/projects/alpha/cover"
	var data bytes.Buffer
	if err := jpeg.Encode(&data, image.NewRGBA(image.Rect(0, 0, 30, 30)), nil); err != nil {
		t.Fatal(err)
	}
	input := projects.Cover{Image: "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(data.Bytes()), Source: "upload"}
	if status, _ := c.call("PUT", base, input, nil); status != 422 {
		t.Fatal("revision required", status)
	}
	if status, value := c.call("PUT", base, input, map[string]string{"If-Match": "0"}); status != 200 || value["hasImage"] != true {
		t.Fatal(status, value)
	}
	if status, _ := c.call("PUT", base, input, map[string]string{"If-Match": "0"}); status != 409 {
		t.Fatal("stale revision accepted", status)
	}
	response, err := c.http.Get(c.origin + base + "/content")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 || response.Header.Get("Content-Type") != "image/jpeg" || response.Header.Get("Cache-Control") != "private, no-store" {
		t.Fatal(response.StatusCode, response.Header)
	}
	if _, err := jpeg.Decode(response.Body); err != nil {
		t.Fatal(err)
	}
	anon, err := http.Get(c.origin + base + "/content")
	if err != nil {
		t.Fatal(err)
	}
	anon.Body.Close()
	if anon.StatusCode != 401 {
		t.Fatal("anonymous image access", anon.StatusCode)
	}
	if status, value := c.call("GET", "/api/v1/project-covers", nil, nil); status != 200 || len(value["items"].([]any)) != 1 {
		t.Fatal(status, value)
	}
}
