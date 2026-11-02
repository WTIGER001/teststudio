package runs_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestRunListBoundsScanningAndBindsFiltersToCursor(t *testing.T) {
	s, _, _ := setup(t, nil)
	for batch := 0; batch < 6; batch++ {
		changes := []storage.Change{}
		for i := batch * 100; i < min((batch+1)*100, 505); i++ {
			v := runs.Run{ID: fmt.Sprintf("run-%04d", i), Project: "project", State: "completed", Outcome: "passed", CreatedAt: time.Now().UTC(), Request: runs.Request{Suite: "health", Environment: "local"}, Environment: &projects.Environment{ID: "local", Name: "Original local name", Application: "app", Stage: "local"}}
			if i == 0 {
				v.State = "incomplete"
			}
			change, err := storage.Put("runs", v.ID, v.Project, 0, v)
			if err != nil {
				t.Fatal(err)
			}
			changes = append(changes, change)
		}
		if err := s.Store.Commit(t.Context(), changes...); err != nil {
			t.Fatal(err)
		}
	}
	f := runs.Filter{State: "incomplete"}
	page, err := s.List(t.Context(), "admin", "project", f, "", 99)
	if err != nil || page.Scanned != 500 || !page.Partial || page.Cursor == "" || len(page.Items) != 0 {
		t.Fatalf("bounded scan: %+v %v", page, err)
	}
	next, err := s.List(t.Context(), "admin", "project", f, page.Cursor, 99)
	if err != nil || next.Scanned != 5 || len(next.Items) != 1 || next.Cursor != "" {
		t.Fatalf("next scan: %+v %v", next, err)
	}
	var slim map[string]json.RawMessage
	if err := json.Unmarshal(next.Items[0].Data, &slim); err != nil {
		t.Fatal(err)
	}
	if _, ok := slim["tests"]; ok {
		t.Fatal("summary includes full tests")
	}
	if _, err := s.List(t.Context(), "admin", "project", runs.Filter{State: "completed"}, page.Cursor, 99); err == nil {
		t.Fatal("cursor accepted changed filters")
	}
	ordered, err := s.List(t.Context(), "admin", "project", runs.Filter{Application: "app", Stage: "local"}, "", 2)
	if err != nil || len(ordered.Items) != 2 || ordered.Items[0].ID != "run-0504" || ordered.Items[1].ID != "run-0503" {
		t.Fatalf("newest order: %+v %v", ordered, err)
	}
}
