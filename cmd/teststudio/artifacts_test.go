package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/adapters/artifacts/filesystem"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/localauth"
	"github.com/wtiger001/teststudio/internal/passwords"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func TestArtifactRolePreservesPasswordModeAndRechecksExistingGrants(t *testing.T) {
	root := t.TempDir()
	db, err := testkit.Open(t, filepath.Join(root, "artifact-role.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	directory := &identity.Directory{Store: db}
	if err := directory.SaveUser(t.Context(), identity.User{ID: "owner", Name: "Owner"}, 0); err != nil {
		t.Fatal(err)
	}
	auth, err := authorization.New(t.Context(), testkit.PermissionStore(t, db), directory)
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "owner"}, "builtin.teststudio.installation-admin", nil); err != nil {
		t.Fatal(err)
	}
	if err := localauth.Configure(t.Context(), directory, true); err != nil {
		t.Fatal(err)
	}
	hash, err := passwords.Hash(t.Context(), "Dedicated artifact access 2026!")
	if err != nil {
		t.Fatal(err)
	}
	credential, err := storage.Put("local-credentials", "owner", "", 0, identity.LocalCredential{UserID: "owner", Hash: hash, CreatedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Commit(t.Context(), credential); err != nil {
		t.Fatal(err)
	}
	credentialRow, err := db.Get(t.Context(), "local-credentials", "owner")
	if err != nil {
		t.Fatal(err)
	}
	sessionService := &sessions.Service{Store: db, Directory: directory, LocalAccounts: true}
	session, _, err := sessionService.CreatePassword(t.Context(), "owner", credentialRow.Revision, "")
	if err != nil {
		t.Fatal(err)
	}
	blobs, err := filesystem.New(filepath.Join(root, "evidence"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { blobs.Close() })
	service := &evidence.Service{Store: db, Blobs: blobs}
	artifact, err := service.SaveBytes(t.Context(), evidence.Artifact{Project: "project", Run: "run", Kind: "log", Filename: "run.log", ContentType: "text/plain"}, []byte("private retained log"), 1024)
	if err != nil {
		t.Fatal(err)
	}
	token := sessions.Random()
	grant, err := storage.Put("artifact-access", sessions.Digest(token), "project", 0, map[string]any{"session": session.ID, "user": "owner", "project": "project", "run": "run", "artifact": artifact.ID, "action": "view", "expiresAt": time.Now().Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Commit(t.Context(), grant); err != nil {
		t.Fatal(err)
	}
	before, err := db.Get(t.Context(), "local-auth-mode", "installation")
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- serveArtifacts(ctx, address, "https://studio.example.test", "http://"+address, "", db, auth, service)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(2 * time.Second):
			t.Error("artifact listener did not stop")
		}
	})
	client := &http.Client{Timeout: time.Second}
	ready := false
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		response, err := client.Get("http://" + address + "/healthz")
		if err == nil {
			response.Body.Close()
			ready = response.StatusCode == 200
		}
		if ready {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !ready {
		t.Fatal("artifact listener did not become ready")
	}
	after, err := db.Get(t.Context(), "local-auth-mode", "installation")
	if err != nil || after.Revision != before.Revision {
		t.Fatal("artifact startup changed local sign-in mode", err)
	}
	access := func(expected int) {
		t.Helper()
		response, err := client.Get("http://" + address + "/access/" + token)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		data, err := io.ReadAll(response.Body)
		if err != nil || response.StatusCode != expected {
			t.Fatal("incorrect artifact access", response.StatusCode, err)
		}
		if expected == 200 && string(data) != "private retained log" {
			t.Fatal("wrong retained artifact")
		}
	}
	access(200)
	if err := localauth.Configure(t.Context(), directory, false); err != nil {
		t.Fatal(err)
	}
	access(401)
}
