// Copyright (C) 2026 Godynheil A. Quisto <godynheil@quisto.ph>
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package app

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/godynheil/kk/internal/gdrive"
)

func TestGoogleAuthURLContainsExpectedParameters(t *testing.T) {
	scope := "https://www.googleapis.com/auth/drive.file"
	raw := GoogleAuthURL("client-1", "http://127.0.0.1:9999/callback", "state-1", "challenge-1", scope)
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	values := parsed.Query()
	if values.Get("client_id") != "client-1" {
		t.Fatalf("unexpected client_id: %q", values.Get("client_id"))
	}
	if values.Get("redirect_uri") != "http://127.0.0.1:9999/callback" {
		t.Fatalf("unexpected redirect_uri: %q", values.Get("redirect_uri"))
	}
	if values.Get("scope") != scope {
		t.Fatalf("unexpected scope: %q", values.Get("scope"))
	}
	if values.Get("code_challenge_method") != "S256" {
		t.Fatalf("unexpected code_challenge_method: %q", values.Get("code_challenge_method"))
	}

	fullScope := "https://www.googleapis.com/auth/drive"
	rawFull := GoogleAuthURL("client-1", "http://127.0.0.1:9999/callback", "state-1", "challenge-1", fullScope)
	parsedFull, err := url.Parse(rawFull)
	if err != nil {
		t.Fatal(err)
	}
	if parsedFull.Query().Get("scope") != fullScope {
		t.Fatalf("unexpected scope: %q", parsedFull.Query().Get("scope"))
	}
}

func TestSetupUnknownTarget(t *testing.T) {
	app := New(".")
	err := app.Setup([]string{"unknown"})
	if err == nil {
		t.Fatal("expected error for unknown setup target")
	}
	if !strings.Contains(err.Error(), "unknown setup target") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGoogleOAuthClientCredentialsUsesDefaultClientID(t *testing.T) {
	origClientID := DefaultGoogleOAuthClientID
	DefaultGoogleOAuthClientID = "test-default-client-id"
	defer func() { DefaultGoogleOAuthClientID = origClientID }()

	t.Setenv("KK_GOOGLE_CLIENT_ID", "")
	t.Setenv("KK_GOOGLE_CLIENT_SECRET", "")
	clientID, clientSecret, err := GoogleOAuthClientCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if clientID != "test-default-client-id" {
		t.Fatalf("unexpected clientID: %q", clientID)
	}
	if clientSecret != "" {
		t.Fatalf("unexpected clientSecret: %q", clientSecret)
	}
}

func TestListLocalAccounts(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("KK_GDRIVE_AUTH_DIR", tmpDir)

	auth := gdrive.Auth{
		ClientID:     "client-id-1",
		RefreshToken: "refresh-token-1",
		Email:        "user@example.com",
		DisplayName:  "Example User",
	}
	data, err := json.Marshal(auth)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(tmpDir, "profile1.json"), data, 0o600)
	if err != nil {
		t.Fatal(err)
	}

	accounts, err := listLocalAccounts()
	if err != nil {
		t.Fatal(err)
	}

	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}
	if accounts[0].Name != "profile1" {
		t.Errorf("expected profile name 'profile1', got %q", accounts[0].Name)
	}
	if accounts[0].Email != "user@example.com" {
		t.Errorf("expected email 'user@example.com', got %q", accounts[0].Email)
	}
}

func TestBuildCloneRemoteConfig_FallbackSingle(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("KK_GDRIVE_AUTH_DIR", tmpDir)

	auth := gdrive.Auth{
		ClientID:     "client-id-other",
		RefreshToken: "refresh-token-other",
		Email:        "other@example.com",
		DisplayName:  "Other User",
	}
	data, err := json.Marshal(auth)
	if err != nil {
		t.Fatal(err)
	}

	otherPath := filepath.Join(tmpDir, "other.json")
	err = os.WriteFile(otherPath, data, 0o600)
	if err != nil {
		t.Fatal(err)
	}

	app := App{}
	cfg, err := app.buildCloneRemoteConfig("drive", "folder-id-123", "")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.DriveAuthPath != otherPath {
		t.Errorf("expected DriveAuthPath to be %q, got %q", otherPath, cfg.DriveAuthPath)
	}
	if cfg.DriveFolderID != "folder-id-123" {
		t.Errorf("expected DriveFolderID to be 'folder-id-123', got %q", cfg.DriveFolderID)
	}
}

func TestBuildCloneRemoteConfig_MultipleChoice(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("KK_GDRIVE_AUTH_DIR", tmpDir)

	auth1 := gdrive.Auth{
		ClientID:     "client-1",
		RefreshToken: "refresh-1",
		Email:        "user1@example.com",
	}
	data1, _ := json.Marshal(auth1)
	_ = os.WriteFile(filepath.Join(tmpDir, "account1.json"), data1, 0o600)

	auth2 := gdrive.Auth{
		ClientID:     "client-2",
		RefreshToken: "refresh-2",
		Email:        "user2@example.com",
	}
	data2, _ := json.Marshal(auth2)
	_ = os.WriteFile(filepath.Join(tmpDir, "account2.json"), data2, 0o600)

	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write([]byte("2\n"))
	}()

	app := App{}
	cfg, err := app.buildCloneRemoteConfig("drive", "folder-id-123", "")
	os.Stdin = oldStdin
	_ = r.Close()

	if err != nil {
		t.Fatal(err)
	}

	expectedPath := filepath.Join(tmpDir, "account2.json")
	if cfg.DriveAuthPath != expectedPath {
		t.Errorf("expected DriveAuthPath to be %q, got %q", expectedPath, cfg.DriveAuthPath)
	}
}

func TestBuildCloneRemoteConfig_SingleChoiceSelect(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("KK_GDRIVE_AUTH_DIR", tmpDir)

	auth := gdrive.Auth{
		ClientID:     "client-id-other",
		RefreshToken: "refresh-token-other",
		Email:        "other@example.com",
		DisplayName:  "Other User",
	}
	data, err := json.Marshal(auth)
	if err != nil {
		t.Fatal(err)
	}

	otherPath := filepath.Join(tmpDir, "other.json")
	err = os.WriteFile(otherPath, data, 0o600)
	if err != nil {
		t.Fatal(err)
	}

	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write([]byte("1\n"))
	}()

	app := App{}
	cfg, err := app.buildCloneRemoteConfig("drive", "folder-id-123", "")
	os.Stdin = oldStdin
	_ = r.Close()

	if err != nil {
		t.Fatal(err)
	}

	if cfg.DriveAuthPath != otherPath {
		t.Errorf("expected DriveAuthPath to be %q, got %q", otherPath, cfg.DriveAuthPath)
	}
}
