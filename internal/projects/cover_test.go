package projects_test

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

func coverImage(t *testing.T) string {
	t.Helper()
	var data bytes.Buffer
	canvas := image.NewRGBA(image.Rect(0, 0, 32, 24))
	canvas.Set(4, 4, color.RGBA{255, 100, 0, 255})
	if err := jpeg.Encode(&data, canvas, nil); err != nil {
		t.Fatal(err)
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(data.Bytes())
}

func TestProjectCoverPrivateRevisionedLifecycle(t *testing.T) {
	s := creatorFixture(t)
	if err := s.SaveProject(t.Context(), "creator", projects.Project{ID: "alpha", Name: "Alpha"}, 0); err != nil {
		t.Fatal(err)
	}
	before, err := s.ProjectCover(t.Context(), "creator", "alpha")
	if err != nil || before.Revision != 0 || before.HasImage || !before.CanManage {
		t.Fatal(before, err)
	}
	input := projects.Cover{Image: coverImage(t), Source: "capture", URL: "https://example.test"}
	value, err := s.SaveProjectCover(t.Context(), "creator", "alpha", input, 0)
	if err != nil || !value.HasImage || value.Revision != 1 || value.Cover.Source != "capture" {
		t.Fatal(value, err)
	}
	if _, err := s.SaveProjectCover(t.Context(), "creator", "alpha", input, 0); !errors.Is(err, storage.ErrConflict) {
		t.Fatal("stale image accepted", err)
	}
	if _, err := s.ProjectCover(t.Context(), "other", "alpha"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("private image disclosed", err)
	}
	if rows, err := s.ProjectCovers(t.Context(), "other"); err != nil || len(rows) != 0 {
		t.Fatal("private metadata disclosed", rows, err)
	}
	if _, err := s.SaveProjectCover(t.Context(), "other", "alpha", input, 1); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("unauthorized image changed", err)
	}
	value, err = s.SaveProjectCover(t.Context(), "creator", "alpha", projects.Cover{}, 1)
	if err != nil || value.HasImage || value.Revision != 2 {
		t.Fatal(value, err)
	}
	row, err := s.Store.Get(t.Context(), "projects", "alpha")
	if err != nil || row.Revision != 1 {
		t.Fatal("image editing changed project settings", row.Revision, err)
	}
	rows, err := s.Store.All(t.Context(), "audit", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if bytes.Contains(row.Data, []byte("data:image")) {
			t.Fatal("audit duplicates private image data")
		}
	}
}

func TestProjectCoverRejectsInvalidInputWithoutChangingRevision(t *testing.T) {
	s := creatorFixture(t)
	if err := s.SaveProject(t.Context(), "creator", projects.Project{ID: "alpha", Name: "Alpha"}, 0); err != nil {
		t.Fatal(err)
	}
	image := coverImage(t)
	for _, input := range []projects.Cover{
		{Image: "data:image/svg+xml,<svg/>", Source: "upload"},
		{Image: "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString([]byte("not an image")), Source: "upload"},
		{Image: image, Source: "other"},
		{Image: image, Source: "capture", URL: "https://user:secret@example.test"},
		{Image: image, Source: "capture", URL: "javascript:alert(1)"},
		{Image: "data:image/jpeg;base64," + strings.Repeat("A", 400000), Source: "upload"},
	} {
		if _, err := s.SaveProjectCover(t.Context(), "creator", "alpha", input, 0); err == nil {
			t.Fatal("invalid image accepted")
		}
	}
	value, err := s.ProjectCover(t.Context(), "creator", "alpha")
	if err != nil || value.Revision != 0 {
		t.Fatal("invalid images changed stored state", value.Revision, err)
	}
}
