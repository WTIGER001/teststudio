package localauth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/wtiger001/teststudio/internal/storage"
)

type counter struct {
	Count int       `json:"count"`
	Until time.Time `json:"until"`
}

func (s *Service) slot(kind, value string, count uint32) string {
	h := hmac.New(sha256.New, s.key)
	h.Write([]byte(kind + "\x00" + value))
	return fmt.Sprintf("%s-%04x", kind, binary.BigEndian.Uint32(h.Sum(nil)[:4])%count)
}

func (s *Service) reserve(ctx context.Context, user, peer string) error {
	slots := []struct {
		id    string
		limit int
	}{{s.slot("account", user, 4096), 10}, {s.slot("peer", peer, 256), 100}}
	for range 16 {
		now := time.Now().UTC()
		changes := []storage.Change{}
		for _, slot := range slots {
			row, err := s.Store.Get(ctx, "local-auth-attempts", slot.id)
			if err != nil && !errors.Is(err, storage.ErrNotFound) {
				return err
			}
			value := counter{}
			if row.Revision > 0 {
				value, err = storage.Decode[counter](row)
				if err != nil {
					return err
				}
			}
			if !now.Before(value.Until) {
				value = counter{Until: now.Add(15 * time.Minute)}
			}
			if value.Count >= slot.limit {
				return ErrLimited
			}
			value.Count++
			c, err := storage.Put("local-auth-attempts", slot.id, "", row.Revision, value)
			if err != nil {
				return err
			}
			changes = append(changes, c)
		}
		if err := s.Store.Commit(ctx, changes...); errors.Is(err, storage.ErrConflict) {
			continue
		} else {
			return err
		}
	}
	return ErrLimited
}
