package sources_test

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/schedules"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/trends"
)

func TestRecordedWebsiteManualAndScheduledRunsKeepNormalHistoryAndEvidence(t *testing.T) {
	if os.Getenv("TESTSTUDIO_NATIVE_RECORDING_SOURCE") == "" || os.Getenv("TESTSTUDIO_BROWSER_GATEWAY_IMAGE") == "" {
		t.Skip("supply the newly captured local recording and trusted gateway image")
	}
	verifyBrowserSourceWorkflow(t, true, true, true)
}

func verifyScheduledWebsiteParity(t *testing.T, runner *runs.Service, manual runs.Run, root string) runs.Run {
	t.Helper()
	clock := time.Now().UTC()
	scheduler := &schedules.Service{Store: runner.Store, Auth: runner.Auth, Runs: runner, Now: func() time.Time { return clock }}
	definition := schedules.Definition{ID: "recorded-website", Name: "Recorded website", Enabled: true, Kind: "interval", IntervalSeconds: 5, Timezone: "UTC", Overlap: "skip", Request: manual.Request}
	if err := scheduler.Save(t.Context(), "admin", "alpha", definition, 0); err != nil {
		t.Fatal(err)
	}
	row, err := runner.Store.Get(t.Context(), schedules.Bucket("alpha"), definition.ID)
	if err != nil {
		t.Fatal(err)
	}
	configured, err := storage.Decode[schedules.Schedule](row)
	if err != nil || configured.NextAt == nil {
		t.Fatal("schedule has no due time", err)
	}
	clock = *configured.NextAt
	if _, err := scheduler.Tick(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
	row, err = runner.Store.Get(t.Context(), schedules.Bucket("alpha"), definition.ID)
	if err != nil {
		t.Fatal(err)
	}
	configured, _ = storage.Decode[schedules.Schedule](row)
	var scheduled runs.Run
	deadline := time.Now().Add(70 * time.Second)
	for time.Now().Before(deadline) {
		scheduled, _, err = runner.Get(t.Context(), "alpha", configured.LastRun)
		if err != nil {
			t.Fatal(err)
		}
		if runs.Terminal(scheduled.State) {
			break
		}
		time.Sleep(30 * time.Millisecond)
	}
	if scheduled.State != "completed" || scheduled.Outcome != "passed" || scheduled.EvidenceState != "complete" || scheduled.Counts.Total != 1 || scheduled.Counts.Passed != 1 || len(scheduled.Results) != 1 {
		t.Fatalf("scheduled website: state=%s outcome=%s results=%+v gaps=%v", scheduled.State, scheduled.Outcome, scheduled.Results, scheduled.EvidenceErrors)
	}
	if scheduled.Schedule == nil || scheduled.Trigger != "schedule" || !scheduled.Schedule.DueAt.Equal(clock) || scheduled.Release.ID != manual.Release.ID || !reflect.DeepEqual(scheduled.Tests, manual.Tests) || !reflect.DeepEqual(scheduled.Environment, manual.Environment) || !reflect.DeepEqual(scheduled.TestCapture, manual.TestCapture) {
		t.Fatal("scheduled website changes the reviewed execution snapshots")
	}
	artifacts := map[string][]evidence.Artifact{}
	for _, run := range []runs.Run{manual, scheduled} {
		if len(run.Results) != 1 || len(run.Results[0].Steps) == 0 || run.Results[0].Timing == nil || run.Results[0].Timing.Censored {
			t.Fatal("website loses logical steps or measured timing")
		}
		attempts, err := runner.Store.All(t.Context(), "attempts/"+run.ID, "alpha")
		if err != nil || len(attempts) != 1 {
			t.Fatal("website attempt count differs", err, len(attempts))
		}
		rows, err := runner.Store.All(t.Context(), evidence.Bucket(run.ID), "alpha")
		if err != nil {
			t.Fatal(err)
		}
		kinds := map[string]int{}
		for _, row := range rows {
			artifact, err := storage.Decode[evidence.Artifact](row)
			if err != nil {
				t.Fatal(err)
			}
			_, reader, err := runner.Evidence.Open(t.Context(), "alpha", run.ID, artifact.ID)
			if err != nil {
				t.Fatal(err)
			}
			data, readErr := io.ReadAll(reader)
			closeErr := reader.Close()
			checksum := sha256.Sum256(data)
			if readErr != nil || closeErr != nil || int64(len(data)) != artifact.Size || hex.EncodeToString(checksum[:]) != artifact.Checksum {
				t.Fatal("website evidence content differs", readErr, closeErr)
			}
			if artifact.Kind == "screenshot" {
				if _, err := png.DecodeConfig(bytes.NewReader(data)); err != nil {
					t.Fatal("website screenshot is unreadable", err)
				}
			}
			if artifact.Kind == "trace" {
				archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
				if err != nil || len(archive.File) == 0 {
					t.Fatal("website trace is unreadable", err)
				}
			}
			kinds[artifact.Kind]++
			artifacts[run.ID] = append(artifacts[run.ID], artifact)
		}
		if kinds["screenshot"] != 1 || kinds["trace"] != 1 || kinds["log"] == 0 {
			t.Fatal("website capture contract differs", kinds)
		}
	}
	history, err := runner.TestHistory(t.Context(), "admin", "alpha", "health", manual.Tests[0].ID, "", 10)
	if err != nil || len(history.Items) != 2 {
		t.Fatal("website history does not use normal logical results", err)
	}
	trend, err := (&trends.Service{Store: runner.Store, Auth: runner.Auth}).Read(t.Context(), "admin", "alpha", trends.Filter{Suite: "health", Since: time.Now().Add(-time.Hour), Until: time.Now().Add(time.Minute)}, "", 10)
	if err != nil || trend.Totals.Total != 2 || trend.Totals.Passed != 2 || len(trend.Points) != 2 {
		t.Fatal("website child steps inflate trends", err, trend.Totals)
	}
	data, err := json.MarshalIndent(map[string]any{"manual": manual, "scheduled": scheduled, "history": history, "trends": trend, "artifacts": artifacts}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "summary.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	return scheduled
}
