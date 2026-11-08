package transfer

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

func recordingRuntimeBucket(bucket string) bool {
	return bucket == "recording-worker-leases" || bucket == "recording-worker-pools"
}

func recordingWorkersDrained(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `SELECT bucket,payload FROM records WHERE bucket IN ('recording-worker-leases','recording-worker-pools')`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var bucket, payload string
		if err := rows.Scan(&bucket, &payload); err != nil {
			return err
		}
		var value struct {
			CleanupPending bool      `json:"cleanupPending"`
			Until          time.Time `json:"until"`
		}
		if json.Unmarshal([]byte(payload), &value) != nil {
			return ErrInvalidArchive
		}
		if value.CleanupPending || bucket == "recording-worker-pools" && time.Now().Before(value.Until) {
			return ErrBusy
		}
	}
	return rows.Err()
}
