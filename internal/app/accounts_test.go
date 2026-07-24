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
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/godynheil/kk/internal/gdrive"
)

func TestAccounts_NoAccounts(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("KK_GDRIVE_AUTH_DIR", tmpDir)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	app := New(".")
	err := app.Accounts(nil)
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "No accounts found in cache.") {
		t.Errorf("expected output to contain 'No accounts found in cache.', got: %q", output)
	}
}

func TestAccounts_NoAccountsJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("KK_GDRIVE_AUTH_DIR", tmpDir)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	app := New(".")
	err := app.Accounts([]string{"--json"})
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if strings.TrimSpace(output) != "[]" {
		t.Errorf("expected output to be '[]', got: %q", output)
	}
}

func TestAccounts_DeleteAll(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("KK_GDRIVE_AUTH_DIR", tmpDir)

	auth := gdrive.Auth{
		ClientID:     "client",
		AccessToken:  "access",
		RefreshToken: "refresh",
		Expiry:       time.Now().Add(time.Hour),
	}
	data, _ := json.Marshal(auth)
	if err := os.WriteFile(filepath.Join(tmpDir, "default.json"), data, 0o600); err != nil {
		t.Fatalf("failed to write default auth file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "work.json"), data, 0o600); err != nil {
		t.Fatalf("failed to write work auth file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "notes.txt"), []byte("keep"), 0o600); err != nil {
		t.Fatalf("failed to write non-auth file: %v", err)
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	app := New(".")
	err := app.Accounts([]string{"--delete-all"})
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	if !strings.Contains(buf.String(), "Deleted 2 account(s) from cache.") {
		t.Fatalf("expected delete summary, got: %q", buf.String())
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "default.json")); !os.IsNotExist(err) {
		t.Fatalf("expected default auth file to be deleted, stat error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "work.json")); !os.IsNotExist(err) {
		t.Fatalf("expected work auth file to be deleted, stat error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "notes.txt")); err != nil {
		t.Fatalf("expected non-auth file to remain: %v", err)
	}
}

func TestAccounts_DeleteAllJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("KK_GDRIVE_AUTH_DIR", tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "bad.json"), []byte("{invalid json"), 0o600); err != nil {
		t.Fatalf("failed to write mock auth file: %v", err)
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	app := New(".")
	err := app.Accounts([]string{"--delete-all", "--json"})
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	var result deleteAccountsResult
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal output: %v, output was: %s", err, buf.String())
	}
	if result.Deleted != 1 {
		t.Fatalf("expected 1 deleted account, got %d", result.Deleted)
	}
}

func TestAccounts_DeleteProfile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("KK_GDRIVE_AUTH_DIR", tmpDir)

	auth := gdrive.Auth{
		ClientID:     "client",
		AccessToken:  "access",
		RefreshToken: "refresh",
		Expiry:       time.Now().Add(time.Hour),
	}
	data, _ := json.Marshal(auth)
	if err := os.WriteFile(filepath.Join(tmpDir, "default.json"), data, 0o600); err != nil {
		t.Fatalf("failed to write default auth file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "work.json"), data, 0o600); err != nil {
		t.Fatalf("failed to write work auth file: %v", err)
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	app := New(".")
	err := app.Accounts([]string{"--delete", "work"})
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	if !strings.Contains(buf.String(), `Deleted account "work" from cache.`) {
		t.Fatalf("expected delete summary, got: %q", buf.String())
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "work.json")); !os.IsNotExist(err) {
		t.Fatalf("expected work auth file to be deleted, stat error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "default.json")); err != nil {
		t.Fatalf("expected default auth file to remain: %v", err)
	}
}

func TestAccounts_DeleteProfileJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("KK_GDRIVE_AUTH_DIR", tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "work.json"), []byte("{invalid json"), 0o600); err != nil {
		t.Fatalf("failed to write mock auth file: %v", err)
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	app := New(".")
	err := app.Accounts([]string{"--delete", "work", "--json"})
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	var result deleteAccountsResult
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal output: %v, output was: %s", err, buf.String())
	}
	if result.Deleted != 1 || result.Profile != "work" {
		t.Fatalf("expected work profile deletion, got %+v", result)
	}
}

func TestAccounts_DeleteProfileMissing(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("KK_GDRIVE_AUTH_DIR", tmpDir)

	app := New(".")
	err := app.Accounts([]string{"--delete", "missing"})
	if err == nil {
		t.Fatal("expected missing profile error")
	}
	if !strings.Contains(err.Error(), `account "missing" not found`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAccounts_CorruptedAccount(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("KK_GDRIVE_AUTH_DIR", tmpDir)

	err := os.WriteFile(filepath.Join(tmpDir, "bad.json"), []byte("{invalid json"), 0o600)
	if err != nil {
		t.Fatalf("failed to write mock auth file: %v", err)
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	app := New(".")
	err = app.Accounts([]string{"--json"})
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	var result []AccountInfo
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal output: %v, output was: %s", err, output)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}

	if result[0].Profile != "bad" || result[0].Status != "Corrupted" {
		t.Errorf("unexpected account info: %+v", result[0])
	}
}

func TestAccounts_OfflineStatus(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("KK_GDRIVE_AUTH_DIR", tmpDir)

	t.Setenv("KK_GDRIVE_API_BASE", "http://127.0.0.1:9999/offline")
	t.Setenv("KK_GDRIVE_TOKEN_ENDPOINT", "http://127.0.0.1:9999/offline")

	auth1 := gdrive.Auth{
		ClientID:     "client1",
		AccessToken:  "access1",
		RefreshToken: "refresh1",
		Expiry:       time.Now().Add(10 * time.Minute),
		Email:        "user1@gmail.com",
		DisplayName:  "User One",
	}
	data1, _ := json.Marshal(auth1)
	_ = os.WriteFile(filepath.Join(tmpDir, "active_offline.json"), data1, 0o600)

	auth2 := gdrive.Auth{
		ClientID:     "client2",
		AccessToken:  "access2",
		RefreshToken: "refresh2",
		Expiry:       time.Now().Add(-10 * time.Minute),
		Email:        "user2@gmail.com",
		DisplayName:  "User Two",
	}
	data2, _ := json.Marshal(auth2)
	_ = os.WriteFile(filepath.Join(tmpDir, "expired_offline.json"), data2, 0o600)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	app := New(".")
	err := app.Accounts([]string{"--json"})
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	var result []AccountInfo
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal output: %v, output was: %s", err, buf.String())
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}

	var pActive, pExpired *AccountInfo
	for i := range result {
		switch result[i].Profile {
		case "active_offline":
			pActive = &result[i]
		case "expired_offline":
			pExpired = &result[i]
		}
	}

	if pActive == nil || pActive.Status != "Active (Offline)" {
		t.Errorf("expected active_offline profile to have status 'Active (Offline)', got %+v", pActive)
	}
	if pExpired == nil || pExpired.Status != "Offline (Expired)" {
		t.Errorf("expected expired_offline profile to have status 'Offline (Expired)', got %+v", pExpired)
	}
}

func TestAccounts_OnlineActiveAndInvalid(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path == "/token" {
			_ = r.ParseForm()
			refreshToken := r.Form.Get("refresh_token")
			if refreshToken == "refresh_valid" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{
					"access_token": "new_access_token",
					"expires_in": 3600,
					"token_type": "Bearer",
					"scope": "drive"
				}`))
			} else {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error": "invalid_grant"}`))
			}
			return
		}

		if r.URL.Path == "/about" {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "Bearer access_valid" || authHeader == "Bearer new_access_token" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{
					"user": {
						"displayName": "Mock User",
						"emailAddress": "mock.user@gmail.com"
					}
				}`))
			} else {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error": "invalid_token"}`))
			}
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	t.Setenv("KK_GDRIVE_AUTH_DIR", tmpDir)
	t.Setenv("KK_GDRIVE_API_BASE", server.URL)
	t.Setenv("KK_GDRIVE_TOKEN_ENDPOINT", server.URL+"/token")

	auth1 := gdrive.Auth{
		ClientID:     "client1",
		AccessToken:  "expired_access",
		RefreshToken: "refresh_valid",
		Expiry:       time.Now().Add(-10 * time.Minute),
	}
	data1, _ := json.Marshal(auth1)
	_ = os.WriteFile(filepath.Join(tmpDir, "valid_user.json"), data1, 0o600)

	auth2 := gdrive.Auth{
		ClientID:     "client2",
		AccessToken:  "expired_access",
		RefreshToken: "refresh_invalid",
		Expiry:       time.Now().Add(-10 * time.Minute),
	}
	data2, _ := json.Marshal(auth2)
	_ = os.WriteFile(filepath.Join(tmpDir, "invalid_user.json"), data2, 0o600)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	app := New(".")
	err := app.Accounts([]string{"--json"})
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	var result []AccountInfo
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal output: %v, output was: %s", err, buf.String())
	}

	var pValid, pInvalid *AccountInfo
	for i := range result {
		switch result[i].Profile {
		case "valid_user":
			pValid = &result[i]
		case "invalid_user":
			pInvalid = &result[i]
		}
	}

	if pValid == nil || pValid.Status != "Active" || pValid.Email != "mock.user@gmail.com" || pValid.DisplayName != "Mock User" {
		t.Errorf("expected valid_user profile to be Active and have user info, got %+v", pValid)
	}
	if pInvalid == nil || pInvalid.Status != "Invalid Credentials" {
		t.Errorf("expected invalid_user profile to have status 'Invalid Credentials', got %+v", pInvalid)
	}

	updatedAuth, err := gdrive.LoadAuth(filepath.Join(tmpDir, "valid_user.json"))
	if err != nil {
		t.Fatalf("failed to load updated auth file: %v", err)
	}
	if updatedAuth.Email != "mock.user@gmail.com" || updatedAuth.DisplayName != "Mock User" {
		t.Errorf("expected updated auth file to have email and display name, got: %+v", updatedAuth)
	}
}
