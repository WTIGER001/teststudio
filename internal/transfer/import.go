package transfer

import (
	"bufio"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	permissionpg "github.com/wtiger001/teststudio/adapters/permissions/postgres"
	permissionsqlite "github.com/wtiger001/teststudio/adapters/permissions/sqlite"
	"github.com/wtiger001/teststudio/internal/installstate"
	"github.com/wtiger001/teststudio/internal/storage"
)

func Import(ctx context.Context, destination *storage.Store, input io.Reader) (Stats, error) {
	stats := Stats{Permissions: map[string]int64{}}
	err := destination.Transaction(ctx, func(db *storage.Store, tx *sql.Tx) error {
		if err := emptyDestination(ctx, db, tx); err != nil {
			return err
		}
		scanner := bufio.NewScanner(input)
		scanner.Buffer(make([]byte, 64<<10), MaxEntryBytes+1)
		hash := sha256.New()
		header := Header{}
		seenHeader, seenFooter := false, false
		phase := 0
		sequence := int64(0)
		for scanner.Scan() {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			line := scanner.Bytes()
			item, err := installstate.Decode[Entry](line)
			if err != nil {
				return ErrInvalidArchive
			}
			if seenFooter {
				return ErrInvalidArchive
			}
			if !seenHeader {
				if item.Kind != "header" {
					return ErrInvalidArchive
				}
				header, err = installstate.Decode[Header](item.Data)
				if err != nil || header.Schema != Schema || header.PermissionVersion != PermissionVersion || (header.MetadataVersion != 1 && header.MetadataVersion != 2) || (header.Source != "sqlite" && header.Source != "postgres") || header.CreatedAt.IsZero() || header.GrantSequence < 0 {
					return ErrInvalidArchive
				}
				sequence = header.GrantSequence
				seenHeader = true
			} else if item.Kind == "footer" {
				end, err := installstate.Decode[footer](item.Data)
				if err != nil || end.Checksum != hex.EncodeToString(hash.Sum(nil)) || !reflect.DeepEqual(stats, end.Stats) {
					return ErrInvalidArchive
				}
				seenFooter = true
				continue
			} else {
				order := entryOrder(item.Kind)
				if order == 0 || order < phase {
					return ErrInvalidArchive
				}
				phase = order
				switch item.Kind {
				case "reliability":
					if header.MetadataVersion < 2 {
						return ErrInvalidArchive
					}
					row, err := installstate.Decode[reliabilityEntry](item.Data)
					if err != nil {
						return ErrInvalidArchive
					}
					if err = importReliability(ctx, tx, row); err != nil {
						return err
					}
					stats.Reliability++
				case "record":
					row, err := installstate.Decode[Record](item.Data)
					if err != nil || !validRecord(row) {
						return ErrInvalidArchive
					}
					if _, err := tx.ExecContext(ctx, `INSERT INTO records (bucket,id,project,revision,payload,updated_at) VALUES ($1,$2,$3,$4,$5,$6)`, row.Bucket, row.ID, row.Project, row.Revision, row.Payload, row.UpdatedAt); err != nil {
						return errors.New("installation record restoration failed")
					}
					stats.Records++
				case "counter":
					row, err := installstate.Decode[Counter](item.Data)
					if err != nil || row.Name == "" || row.Value < 0 || strings.ContainsRune(row.Name, 0) {
						return ErrInvalidArchive
					}
					if _, err := tx.ExecContext(ctx, `INSERT INTO counters (name,value) VALUES ($1,$2)`, row.Name, row.Value); err != nil {
						return errors.New("installation counter restoration failed")
					}
					stats.Counters++
					if row.Name == "permission-grants" {
						sequence = max(sequence, row.Value)
					}
				default:
					native := installstate.Permission{Kind: item.Kind, Data: item.Data}
					if db.Dialect == "postgres" {
						err = permissionpg.ImportState(ctx, tx, native)
					} else {
						err = permissionsqlite.ImportState(ctx, db, native)
					}
					if err != nil {
						return errors.New("native permission restoration failed")
					}
					stats.Permissions[item.Kind]++
					if item.Kind == "grant" {
						grant, err := installstate.Decode[permissions.Grant](item.Data)
						if err != nil {
							return ErrInvalidArchive
						}
						sequence = max(sequence, grant.ID)
					}
				}
			}
			_, _ = hash.Write(line)
			_, _ = hash.Write([]byte{'\n'})
		}
		if err := scanner.Err(); err != nil {
			return ErrInvalidArchive
		}
		if !seenHeader || !seenFooter {
			return ErrInvalidArchive
		}
		if db.Dialect == "postgres" {
			return permissionpg.ResetGrantSequence(ctx, tx, sequence)
		}
		if sequence > 0 {
			_, err := tx.ExecContext(ctx, `INSERT INTO counters (name,value) VALUES ('permission-grants',$1) ON CONFLICT(name) DO UPDATE SET value=max(counters.value,excluded.value)`, sequence)
			return err
		}
		return nil
	})
	return stats, err
}

func entryOrder(kind string) int {
	switch kind {
	case "role":
		return 1
	case "grant":
		return 2
	case "edge":
		return 3
	case "assignment":
		return 4
	case "disabled":
		return 5
	case "record":
		return 6
	case "counter":
		return 7
	case "reliability":
		return 8
	}
	return 0
}

func validRecord(row Record) bool {
	if row.Bucket == "" || row.ID == "" || row.Revision < 1 || !json.Valid([]byte(row.Payload)) || installstate.IsNativeBucket(row.Bucket) || strings.ContainsRune(row.Bucket+row.ID+row.Project+row.Payload, 0) {
		return false
	}
	_, err := time.Parse(time.RFC3339Nano, row.UpdatedAt)
	return err == nil
}

func emptyDestination(ctx context.Context, db *storage.Store, tx *sql.Tx) error {
	if db.Dialect == "postgres" {
		if _, err := tx.ExecContext(ctx, `LOCK TABLE records,counters,reliability_series,reliability_authorities,reliability_observations,reliability_jobs,reliability_policies,reliability_summaries,teststudio_permissions.roles,teststudio_permissions.permission_grants,teststudio_permissions.principal_roles,teststudio_permissions.role_inheritance,teststudio_permissions.role_closure,teststudio_permissions.disabled_builtin_roles IN ACCESS EXCLUSIVE MODE`); err != nil {
			return errors.New("destination permission schema is unavailable")
		}
		empty, err := permissionpg.EmptyState(ctx, tx)
		if err != nil {
			return err
		}
		if !empty {
			return ErrNotEmpty
		}
	}
	var version int
	if err := tx.QueryRowContext(ctx, `SELECT coalesce(max(version),0) FROM schema_migrations`).Scan(&version); err != nil {
		return err
	}
	if version != 2 {
		return ErrInvalidArchive
	}
	var present bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM records) OR EXISTS(SELECT 1 FROM counters)`).Scan(&present); err != nil {
		return err
	}
	if present {
		return ErrNotEmpty
	}
	for _, table := range reliabilityTables {
		if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM "+table.name+")").Scan(&present); err != nil {
			return err
		}
		if present {
			return ErrNotEmpty
		}
	}
	return nil
}
