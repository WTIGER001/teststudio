package evidence_test

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/wtiger001/teststudio/internal/evidence"
)

func TestPrivateTraceValidationBoundsArchiveEntriesAndExpansion(t *testing.T) {
	for _, scenario := range []string{"valid", "traversal", "oversized", "not-trace"} {
		t.Run(scenario, func(t *testing.T) {
			s, _ := fixture(t)
			var data bytes.Buffer
			writer := zip.NewWriter(&data)
			name := "trace.trace"
			if scenario == "traversal" {
				name = "../trace.trace"
			}
			if scenario == "not-trace" {
				name = "readme.txt"
			}
			entry, err := writer.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			_, _ = entry.Write([]byte("{}\n"))
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			body := data.Bytes()
			if scenario == "oversized" {
				offset := bytes.Index(body, []byte{'P', 'K', 1, 2})
				if offset < 0 {
					t.Fatal("central directory missing")
				}
				binary.LittleEndian.PutUint32(body[offset+24:], 257<<20)
			}
			a, err := s.SaveBytes(t.Context(), evidence.Artifact{Project: "project", Run: "run", Kind: "trace", Filename: "trace.zip", ContentType: "application/zip"}, body, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			err = s.ValidateTrace(t.Context(), "project", "run", a.ID)
			if (err == nil) != (scenario == "valid") {
				t.Fatalf("%s: %v", scenario, err)
			}
		})
	}
}
