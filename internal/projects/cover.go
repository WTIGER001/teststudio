package projects

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image/jpeg"
	"net/url"
	"strings"
	"unicode"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

const CoverLimit = 256 << 10

type Cover struct {
	Image  string `json:"image"`
	Source string `json:"source,omitempty"`
	URL    string `json:"url,omitempty"`
}

type CoverInfo struct {
	ID        string `json:"id"`
	Revision  int64  `json:"revision"`
	HasImage  bool   `json:"hasImage"`
	CanManage bool   `json:"canManage"`
}

type CoverDetail struct {
	CoverInfo
	Cover Cover `json:"cover"`
}

func (s *Service) ProjectCover(ctx context.Context, user, project string) (CoverDetail, error) {
	result := CoverDetail{CoverInfo: CoverInfo{ID: project}}
	if !domain.ValidID(project) {
		return result, errors.New("valid project identifier is required")
	}
	if err := s.Auth.Check(ctx, user, project, "", "projects.read"); err != nil {
		return result, err
	}
	if _, err := s.Store.Get(ctx, "projects", project); err != nil {
		return result, err
	}
	if err := s.Auth.Check(ctx, user, project, "", "projects.manage"); err == nil {
		result.CanManage = true
	} else if !errors.Is(err, authorization.ErrForbidden) {
		return result, err
	}
	row, err := s.Store.Get(ctx, "project-covers", project)
	if errors.Is(err, storage.ErrNotFound) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	result.Cover, err = storage.Decode[Cover](row)
	result.Revision, result.HasImage = row.Revision, result.Cover.Image != ""
	return result, err
}

func (s *Service) ProjectCovers(ctx context.Context, user string) ([]CoverInfo, error) {
	rows, err := s.Store.All(ctx, "projects", "")
	if err != nil {
		return nil, err
	}
	result := []CoverInfo{}
	for _, row := range rows {
		value, err := s.ProjectCover(ctx, user, row.ID)
		if errors.Is(err, authorization.ErrForbidden) {
			continue
		}
		if err != nil {
			return nil, err
		}
		result = append(result, value.CoverInfo)
	}
	return result, nil
}

func (s *Service) SaveProjectCover(ctx context.Context, user, project string, input Cover, revision int64) (CoverDetail, error) {
	err := s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth}
		before, err := bound.ProjectCover(ctx, user, project)
		if err != nil {
			return err
		}
		if !before.CanManage {
			return authorization.ErrForbidden
		}
		if before.Revision != revision {
			return storage.ErrConflict
		}
		clean, err := cleanCover(input)
		if err != nil {
			return err
		}
		change, err := storage.Put("project-covers", project, project, revision, clean)
		if err != nil {
			return err
		}
		audit, err := domain.Audit(user, project, "project.cover", project, map[string]any{"revision": before.Revision, "hasImage": before.HasImage}, map[string]any{"hasImage": clean.Image != "", "source": clean.Source})
		if err != nil {
			return err
		}
		return bound.Store.Commit(ctx, change, audit)
	})
	if err != nil {
		return CoverDetail{}, err
	}
	return s.ProjectCover(ctx, user, project)
}

func cleanCover(input Cover) (Cover, error) {
	if input.Image == "" {
		return Cover{}, nil
	}
	if (input.Source != "upload" && input.Source != "capture") || (input.URL != "" && !validCoverURL(input.URL)) {
		return Cover{}, errors.New("select upload or capture and use an HTTP or HTTPS address without credentials")
	}
	const prefix = "data:image/jpeg;base64,"
	if !strings.HasPrefix(input.Image, prefix) || len(input.Image) > base64.StdEncoding.EncodedLen(CoverLimit)+len(prefix) {
		return Cover{}, errors.New("project images must be JPEG and no larger than 256 KiB")
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(input.Image, prefix))
	if err != nil || len(data) > CoverLimit {
		return Cover{}, errors.New("project image data is invalid or too large")
	}
	config, err := jpeg.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width > 1600 || config.Height > 1600 || config.Width < 1 || config.Height < 1 {
		return Cover{}, errors.New("project images must be valid JPEG images up to 1600 pixels per side")
	}
	image, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return Cover{}, errors.New("project image cannot be decoded")
	}
	var output bytes.Buffer
	for _, quality := range []int{80, 65, 45, 25} {
		output.Reset()
		if err := jpeg.Encode(&output, image, &jpeg.Options{Quality: quality}); err != nil {
			return Cover{}, err
		}
		if output.Len() <= CoverLimit {
			break
		}
	}
	if output.Len() > CoverLimit {
		return Cover{}, errors.New("project image is too detailed; choose a smaller image")
	}
	input.Image = prefix + base64.StdEncoding.EncodeToString(output.Bytes())
	return input, nil
}

func validCoverURL(value string) bool {
	u, err := url.Parse(value)
	return len(value) <= 2048 && err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != "" && u.User == nil && !strings.ContainsFunc(value, unicode.IsControl)
}
