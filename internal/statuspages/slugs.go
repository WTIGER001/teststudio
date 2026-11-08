package statuspages

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
var tokenPattern = regexp.MustCompile(`^[a-f0-9]{40}$`)
var ErrSlugTaken = fmt.Errorf("that vanity URL is already reserved; choose another: %w", storage.ErrConflict)

type slugLink struct {
	ID    string `json:"pageId"`
	Token string `json:"publicToken"`
}
type SlugAvailability struct {
	Slug      string `json:"slug"`
	Available bool   `json:"available"`
}

func NormalizeSlug(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value != "" && (!slugPattern.MatchString(value) || tokenPattern.MatchString(value)) {
		return "", errors.New("use 1–63 letters, numbers, or hyphens, starting and ending with a letter or number; generated addresses are reserved")
	}
	return value, nil
}
func ownsSlug(row storage.Record, project, id, token string) bool {
	link, err := storage.Decode[slugLink](row)
	return err == nil && row.Project == project && link.ID == id && link.Token == token
}
func (s *Service) slugReservation(ctx context.Context, project string, page Page) ([]storage.Change, error) {
	if page.Slug == "" {
		return nil, nil
	}
	row, err := s.Store.Get(ctx, "status-page-slugs", page.Slug)
	if err == nil {
		if !ownsSlug(row, project, page.ID, page.Token) {
			return nil, ErrSlugTaken
		}
		return []storage.Change{{Record: row, Assert: true}}, nil
	}
	if !errors.Is(err, storage.ErrNotFound) {
		return nil, err
	}
	change, err := storage.Put("status-page-slugs", page.Slug, project, 0, slugLink{ID: page.ID, Token: page.Token})
	return []storage.Change{change}, err
}
func (s *Service) SlugAvailable(ctx context.Context, user, project, slug, id string) (SlugAvailability, error) {
	if id == "" {
		id = "new"
	}
	if !domain.ValidID(id) {
		return SlugAvailability{}, storage.ErrNotFound
	}
	if err := s.authority(ctx, user, project, id, "manage"); err != nil {
		return SlugAvailability{}, err
	}
	normalized, err := NormalizeSlug(slug)
	if err != nil || normalized == "" {
		return SlugAvailability{}, errors.New("enter a valid vanity URL name")
	}
	result := SlugAvailability{Slug: normalized, Available: true}
	row, err := s.Store.Get(ctx, "status-page-slugs", normalized)
	if errors.Is(err, storage.ErrNotFound) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	result.Available = false
	pageRow, err := s.Store.Get(ctx, Bucket(project), id)
	if err == nil {
		page, e := storage.Decode[Page](pageRow)
		if e != nil {
			return result, e
		}
		result.Available = ownsSlug(row, project, id, page.Token)
	} else if !errors.Is(err, storage.ErrNotFound) {
		return result, err
	}
	return result, nil
}
