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
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/godynheil/kk/internal/core"
	"github.com/godynheil/kk/internal/gdrive"
	"github.com/godynheil/kk/internal/remote"
	"github.com/godynheil/kk/internal/storage"
)

const GoogleDriveScope = "https://www.googleapis.com/auth/drive.file"

var DefaultGoogleOAuthClientID string

var DefaultGoogleOAuthClientSecret string

var googleOAuthHTTPClient = &http.Client{Timeout: 30 * time.Second}

type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

type oauthCallback struct {
	code string
	err  error
}

func (a App) Setup(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: kk setup <gdrive>")
	}
	switch strings.ToLower(args[0]) {
	case "gdrive", "google-drive":
		return a.setupGoogleDrive(args[1:])
	default:
		return fmt.Errorf("unknown setup target: %s", args[0])
	}
}

func (a App) setupGoogleDrive(args []string) error {
	authOnly := hasFlag(args, "--auth-only")
	args = removeFlags(args, "--auth-only")

	accountName := ""
	remoteName := "gdrive"
	folderID := ""
	scopeOpt := "file"
	var remaining []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--account":
			if i+1 >= len(args) {
				return fmt.Errorf("--account requires a value")
			}
			accountName = args[i+1]
			i++
		case "--name":
			if i+1 >= len(args) {
				return fmt.Errorf("--name requires a value")
			}
			remoteName = args[i+1]
			i++
		case "--folder":
			if i+1 >= len(args) {
				return fmt.Errorf("--folder requires a value")
			}
			folderID = args[i+1]
			i++
		case "--scope":
			if i+1 >= len(args) {
				return fmt.Errorf("--scope requires a value ('file' or 'full')")
			}
			scopeOpt = args[i+1]
			i++
		default:
			remaining = append(remaining, args[i])
		}
	}
	args = remaining

	var driveScope string
	switch scopeOpt {
	case "file":
		driveScope = "https://www.googleapis.com/auth/drive.file"
	case "full":
		driveScope = "https://www.googleapis.com/auth/drive"
	default:
		return fmt.Errorf("invalid scope %q: choose 'file' or 'full'", scopeOpt)
	}

	if len(args) != 0 {
		return fmt.Errorf("usage: kk setup gdrive [--name <remote-name>] [--account <profile>] [--folder <folder-id>] [--auth-only] [--scope file|full]")
	}
	if err := core.ValidateRemoteName(remoteName); err != nil {
		return err
	}

	accounts, err := listLocalAccounts()
	if err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)
	var authPath string
	useExisting := false

	if accountName != "" {
		var found *LocalAccount
		for i := range accounts {
			if accounts[i].Name == accountName {
				found = &accounts[i]
				break
			}
		}
		if found == nil {
			return fmt.Errorf("account %q not found; run 'kk accounts' to list saved accounts", accountName)
		}
		fmt.Printf("Validating saved credentials for account %q...\n", found.Name)
		client := gdrive.NewClient(found.Path)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, testErr := client.FetchUserInfo(ctx)
		cancel()
		if testErr != nil {
			return fmt.Errorf("account %q authentication failed: %w", accountName, testErr)
		}
		fmt.Printf("Using saved Google Drive account %q.\n", found.Name)
		authPath = found.Path
		useExisting = true
	} else if len(accounts) > 0 {
		fmt.Println("Found local Google Drive account(s):")
		for i, acc := range accounts {
			info := acc.Name
			if acc.Email != "" && acc.DisplayName != "" {
				info = fmt.Sprintf("%s (%s - %s)", acc.Name, acc.Email, acc.DisplayName)
			} else if acc.Email != "" {
				info = fmt.Sprintf("%s (%s)", acc.Name, acc.Email)
			}
			fmt.Printf("  [%d] %s\n", i+1, info)
		}
		fmt.Printf("  [%d] Log in to a new Google Drive account online\n", len(accounts)+1)

		choice, err := promptChoice(reader, len(accounts)+1)
		if err != nil {
			return err
		}

		if choice <= len(accounts) {
			selected := accounts[choice-1]
			authPath = selected.Path
			useExisting = true

			fmt.Printf("Validating saved credentials for account %q...\n", selected.Name)
			client := gdrive.NewClient(authPath)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_, testErr := client.FetchUserInfo(ctx)
			cancel()

			if testErr != nil {
				fmt.Printf("Failed to authenticate using saved account %q: %v\n", selected.Name, testErr)
				loginNew, err := promptConfirm(reader, "Would you like to log in online instead?", true)
				if err != nil {
					return err
				}
				if loginNew {
					useExisting = false
				} else {
					return fmt.Errorf("authentication failed: %w", testErr)
				}
			} else {
				fmt.Printf("Using saved Google Drive account %q.\n", selected.Name)
			}
		}
	}

	if !useExisting {
		var err error
		authPath, err = a.loginNewGoogleDriveAccount(reader, driveScope)
		if err != nil {
			return err
		}
	}

	if authOnly {
		fmt.Println("kk: auth-only mode — skipping folder setup.")
		fmt.Println("    Ask the project owner for the Drive project folder ID, then run:")
		fmt.Println("    kk clone drive:<project-folder-id>")
		return nil
	}

	client := gdrive.NewClient(authPath)

	info, err := core.ReadRepoInfo(a.Root)
	if err != nil {
		return err
	}

	projectFolderID := folderID
	if projectFolderID == "" {
		fmt.Println("Initializing project folders on Google Drive...")
		rootFolderID, err := ensureGoogleDriveProjectFolder(client)
		if err != nil {
			return err
		}
		projectFolder, err := client.EnsureFolder(context.Background(), rootFolderID, info.Name)
		if err != nil {
			return fmt.Errorf("creating project folder on Drive: %w", err)
		}
		projectFolderID = projectFolder.ID
	}

	cfg, err := core.ReadConfig(a.Root)
	if err != nil {
		return err
	}
	remoteCfg := core.RemoteConfig{
		Type:          "drive",
		DisplayName:   "Google Drive",
		Provider:      "google-drive",
		DriveFolderID: projectFolderID,
		DriveAuthPath: authPath,
		ObjectRoot:    "objects",
		ManifestRoot:  "manifests",
		VerifyMode:    "local-hash",
		Priority:      20,
		Pull:          true,
		Push:          true,
		Tags:          []string{"cloud", "gdrive"},
	}

	driver, err := remote.New(remoteName, remoteCfg)
	if err != nil {
		return err
	}
	fmt.Println("Verifying remote connection...")
	if err := driver.Check(); err != nil {
		return err
	}

	fmt.Println("Checking for existing project files on remote...")
	if existing, ok, err := driver.ReadRemoteRepoInfo(info); err == nil && ok {
		if existing.RepoID != info.RepoID {
			fmt.Printf("kk: adopting remote project ID: %s (local was %s)\n", existing.RepoID, info.RepoID)
			if err := core.WriteRepoInfo(a.Root, existing); err != nil {
				return fmt.Errorf("failed to adopt remote project ID: %w", err)
			}
			info = existing
		}
	}

	fmt.Println("Testing upload...")
	probePath, probePointer, err := writeDriveProbeFile()
	if err != nil {
		return err
	}
	defer func() {
		_ = os.Remove(probePath)
	}()
	if err := driver.PutObject(info, probePointer, probePath); err != nil {
		return err
	}

	fmt.Println("Testing download...")
	downloaded, err := os.CreateTemp("", "kk-gdrive-download-*")
	if err != nil {
		return err
	}
	downloadPath := downloaded.Name()
	_ = downloaded.Close()
	defer func() {
		_ = os.Remove(downloadPath)
	}()
	if err := driver.GetObject(info, probePointer, downloadPath); err != nil {
		return err
	}

	cfg.Remotes[remoteName] = remoteCfg
	if cfg.DefaultRemote == "" {
		cfg.DefaultRemote = remoteName
	}
	if err := core.WriteConfig(a.Root, cfg); err != nil {
		return err
	}

	fmt.Printf("kk: Drive project folder-id: %s\n", projectFolderID)
	fmt.Printf("kk: Run 'kk push' to upload your project to Drive.\n")
	fmt.Println("    Share the project folder ID with teammates so they can clone:")
	fmt.Printf("    kk clone drive:%s\n", projectFolderID)
	fmt.Println("Remote ready.")
	return nil
}

func (a App) loginNewGoogleDriveAccount(reader *bufio.Reader, driveScope string) (string, error) {
	clientID, clientSecret, err := GoogleOAuthClientCredentials()
	if err != nil {
		return "", err
	}

	fmt.Println("Connect Google Drive")
	fmt.Println("Opening browser for authorization...")

	auth, err := authorizeGoogleDrive(clientID, clientSecret, driveScope)
	if err != nil {
		return "", err
	}

	profileName, err := promptProfileName(reader)
	if err != nil {
		return "", err
	}

	var authPath string
	for {
		targetPath := gdrive.DefaultAuthPath(profileName)
		if err := validateDefaultAuthPath(profileName, targetPath); err != nil {
			return "", err
		}
		if _, statErr := os.Stat(targetPath); !os.IsNotExist(statErr) { // #nosec G703 -- profileName is validated and targetPath is constrained to the configured auth directory.
			overwrite, err := promptConfirm(reader, fmt.Sprintf("Account %q already exists. Overwrite?", profileName), false)
			if err != nil {
				return "", err
			}
			if overwrite {
				authPath = targetPath
				break
			} else {
				profileName, err = promptProfileName(reader)
				if err != nil {
					return "", err
				}
			}
		} else {
			authPath = targetPath
			break
		}
	}

	if err := gdrive.SaveAuth(authPath, auth); err != nil {
		return "", err
	}
	fmt.Printf("kk: Google Drive auth saved to %s\n", authPath)

	fmt.Println("Retrieving Google user profile info...")
	client := gdrive.NewClient(authPath)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	_, _ = client.FetchUserInfo(ctx)
	cancel()

	return authPath, nil
}

func GoogleOAuthClientCredentials() (string, string, error) {
	clientID := strings.TrimSpace(os.Getenv("KK_GOOGLE_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("KK_GOOGLE_CLIENT_SECRET"))

	if clientID != "" {
		if clientSecret == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Google OAuth client secret: ")
			line, _ := reader.ReadString('\n')
			clientSecret = strings.TrimSpace(line)
			if clientSecret == "" {
				return "", "", fmt.Errorf("KK_GOOGLE_CLIENT_SECRET is required for Desktop app OAuth clients")
			}
		}
		return clientID, clientSecret, nil
	}

	if DefaultGoogleOAuthClientID != "" {
		if clientSecret == "" {
			clientSecret = DefaultGoogleOAuthClientSecret
		}
		return DefaultGoogleOAuthClientID, clientSecret, nil
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Google OAuth client ID: ")
	clientID, _ = reader.ReadString('\n')
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return "", "", fmt.Errorf("google oauth client ID is required")
	}
	fmt.Print("Google OAuth client secret: ")
	clientSecret, _ = reader.ReadString('\n')
	clientSecret = strings.TrimSpace(clientSecret)
	if clientSecret == "" {
		return "", "", fmt.Errorf("google OAuth client secret is required for Desktop app clients")
	}
	return clientID, clientSecret, nil
}

func authorizeGoogleDrive(clientID, clientSecret, scope string) (gdrive.Auth, error) {
	verifier, err := randomToken(64)
	if err != nil {
		return gdrive.Auth{}, err
	}
	state, err := randomToken(32)
	if err != nil {
		return gdrive.Auth{}, err
	}
	challenge := pkceChallenge(verifier)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return gdrive.Auth{}, err
	}
	defer func() {
		_ = listener.Close()
	}()
	redirectURI := "http://" + listener.Addr().String() + "/oauth2/callback"

	resultCh := make(chan oauthCallback, 1)
	server := &http.Server{
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			if q.Get("state") != state {
				http.Error(w, "state mismatch", http.StatusBadRequest)
				resultCh <- oauthCallback{err: fmt.Errorf("google oauth state mismatch")}
				return
			}
			if oauthErr := q.Get("error"); oauthErr != "" {
				http.Error(w, "authorization failed", http.StatusBadRequest)
				resultCh <- oauthCallback{err: fmt.Errorf("google authorization failed: %s", oauthErr)}
				return
			}
			code := q.Get("code")
			if code == "" {
				http.Error(w, "missing code", http.StatusBadRequest)
				resultCh <- oauthCallback{err: fmt.Errorf("google authorization did not return a code")}
				return
			}
			_, _ = w.Write([]byte("Google Drive connected. You can close this window."))
			resultCh <- oauthCallback{code: code}
		}),
	}
	go func() {
		_ = server.Serve(listener)
	}()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	authURL := GoogleAuthURL(clientID, redirectURI, state, challenge, scope)
	fmt.Println("Open this URL to connect Google Drive:")
	fmt.Println(authURL)
	if err := openBrowser(authURL); err != nil {
		fmt.Println("Browser launch failed. Copy and paste the URL above manually.")
	}

	select {
	case result := <-resultCh:
		if result.err != nil {
			return gdrive.Auth{}, result.err
		}
		fmt.Println("Authorization received. Exchanging code for access tokens...")
		return exchangeGoogleAuthCode(clientID, clientSecret, redirectURI, verifier, result.code)
	case <-time.After(5 * time.Minute):
		return gdrive.Auth{}, fmt.Errorf("timed out waiting for Google authorization")
	}
}

func GoogleAuthURL(clientID, redirectURI, state, challenge, scope string) string {
	values := url.Values{
		"response_type":         {"code"},
		"client_id":             {clientID},
		"redirect_uri":          {redirectURI},
		"scope":                 {scope},
		"access_type":           {"offline"},
		"prompt":                {"consent"},
		"state":                 {state},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	return "https://accounts.google.com/o/oauth2/v2/auth?" + values.Encode()
}

func exchangeGoogleAuthCode(clientID, clientSecret, redirectURI, verifier, code string) (gdrive.Auth, error) {
	values := url.Values{
		"client_id":     {clientID},
		"code":          {code},
		"code_verifier": {verifier},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {redirectURI},
	}
	if clientSecret != "" {
		values.Set("client_secret", clientSecret)
	}
	req, err := http.NewRequest(http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(values.Encode()))
	if err != nil {
		return gdrive.Auth{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := googleOAuthHTTPClient.Do(req)
	if err != nil {
		return gdrive.Auth{}, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		body = bytes.TrimSpace(body)
		if len(body) > 0 {
			return gdrive.Auth{}, fmt.Errorf("exchange Google authorization code: %s: %s", resp.Status, body)
		}
		return gdrive.Auth{}, fmt.Errorf("exchange Google authorization code: %s", resp.Status)
	}
	var tr oauthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return gdrive.Auth{}, err
	}
	return gdrive.Auth{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		TokenType:    tr.TokenType,
		Scope:        tr.Scope,
		Expiry:       time.Now().UTC().Add(time.Duration(tr.ExpiresIn) * time.Second),
	}, nil
}

func ensureGoogleDriveProjectFolder(client *gdrive.Client) (string, error) {
	ctx := context.Background()
	kkFolder, err := client.EnsureFolder(ctx, "root", "KK")
	if err != nil {
		return "", err
	}
	return kkFolder.ID, nil
}

func randomToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func validateDefaultAuthPath(profileName, authPath string) error {
	if filepath.Base(authPath) != profileName+".json" {
		return fmt.Errorf("invalid Google Drive auth profile path: %s", authPath)
	}
	authDir, err := filepath.Abs(gdrive.AuthDir())
	if err != nil {
		return err
	}
	authPathAbs, err := filepath.Abs(authPath)
	if err != nil {
		return err
	}
	if filepath.Dir(authPathAbs) != authDir {
		return fmt.Errorf("auth path escapes Google Drive auth directory: %s", authPath)
	}
	return nil
}

func openBrowser(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return fmt.Errorf("refusing to open invalid browser URL")
	}
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL).Start() // #nosec G204 -- fixed Windows URL opener; rawURL is parsed and scheme-limited above.
	case "darwin":
		return exec.Command("open", rawURL).Start() // #nosec G204 -- fixed macOS URL opener; rawURL is parsed and scheme-limited above.
	default:
		return exec.Command("xdg-open", rawURL).Start() // #nosec G204 -- fixed Linux URL opener; rawURL is parsed and scheme-limited above.
	}
}

func writeDriveProbeFile() (string, core.Pointer, error) {
	tmp, err := os.CreateTemp("", "kk-gdrive-probe-*.bin")
	if err != nil {
		return "", core.Pointer{}, err
	}
	path := tmp.Name()
	if _, err := tmp.Write([]byte("kk gdrive probe\n")); err != nil {
		_ = tmp.Close()
		_ = os.Remove(path)
		return "", core.Pointer{}, err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(path)
		return "", core.Pointer{}, err
	}
	oid, size, err := storage.HashFile(path)
	if err != nil {
		_ = os.Remove(path)
		return "", core.Pointer{}, err
	}
	return path, core.Pointer{OID: oid, Size: size}, nil
}

type LocalAccount struct {
	Name        string
	Email       string
	DisplayName string
	Path        string
}

func listLocalAccounts() ([]LocalAccount, error) {
	dir := gdrive.AuthDir()
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var accounts []LocalAccount
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, f.Name())
		auth, err := gdrive.LoadAuth(path)
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(f.Name(), ".json")
		accounts = append(accounts, LocalAccount{
			Name:        name,
			Email:       auth.Email,
			DisplayName: auth.DisplayName,
			Path:        path,
		})
	}
	return accounts, nil
}

func promptChoice(reader *bufio.Reader, max int) (int, error) {
	for {
		fmt.Printf("Select an option [1-%d]: ", max)
		line, err := reader.ReadString('\n')
		if err != nil {
			return 0, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var val int
		if _, err := fmt.Sscanf(line, "%d", &val); err == nil && val >= 1 && val <= max {
			return val, nil
		}
		fmt.Println("Invalid choice. Please try again.")
	}
}

func promptProfileName(reader *bufio.Reader) (string, error) {
	for {
		fmt.Print("Enter a profile name for this account (default: 'default'): ")
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		name := strings.TrimSpace(line)
		if name == "" {
			return "default", nil
		}

		valid := true
		for _, r := range name {
			if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' && r != '-' {
				valid = false
				break
			}
		}
		if valid {
			return name, nil
		}
		fmt.Println("Invalid name. Use only alphanumeric characters, underscores, or hyphens.")
	}
}

func promptConfirm(reader *bufio.Reader, message string, defaultYes bool) (bool, error) {
	for {
		if defaultYes {
			fmt.Printf("%s [Y/n]: ", message)
		} else {
			fmt.Printf("%s [y/N]: ", message)
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}
		line = strings.ToLower(strings.TrimSpace(line))
		if line == "" {
			return defaultYes, nil
		}
		if line == "y" || line == "yes" {
			return true, nil
		}
		if line == "n" || line == "no" {
			return false, nil
		}
		fmt.Println("Please answer 'y' or 'n'.")
	}
}
