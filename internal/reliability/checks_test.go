package reliability_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/reliability"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestIndividualChecksPageExactObservationsAndReauthorize(t *testing.T) {
	s, auth := fixture(t)
	at := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	scope := reliability.Scope{Project: "project", Suite: "suite", Test: "individual", Location: "local"}
	values := []reliability.Observation{}
	states := []string{"passed", "failed", "timed-out", "flaky", "skipped", "unknown"}
	expected := []string{}
	for i := range 132 {
		duration := float64(i) + .125
		v := reliability.Observation{Scope: scope, Run: fmt.Sprintf("run-%03d", i), At: at, Created: at.Add(-time.Minute), Started: at.Add(-time.Second), State: states[i%6], DurationMs: &duration, Censored: i%6 == 2, Source: json.RawMessage(`{}`)}
		if i == 130 {
			v.At = at.Add(-time.Hour)
		}
		if i == 131 {
			v.At = at.Add(time.Hour)
		}
		values = append(values, v)
		if i < 130 {
			expected = append(expected, v.Run)
		}
	}
	sort.Slice(expected, func(i, j int) bool {
		return reliability.Hash([]string{expected[i], scope.Test}) > reliability.Hash([]string{expected[j], scope.Test})
	})
	if err := s.Store.Transaction(t.Context(), func(_ *storage.Store, tx *sql.Tx) error {
		return reliability.Capture(t.Context(), tx, s.Store.Dialect, values)
	}); err != nil {
		t.Fatal(err)
	}
	q := reliability.Query{Scope: scope, Since: at, Until: at.Add(time.Hour)}
	read := func(cursor string) reliability.CheckPage {
		t.Helper()
		reader, err := auth.NewRunReadBatch(t.Context(), "reader")
		if err != nil {
			t.Fatal(err)
		}
		p, err := s.Checks(t.Context(), reader, q, "reader", cursor, 17)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	first := read("")
	if len(first.Items) != 17 || first.Cursor == "" {
		t.Fatal("missing page", len(first.Items))
	}
	reader, _ := auth.NewRunReadBatch(t.Context(), "reader")
	changed := q
	changed.Since = at.Add(-time.Second)
	if _, err := s.Checks(t.Context(), reader, changed, "reader", first.Cursor, 17); err == nil {
		t.Fatal("cursor crossed boundaries")
	}
	if _, err := s.Checks(t.Context(), reader, q, "other-reader", first.Cursor, 17); err == nil {
		t.Fatal("cursor crossed reader")
	}
	changed = q
	changed.Match = "changed-target"
	if _, err := s.Checks(t.Context(), reader, changed, "reader", first.Cursor, 17); err == nil {
		t.Fatal("cursor crossed configuration")
	}
	unmatched, err := s.Checks(t.Context(), reader, changed, "reader", "", 17)
	if err != nil || len(unmatched.Items) != 0 || unmatched.Cursor != "" {
		t.Fatal("old target leaked into current raw monitor history", err)
	}
	denied := expected[17]
	object := "project/run/" + denied
	if err := auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "reader", TeamScope: "project", ObjectScope: &object, PermissionName: "teststudio.runs.read", Effect: permissions.EffectDeny}); err != nil {
		t.Fatal(err)
	}
	got := append([]reliability.Check{}, first.Items...)
	cursor := first.Cursor
	for cursor != "" {
		p := read(cursor)
		got = append(got, p.Items...)
		cursor = p.Cursor
	}
	filtered := append(append([]string{}, expected[:17]...), expected[18:]...)
	if len(got) != 129 {
		t.Fatalf("lost raw checks beyond display limit: %d", len(got))
	}
	for i, c := range got {
		if c.Run != filtered[i] || !c.At.Equal(at) {
			t.Fatalf("unstable tied pagination at %d", i)
		}
		var index int
		fmt.Sscanf(c.Run, "run-%03d", &index)
		if c.State != states[index%6] || c.Duration == nil || *c.Duration != float64(index)+.125 || c.Censored != (index%6 == 2) {
			t.Fatal("raw outcome was transformed", c)
		}
	}
}

func TestIndividualChecksContinueAfterEmptyBoundedScan(t *testing.T) {
	s, auth := fixture(t)
	at := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	scope := reliability.Scope{Project: "project", Suite: "suite", Test: "filtered", Location: "local"}
	values := make([]reliability.Observation, 501)
	for i := range values {
		values[i] = reliability.Observation{Scope: scope, Run: fmt.Sprintf("filtered-%03d", i), At: at.Add(time.Duration(i) * time.Second), Created: at, State: "passed", Match: "old-target", Source: json.RawMessage(`{}`)}
	}
	if err := s.Store.Transaction(t.Context(), func(_ *storage.Store, tx *sql.Tx) error {
		return reliability.Capture(t.Context(), tx, s.Store.Dialect, values)
	}); err != nil {
		t.Fatal(err)
	}
	reader, err := auth.NewRunReadBatch(t.Context(), "reader")
	if err != nil {
		t.Fatal(err)
	}
	q := reliability.Query{Scope: scope, Since: at, Until: at.Add(time.Hour), Match: "current-target"}
	page, err := s.Checks(t.Context(), reader, q, "reader", "", 50)
	if err != nil || len(page.Items) != 0 || !page.Partial || page.Cursor == "" {
		t.Fatal("empty scan lost its continuation", page, err)
	}
	last, err := s.Checks(t.Context(), reader, q, "reader", page.Cursor, 50)
	if err != nil || len(last.Items) != 0 || last.Partial || last.Cursor != "" {
		t.Fatal("bounded continuation did not finish", last, err)
	}
}
