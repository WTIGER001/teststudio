package evidence_test

import (
	"fmt"
	"strings"
	"testing"
)

func TestLogPagesBindTheirSelectionAndTailKeepsNewestBytes(t *testing.T) {
	s, _ := fixture(t)
	for i := 1; i <= 20; i++ {
		if _, err := s.Log(t.Context(), "project", "run", "worker", "stdout", "TEST-1", 1, int64(i), fmt.Sprintf("%02d", i)+strings.Repeat("x", 65534), 2<<20); err != nil {
			t.Fatal(err)
		}
	}
	first, err := s.Logs(t.Context(), "project", "run", "stdout", "TEST-1", "", false, 3)
	if err != nil || len(first.Segments) != 3 || first.Cursor == "" || !strings.HasPrefix(first.Segments[0].Text, "01") {
		t.Fatalf("first page: %+v %v", len(first.Segments), err)
	}
	next, err := s.Logs(t.Context(), "project", "run", "stdout", "TEST-1", first.Cursor, false, 3)
	if err != nil || len(next.Segments) != 3 || !strings.HasPrefix(next.Segments[0].Text, "04") {
		t.Fatalf("next page: %+v %v", len(next.Segments), err)
	}
	if _, err := s.Logs(t.Context(), "project", "run", "stderr", "TEST-1", first.Cursor, false, 3); err == nil {
		t.Fatal("cursor accepted another stream")
	}
	tail, err := s.Logs(t.Context(), "project", "run", "stdout", "TEST-1", "", true, 20)
	if err != nil || !tail.Truncated || len(tail.Segments) != 16 || !strings.HasPrefix(tail.Segments[0].Text, "05") || !strings.HasPrefix(tail.Segments[15].Text, "20") {
		t.Fatalf("bounded tail: %d %v", len(tail.Segments), err)
	}
	if _, err := s.Log(t.Context(), "project", "run", "worker", "stdout", "TEST-1", 1, 21, "new", 2<<20); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Logs(t.Context(), "project", "run", "stdout", "TEST-1", first.Cursor, false, 3); err == nil {
		t.Fatal("cursor accepted a changed log snapshot")
	}
}
