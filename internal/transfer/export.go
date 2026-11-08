package transfer

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"time"

	permissionpg "github.com/wtiger001/teststudio/adapters/permissions/postgres"
	permissionsqlite "github.com/wtiger001/teststudio/adapters/permissions/sqlite"
	"github.com/wtiger001/teststudio/internal/installstate"
	"github.com/wtiger001/teststudio/internal/storage"
)

func Export(ctx context.Context, source *storage.Store, output io.Writer) (Stats, error) {
	stats := Stats{Permissions: map[string]int64{}}
	err := source.ReadTransaction(ctx, func(db *storage.Store, tx *sql.Tx) error {
		var busy bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM records WHERE bucket IN ('run-leases','execution-work','worker-exchanges') OR bucket LIKE 'preparation-leases/%')`).Scan(&busy); err != nil {
			return err
		}
		if busy {
			return ErrBusy
		}
		if err := recordingWorkersDrained(ctx, tx); err != nil {
			return err
		}
		recordings, err := tx.QueryContext(ctx, `SELECT payload FROM records WHERE bucket='browser-recordings'`)
		if err != nil {
			return err
		}
		for recordings.Next() {
			var data string
			var recording struct {
				State string `json:"state"`
			}
			if err := recordings.Scan(&data); err != nil {
				recordings.Close()
				return err
			}
			if json.Unmarshal([]byte(data), &recording) != nil {
				recordings.Close()
				return ErrInvalidArchive
			}
			if recording.State == "starting" || recording.State == "recording" || recording.State == "pairing" || recording.State == "stopping" {
				recordings.Close()
				return ErrBusy
			}
		}
		err = recordings.Err()
		recordings.Close()
		if err != nil {
			return err
		}
		var version int
		if err := tx.QueryRowContext(ctx, `SELECT coalesce(max(version),0) FROM schema_migrations`).Scan(&version); err != nil {
			return err
		}
		if version != 1 && version != 2 {
			return ErrInvalidArchive
		}
		var sequence int64
		if db.Dialect == "postgres" {
			value, err := permissionpg.GrantSequence(ctx, tx)
			if err != nil {
				return err
			}
			sequence = value
		} else {
			err := tx.QueryRowContext(ctx, `SELECT value FROM counters WHERE name='permission-grants'`).Scan(&sequence)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}
		}
		hash := sha256.New()
		write := func(kind string, value any, digest bool) error {
			data, err := json.Marshal(value)
			if err != nil {
				return err
			}
			wire, err := json.Marshal(Entry{Kind: kind, Data: data})
			if err != nil {
				return err
			}
			if len(wire) > MaxEntryBytes {
				return ErrInvalidArchive
			}
			wire = append(wire, '\n')
			writer := output
			if digest {
				writer = io.MultiWriter(output, hash)
			}
			_, err = writer.Write(wire)
			return err
		}
		if err := write("header", Header{Schema: Schema, PermissionVersion: PermissionVersion, MetadataVersion: version, Source: db.Dialect, CreatedAt: time.Now().UTC(), GrantSequence: sequence}, true); err != nil {
			return err
		}
		emit := func(item installstate.Permission) error {
			var raw any = json.RawMessage(item.Data)
			if err := write(item.Kind, raw, true); err != nil {
				return err
			}
			stats.Permissions[item.Kind]++
			return nil
		}
		if db.Dialect == "postgres" {
			if err := permissionpg.ExportState(ctx, tx, emit); err != nil {
				return err
			}
		} else {
			if err := permissionsqlite.ExportState(ctx, db, emit); err != nil {
				return err
			}
		}
		rows, err := tx.QueryContext(ctx, `SELECT bucket,id,project,revision,payload,updated_at FROM records ORDER BY bucket,id`)
		if err != nil {
			return err
		}
		for rows.Next() {
			var row Record
			if err := rows.Scan(&row.Bucket, &row.ID, &row.Project, &row.Revision, &row.Payload, &row.UpdatedAt); err != nil {
				rows.Close()
				return err
			}
			if installstate.IsNativeBucket(row.Bucket) || recordingRuntimeBucket(row.Bucket) {
				continue
			}
			if err := write("record", row, true); err != nil {
				rows.Close()
				return err
			}
			stats.Records++
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		rows, err = tx.QueryContext(ctx, `SELECT name,value FROM counters ORDER BY name`)
		if err != nil {
			return err
		}
		for rows.Next() {
			var row Counter
			if err := rows.Scan(&row.Name, &row.Value); err != nil {
				rows.Close()
				return err
			}
			if err := write("counter", row, true); err != nil {
				rows.Close()
				return err
			}
			stats.Counters++
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if version >= 2 {
			if err := exportReliability(ctx, tx, func(value reliabilityEntry) error {
				if err := write("reliability", value, true); err != nil {
					return err
				}
				stats.Reliability++
				return nil
			}); err != nil {
				return err
			}
		}
		return write("footer", footer{Stats: stats, Checksum: hex.EncodeToString(hash.Sum(nil))}, false)
	})
	return stats, err
}
