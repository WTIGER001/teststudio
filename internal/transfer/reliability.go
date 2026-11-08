package transfer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type reliabilityEntry struct {
	Table  string   `json:"table"`
	Values []string `json:"values"`
}

var reliabilityTables = []struct {
	name    string
	columns []string
}{
	{"reliability_series", []string{"id", "project", "payload", "first_at", "last_at"}},
	{"reliability_authorities", []string{"series", "source"}},
	{"reliability_observations", []string{"id", "series", "at", "created_at", "payload"}},
	{"reliability_policies", []string{"series", "max_age"}},
	{"reliability_jobs", []string{"series", "since_at", "until_at"}},
	{"reliability_summaries", []string{"series", "max_age", "resolution", "start_at", "payload", "compact"}},
}

func exportReliability(ctx context.Context, tx *sql.Tx, emit func(reliabilityEntry) error) error {
	for _, table := range reliabilityTables {
		columns := strings.Join(table.columns, ",")
		rows, err := tx.QueryContext(ctx, "SELECT "+columns+" FROM "+table.name+" ORDER BY "+columns)
		if err != nil {
			return err
		}
		for rows.Next() {
			values := make([]string, len(table.columns))
			targets := make([]any, len(values))
			for i := range targets {
				targets[i] = &values[i]
			}
			if err = rows.Scan(targets...); err != nil {
				rows.Close()
				return err
			}
			if err = emit(reliabilityEntry{Table: table.name, Values: values}); err != nil {
				rows.Close()
				return err
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func importReliability(ctx context.Context, tx *sql.Tx, row reliabilityEntry) error {
	for _, table := range reliabilityTables {
		if table.name != row.Table {
			continue
		}
		if len(row.Values) != len(table.columns) {
			return ErrInvalidArchive
		}
		arguments := make([]any, len(row.Values))
		parameters := make([]string, len(row.Values))
		for i, value := range row.Values {
			if value == "" || strings.ContainsRune(value, 0) {
				return ErrInvalidArchive
			}
			column := table.columns[i]
			if (column == "payload" || column == "source" || column == "compact") && !json.Valid([]byte(value)) {
				return ErrInvalidArchive
			}
			if column == "max_age" {
				v, err := strconv.Atoi(value)
				if err != nil || v < 0 || v > 86400 {
					return ErrInvalidArchive
				}
			}
			if strings.HasSuffix(column, "_at") || column == "at" {
				if _, err := time.Parse(time.RFC3339Nano, value); err != nil {
					return ErrInvalidArchive
				}
			}
			if column == "resolution" && value != "hour" && value != "day" {
				return ErrInvalidArchive
			}
			arguments[i] = value
			parameters[i] = fmt.Sprintf("$%d", i+1)
		}
		_, err := tx.ExecContext(ctx, "INSERT INTO "+table.name+" ("+strings.Join(table.columns, ",")+") VALUES ("+strings.Join(parameters, ",")+")", arguments...)
		if err != nil {
			return ErrInvalidArchive
		}
		return nil
	}
	return ErrInvalidArchive
}
