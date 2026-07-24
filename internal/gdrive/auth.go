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

package gdrive

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func tokenEndpoint() string {
	if env := os.Getenv("KK_GDRIVE_TOKEN_ENDPOINT"); env != "" {
		return env
	}
	return "https://oauth2.googleapis.com/token"
}

type Auth struct {
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret,omitempty"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	Expiry       time.Time `json:"expiry"`
	Email        string    `json:"email,omitempty"`
	DisplayName  string    `json:"display_name,omitempty"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

func AuthDir() string {
	if envDir := os.Getenv("KK_GDRIVE_AUTH_DIR"); envDir != "" {
		return envDir
	}
	cfg, err := os.UserConfigDir()
	if err != nil || cfg == "" {
		cfg = "."
	}
	return filepath.Join(cfg, "KK", "gdrive")
}

func DefaultAuthPath(name string) string {
	if strings.TrimSpace(name) == "" {
		name = "default"
	}
	return filepath.Join(AuthDir(), name+".json")
}

func LoadAuth(path string) (Auth, error) {
	var auth Auth
	data, err := os.ReadFile(path) // #nosec G304 -- auth path is selected by app configuration/profile handling.
	if err != nil {
		return auth, err
	}
	if err := json.Unmarshal(data, &auth); err != nil {
		return auth, err
	}
	if auth.ClientID == "" || auth.RefreshToken == "" {
		return auth, fmt.Errorf("invalid Google Drive auth file: missing client_id or refresh_token")
	}
	return auth, nil
}

func SaveAuth(path string, auth Auth) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	data, err := json.MarshalIndent(auth, "", "  ") // #nosec G117 -- this function intentionally persists OAuth credentials to a 0600 auth file.
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

func EnsureAccessToken(ctx context.Context, httpClient *http.Client, path string) (Auth, string, error) {
	auth, err := LoadAuth(path)
	if err != nil {
		return Auth{}, "", err
	}

	if auth.AccessToken != "" && time.Until(auth.Expiry) > 5*time.Minute {
		return auth, auth.AccessToken, nil
	}
	values := url.Values{
		"client_id":     {auth.ClientID},
		"refresh_token": {auth.RefreshToken},
		"grant_type":    {"refresh_token"},
	}
	if auth.ClientSecret != "" {
		values.Set("client_secret", auth.ClientSecret)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint(), strings.NewReader(values.Encode()))
	if err != nil {
		return Auth{}, "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return Auth{}, "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Auth{}, "", fmt.Errorf("refresh Google token: %s", resp.Status)
	}
	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return Auth{}, "", err
	}
	auth.AccessToken = tr.AccessToken
	auth.TokenType = tr.TokenType
	if tr.Scope != "" {
		auth.Scope = tr.Scope
	}
	auth.Expiry = time.Now().UTC().Add(time.Duration(tr.ExpiresIn) * time.Second)
	if tr.RefreshToken != "" {
		auth.RefreshToken = tr.RefreshToken
	}
	if err := SaveAuth(path, auth); err != nil {
		return Auth{}, "", err
	}
	return auth, auth.AccessToken, nil
}
