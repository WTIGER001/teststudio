package transfer_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/adapters/artifacts/filesystem"
	permissionsqlite "github.com/wtiger001/teststudio/adapters/permissions/sqlite"
	"github.com/wtiger001/teststudio/internal/access"
	"github.com/wtiger001/teststudio/internal/artifacts"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/installstate"
	"github.com/wtiger001/teststudio/internal/localauth"
	"github.com/wtiger001/teststudio/internal/passwords"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
	"github.com/wtiger001/teststudio/internal/transfer"
)

type installation struct {
	db        *storage.Store
	path      string
	blobs     *filesystem.Store
	grant     access.GrantRecord
	run       runs.Run
	artifact  evidence.Artifact
	cookie    *http.Cookie
	highWater int64
}

func seedInstallation(t *testing.T) *installation {
	t.Helper()
	ctx := t.Context()
	root := t.TempDir()
	database := filepath.Join(root, "source.db")
	db, err := storage.Open(ctx, database)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	native := permissionsqlite.New(db)
	directory := &identity.Directory{Store: db}
	for _, id := range []string{"admin", "reader"} {
		if err := directory.SaveUser(ctx, identity.User{ID: id, Name: id}, 0); err != nil {
			t.Fatal(err)
		}
	}
	auth, err := authorization.New(ctx, native, directory)
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.Engine.AssignRole(ctx, permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "admin"}, "builtin.teststudio.installation-admin", nil); err != nil {
		t.Fatal(err)
	}
	projectService := &projects.Service{Store: db, Auth: auth}
	if err := projectService.SaveProject(ctx, "admin", projects.Project{ID: "project", Name: "Fresh transfer project"}, 0); err != nil {
		t.Fatal(err)
	}
	if err := projectService.SaveApplication(ctx, "admin", "project", projects.Application{ID: "app", Name: "Fresh application"}, 0); err != nil {
		t.Fatal(err)
	}
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("fresh local target")) }))
	t.Cleanup(target.Close)
	if err := projectService.SaveEnvironment(ctx, "admin", "project", projects.Environment{ID: "local", Name: "Local fixture", Application: "app", Stage: "local", Endpoints: map[string]string{"api": target.URL}}, 0); err != nil {
		t.Fatal(err)
	}
	if err := projectService.SaveSuite(ctx, "admin", "project", projects.Suite{ID: "health", Name: "Fresh checks", Applications: []string{"app"}, RequiresTarget: true}, 0); err != nil {
		t.Fatal(err)
	}
	catalogService := &catalog.Service{Store: db, Auth: auth}
	definition := catalog.Test{ID: "CHECK", Title: "Fresh transfer check", Type: "https", Runner: "http", State: "enabled", Application: "app", Category: "Availability", Safety: "safe", Rationale: "Reads only the new local fixture", TimeoutSeconds: 10, Check: &catalog.HTTPCheck{Endpoint: "api", Path: "/health", Method: "GET", StatusCodes: []int{200}}}
	if err := catalogService.SaveTest(ctx, "admin", "project", "health", definition, 0); err != nil {
		t.Fatal(err)
	}
	draft, err := catalogService.CreateDraft(ctx, "admin", "project", "health", catalog.Artifact{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := catalogService.Publish(ctx, "admin", "project", "health", draft.ID, "fresh-publication", 1, true); err != nil {
		t.Fatal(err)
	}
	files, err := filesystem.New(filepath.Join(root, "artifacts"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { files.Close() })
	evidenceService := &evidence.Service{Store: db, Blobs: files}
	runService := &runs.Service{Store: db, Auth: auth, Catalog: catalogService, Evidence: evidenceService, HTTP: &execution.HTTP{}, Location: "local", PollInterval: 5 * time.Millisecond}
	created, err := runService.Create(ctx, "admin", "project", runs.Request{Suite: "health", Environment: "local"}, "fresh-run", "manual")
	if err != nil {
		t.Fatal(err)
	}
	worker, cancel := context.WithCancel(ctx)
	done := runService.Start(worker)
	defer func() {
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("owned source worker did not stop")
		}
	}()
	var completed runs.Run
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		completed, _, err = runService.Get(ctx, "project", created.ID)
		if err != nil {
			t.Fatal(err)
		}
		if completed.State == "completed" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if completed.Outcome != "passed" || completed.State != "completed" {
		t.Fatal("fresh source run failed", completed.State, completed.Outcome)
	}
	cancel()
	<-done
	attachment, err := evidenceService.SaveBytes(ctx, evidence.Artifact{ID: "transfer-artifact", Project: "project", Run: completed.ID, Test: "CHECK", Attempt: 1, Kind: "attachment", Filename: "fresh-context.json", ContentType: "application/json"}, []byte(`{"new":"transfer evidence"}`), completed.Capture.MaxBytes)
	if err != nil {
		t.Fatal(err)
	}
	if err := evidenceService.SavePin(ctx, "admin", "project", completed.ID, true, nil, "Preserve the new migration evidence", 0); err != nil {
		t.Fatal(err)
	}
	if err := directory.SaveGroup(ctx, identity.Group{ID: "reviewers", Name: "Reviewers", Project: "project"}, 0); err != nil {
		t.Fatal(err)
	}
	if err := directory.AddMember(ctx, identity.Membership{UserID: "reader", GroupID: "reviewers", Source: "manual"}); err != nil {
		t.Fatal(err)
	}
	managed := &access.Service{Store: db, Auth: auth, LocalUser: "admin"}
	for _, policy := range []access.RolePolicy{
		{ID: "transfer-reader", Name: "Transfer reader", Project: "project", Rules: []access.RoleRule{{Permission: "teststudio.catalog.read", Effect: permissions.EffectAllow}}},
		{ID: "transfer-bundle", Name: "Transfer bundle", Project: "project", Inherits: []string{"transfer-reader"}},
	} {
		if _, err := managed.SaveRolePolicy(ctx, "admin", access.RolePolicyMutation{RolePolicy: policy, Reason: "Prepare fresh transfer access"}, 0); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := managed.CreateAssignment(ctx, "admin", "project", access.AssignmentDefinition{PrincipalKind: permissions.PrincipalGroup, PrincipalID: "reviewers", RoleID: "transfer-bundle", Reason: "Prepare group-bound access"}); err != nil {
		t.Fatal(err)
	}
	grant, err := managed.CreateGrant(ctx, "admin", "project", access.GrantDefinition{PrincipalKind: permissions.PrincipalUser, PrincipalID: "reader", Permission: "teststudio.catalog.read", Effect: permissions.EffectDeny, Object: "test/health/CHECK", Reason: "Retain exact denial through transfer"})
	if err != nil {
		t.Fatal(err)
	}
	if err := native.DisableBuiltInRole(ctx, "builtin.teststudio.viewer"); err != nil {
		t.Fatal(err)
	}
	if err := localauth.Configure(ctx, directory, true); err != nil {
		t.Fatal(err)
	}
	managed.LocalAccounts = true
	if _, err := managed.SetLocalPassword(ctx, "admin", "reader", "Fresh transfer passphrase 42!", "Preserve a new credential", 0); err != nil {
		t.Fatal(err)
	}
	var highWater int64
	for range 3 {
		highWater, err = db.Next(ctx, "permission-grants")
		if err != nil {
			t.Fatal(err)
		}
	}
	response := httptest.NewRecorder()
	sessionService := &sessions.Service{Store: db, Directory: directory, Origin: "http://127.0.0.1:8040"}
	if _, err := sessionService.Create(ctx, response, "admin", "local", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatal("source session cookie missing")
	}
	if _, err := db.Next(ctx, "fresh-sequence"); err != nil {
		t.Fatal(err)
	}
	return &installation{db: db, path: database, blobs: files, grant: grant, run: completed, artifact: attachment, cookie: cookies[0], highWater: highWater}
}

func metadata(t *testing.T, db *storage.Store) []transfer.Record {
	t.Helper()
	rows, err := db.DB.QueryContext(t.Context(), `SELECT bucket,id,project,revision,payload,updated_at FROM records ORDER BY bucket,id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	result := []transfer.Record{}
	for rows.Next() {
		var value transfer.Record
		if err := rows.Scan(&value.Bucket, &value.ID, &value.Project, &value.Revision, &value.Payload, &value.UpdatedAt); err != nil {
			t.Fatal(err)
		}
		if !installstate.IsNativeBucket(value.Bucket) {
			result = append(result, value)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return result
}

func verifyInstallation(t *testing.T, source *installation, db *storage.Store, native permissions.PermissionStore) {
	t.Helper()
	ctx := t.Context()
	if !reflect.DeepEqual(metadata(t, source.db), metadata(t, db)) {
		t.Fatal("transfer changed identifiers, revisions, raw payloads, or timestamps")
	}
	directory := &identity.Directory{Store: db}
	auth, err := authorization.New(ctx, native, directory)
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.Check(ctx, "reader", "project", "", "catalog.read"); err != nil {
		t.Fatal("inherited group access was lost", err)
	}
	if err := auth.Check(ctx, "reader", "project", "test/health/CHECK", "catalog.read"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("exact deny was lost", err)
	}
	if err := auth.Check(ctx, "reader", "other-project", "", "catalog.read"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("scope binding crossed projects", err)
	}
	roles, err := native.DisabledBuiltInRoles(ctx)
	if err != nil || len(roles) != 1 || roles[0] != "builtin.teststudio.viewer" {
		t.Fatal("disabled built-in override changed", roles, err)
	}
	stored, err := db.Get(ctx, "runs", source.run.ID)
	if err != nil {
		t.Fatal(err)
	}
	run, err := storage.Decode[runs.Run](stored)
	if err != nil || run.Release.ID != source.run.Release.ID || run.Results[0].State != "passed" {
		t.Fatal("run snapshot or release changed", err)
	}
	evidenceService := &evidence.Service{Store: db, Blobs: source.blobs}
	artifact, reader, err := evidenceService.Open(ctx, "project", source.run.ID, source.artifact.ID)
	if err != nil {
		t.Fatal("artifact reference no longer resolves", err)
	}
	data, err := io.ReadAll(reader)
	reader.Close()
	if err != nil || string(data) != `{"new":"transfer evidence"}` || artifact.Checksum != source.artifact.Checksum {
		t.Fatal("artifact bytes or checksum changed", err)
	}
	credentialRow, err := db.Get(ctx, "local-credentials", "reader")
	if err != nil {
		t.Fatal(err)
	}
	credential, err := storage.Decode[identity.LocalCredential](credentialRow)
	if err != nil {
		t.Fatal(err)
	}
	if valid, err := passwords.Verify(ctx, credential.Hash, "Fresh transfer passphrase 42!"); err != nil || !valid {
		t.Fatal("credential hash did not survive", err)
	}
	request := httptest.NewRequest("GET", "http://127.0.0.1:8040/private", nil)
	request.AddCookie(source.cookie)
	sessionService := &sessions.Service{Store: db, Directory: directory, Origin: "http://127.0.0.1:8040"}
	session, err := sessionService.Read(ctx, request)
	if err != nil || session.UserID != "admin" {
		t.Fatal("server-side session did not survive", err)
	}
	managed := &access.Service{Store: db, Auth: auth, LocalUser: "admin"}
	if _, err := managed.UpdateGrant(ctx, "admin", "project", source.grant.ID, access.GrantUpdate{Reason: "Verify restored native grant identity", ExpiresAt: func() *time.Time { value := time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond); return &value }()}, source.grant.Revision); err != nil {
		t.Fatal("managed native grant reference changed", err)
	}
	grantRow, err := db.Get(ctx, "access-grants/project", source.grant.ID)
	if err != nil {
		t.Fatal(err)
	}
	var linked struct {
		NativeID int64 `json:"nativeId"`
	}
	if err := json.Unmarshal(grantRow.Data, &linked); err != nil || linked.NativeID <= source.highWater {
		t.Fatal("native sequence reused a previous grant identifier", linked.NativeID, err)
	}
	if next, err := db.Next(ctx, "fresh-sequence"); err != nil || next != 2 {
		t.Fatal("counter high-water mark changed", next, err)
	}
}

func TestSQLiteInstallationArchivePreservesAccessHistoryAndArtifactReferences(t *testing.T) {
	source := seedInstallation(t)
	archive := filepath.Join(t.TempDir(), "installation.ndjson")
	readonly, err := transfer.OpenSource(t.Context(), source.path)
	if err != nil {
		t.Fatal(err)
	}
	defer readonly.Close()
	if _, err := readonly.DB.ExecContext(t.Context(), `DELETE FROM records`); err == nil {
		t.Fatal("export source connection allowed a write")
	}
	stats, err := transfer.ExportFile(t.Context(), readonly, archive)
	if err != nil || stats.Records == 0 || stats.Permissions["grant"] == 0 {
		t.Fatal("export failed", stats, err)
	}
	info, err := os.Stat(archive)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("archive is not private", err)
	}
	before, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transfer.ExportFile(t.Context(), readonly, archive); err == nil {
		t.Fatal("existing archive was overwritten")
	}
	after, _ := os.ReadFile(archive)
	if !bytes.Equal(before, after) {
		t.Fatal("existing archive changed")
	}
	destination, err := storage.Open(t.Context(), filepath.Join(t.TempDir(), "restored.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer destination.Close()
	restored, err := transfer.Import(t.Context(), destination, bytes.NewReader(before))
	if err != nil || !reflect.DeepEqual(stats, restored) {
		t.Fatal("import failed", restored, err)
	}
	verifyInstallation(t, source, destination, permissionsqlite.New(destination))
	if stats.Reliability == 0 {
		t.Fatal("reliability tables were omitted from the archive")
	}
	for _, table := range []string{"reliability_observations", "reliability_series", "reliability_authorities", "reliability_policies", "reliability_jobs", "reliability_summaries"} {
		var before, after int
		if err := source.db.DB.QueryRowContext(t.Context(), "SELECT count(*) FROM "+table).Scan(&before); err != nil {
			t.Fatal(err)
		}
		if err := destination.DB.QueryRowContext(t.Context(), "SELECT count(*) FROM "+table).Scan(&after); err != nil {
			t.Fatal(err)
		}
		if before != after {
			t.Fatal("reliability history changed during transfer", table, before, after)
		}
	}
	if _, err := transfer.Import(t.Context(), destination, bytes.NewReader(before)); !errors.Is(err, transfer.ErrNotEmpty) {
		t.Fatal("nonempty destination accepted", err)
	}
	key := "project/" + source.run.ID + "/" + source.artifact.ID + "/" + source.artifact.Checksum
	if err := source.blobs.Delete(t.Context(), key); err != nil {
		t.Fatal(err)
	}
	missing, reader, err := (&evidence.Service{Store: destination, Blobs: source.blobs}).Open(t.Context(), "project", source.run.ID, source.artifact.ID)
	if reader != nil {
		reader.Close()
	}
	if !errors.Is(err, artifacts.ErrNotFound) || missing.State != "missing" {
		t.Fatal("restoration hid a missing artifact file", missing.State, err)
	}

}

func TestSQLiteToPostgreSQLInstallationTransferPreservesNativeGrantsAndBindings(t *testing.T) {
	if os.Getenv("TESTSTUDIO_TEST_DATABASE") == "" {
		t.Skip("use the local PostgreSQL helper")
	}
	source := seedInstallation(t)
	var archive bytes.Buffer
	if _, err := transfer.Export(t.Context(), source.db, &archive); err != nil {
		t.Fatal(err)
	}
	destination, err := testkit.Open(t, "")
	if err != nil {
		t.Fatal(err)
	}
	defer destination.Close()
	native := testkit.PermissionStore(t, destination)
	corrupt := bytes.Replace(archive.Bytes(), []byte("Fresh transfer project"), []byte("Changed transfer project"), 1)
	if _, err := transfer.Import(t.Context(), destination, bytes.NewReader(corrupt)); err == nil {
		t.Fatal("corrupt archive committed PostgreSQL data")
	}
	if roles, err := native.RoleDefinitions(t.Context()); err != nil || len(roles) != 0 {
		t.Fatal("failed PostgreSQL import left native roles", err)
	}
	if len(metadata(t, destination)) != 0 {
		t.Fatal("failed PostgreSQL import left metadata")
	}
	if _, err := transfer.Import(t.Context(), destination, bytes.NewReader(archive.Bytes())); err != nil {
		t.Fatal(err)
	}
	verifyInstallation(t, source, destination, native)
	var reverse bytes.Buffer
	if _, err := transfer.Export(t.Context(), destination, &reverse); err != nil {
		t.Fatal("PostgreSQL export failed", err)
	}
	local, err := storage.Open(t.Context(), filepath.Join(t.TempDir(), "roundtrip.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer local.Close()
	if _, err := transfer.Import(t.Context(), local, bytes.NewReader(reverse.Bytes())); err != nil {
		t.Fatal("PostgreSQL-to-SQLite restoration failed", err)
	}
	if !reflect.DeepEqual(metadata(t, destination), metadata(t, local)) {
		t.Fatal("reverse transfer changed public metadata")
	}
	roundtripAuth, err := authorization.New(t.Context(), permissionsqlite.New(local), &identity.Directory{Store: local})
	if err != nil {
		t.Fatal(err)
	}
	if err := roundtripAuth.Check(t.Context(), "reader", "project", "", "catalog.read"); err != nil {
		t.Fatal("reverse transfer lost inherited authority", err)
	}
	if err := roundtripAuth.Check(t.Context(), "reader", "project", "test/health/CHECK", "catalog.read"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("reverse transfer lost the exact denial", err)
	}
	user, err := (&identity.Directory{Store: local}).User(t.Context(), "reader")
	if err != nil || user.ID != "reader" {
		t.Fatal("roundtrip lost identity", err)
	}
}

func TestCorruptOrTruncatedInstallationArchiveRollsBackDestination(t *testing.T) {
	source := seedInstallation(t)
	var complete bytes.Buffer
	if _, err := transfer.Export(t.Context(), source.db, &complete); err != nil {
		t.Fatal(err)
	}
	corrupt := bytes.Replace(complete.Bytes(), []byte("Fresh transfer project"), []byte("Altered transfer project"), 1)
	lines := bytes.Split(complete.Bytes(), []byte{'\n'})
	truncated := bytes.Join(lines[:len(lines)-2], []byte{'\n'})
	for _, data := range [][]byte{corrupt, truncated} {
		destination, err := storage.Open(t.Context(), filepath.Join(t.TempDir(), "rejected.db"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := transfer.Import(t.Context(), destination, bytes.NewReader(data)); err == nil {
			t.Fatal("damaged archive was accepted")
		}
		var count int
		if err := destination.DB.QueryRowContext(t.Context(), `SELECT count(*) FROM records`).Scan(&count); err != nil || count != 0 {
			t.Fatal("failed import left partial data", count, err)
		}
		destination.Close()
	}
}

func TestInstallationExportWaitsForActiveWorkAndPublishesNoPartialArchive(t *testing.T) {
	for _, bucket := range []string{"execution-work", "worker-exchanges"} {
		t.Run(bucket, func(t *testing.T) {
			source, err := testkit.Open(t, filepath.Join(t.TempDir(), "busy.db"))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { source.Close() })
			row, err := storage.Put(bucket, "pending", "project", 0, map[string]string{"runId": "pending-run", "executionId": "pending"})
			if err != nil {
				t.Fatal(err)
			}
			if err := source.Commit(t.Context(), row); err != nil {
				t.Fatal(err)
			}
			output := filepath.Join(t.TempDir(), "busy.ndjson")
			if _, err := transfer.ExportFile(t.Context(), source, output); !errors.Is(err, transfer.ErrBusy) {
				t.Fatal("active worker state entered a cutover archive", err)
			}
			if _, err := os.Stat(output); !os.IsNotExist(err) {
				t.Fatal("failed export published an archive")
			}
			files, err := os.ReadDir(filepath.Dir(output))
			if err != nil || len(files) != 0 {
				t.Fatal("failed export left an owned temporary file", err)
			}
		})
	}
}
