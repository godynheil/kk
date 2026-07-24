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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	folderMimeType = "application/vnd.google-apps.folder"
	defaultAPIBase = "https://www.googleapis.com/drive/v3"
	defaultUpload  = "https://www.googleapis.com/upload/drive/v3"
)

func createHTTPClient(timeout time.Duration, disablePool bool) *http.Client {
	client := &http.Client{Timeout: timeout}
	if !disablePool {
		client.Transport = &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			DisableKeepAlives:   false,
		}
	}
	return client
}

var defaultHTTPClient = createHTTPClient(30*time.Second, false)

type File struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	MimeType    string   `json:"mimeType"`
	Parents     []string `json:"parents"`
	Size        string   `json:"size"`
	MD5Checksum string   `json:"md5Checksum"`
}

type Client struct {
	AuthPath   string
	HTTPClient *http.Client
	APIBase    string
	UploadBase string
}

type APIError struct {
	Op         string
	Status     string
	StatusCode int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Op, e.Status)
}

type listResponse struct {
	Files []File `json:"files"`
}

func NewClient(authPath string) *Client {
	return NewClientWithTimeout(authPath, defaultHTTPClient.Timeout, false)
}

func NewClientWithTimeout(authPath string, timeout time.Duration, disablePool bool) *Client {
	apiBase := defaultAPIBase
	if env := os.Getenv("KK_GDRIVE_API_BASE"); env != "" {
		apiBase = env
	}
	uploadBase := defaultUpload
	if env := os.Getenv("KK_GDRIVE_UPLOAD_BASE"); env != "" {
		uploadBase = env
	}

	if env := os.Getenv("KK_GDRIVE_UPLOAD_TIMEOUT"); env != "" {
		if seconds, err := strconv.ParseInt(env, 10, 64); err == nil && seconds > 0 {
			timeout = time.Duration(seconds) * time.Second
		}
	}

	return &Client{
		AuthPath:   authPath,
		HTTPClient: createHTTPClient(timeout, disablePool),
		APIBase:    apiBase,
		UploadBase: uploadBase,
	}
}

type UserInfo struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

func (c *Client) FetchUserInfo(ctx context.Context) (UserInfo, error) {
	u := c.APIBase + "/about?fields=user(displayName,emailAddress)"
	resp, err := c.request(ctx, http.MethodGet, u, nil, "")
	if err != nil {
		return UserInfo{}, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return UserInfo{}, fmt.Errorf("google api returned status %s", resp.Status)
	}
	var about struct {
		User struct {
			DisplayName  string `json:"displayName"`
			EmailAddress string `json:"emailAddress"`
		} `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&about); err != nil {
		return UserInfo{}, err
	}

	auth, err := LoadAuth(c.AuthPath)
	if err == nil {
		updated := false
		if auth.Email != about.User.EmailAddress {
			auth.Email = about.User.EmailAddress
			updated = true
		}
		if auth.DisplayName != about.User.DisplayName {
			auth.DisplayName = about.User.DisplayName
			updated = true
		}
		if updated {
			_ = SaveAuth(c.AuthPath, auth)
		}
	}

	return UserInfo{
		Email:       about.User.EmailAddress,
		DisplayName: about.User.DisplayName,
	}, nil
}

func (c *Client) token(ctx context.Context) (string, error) {
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = defaultHTTPClient
	}
	_, token, err := EnsureAccessToken(ctx, httpClient, c.AuthPath)
	return token, err
}

func (c *Client) request(ctx context.Context, method, rawURL string, body io.Reader, contentType string) (*http.Response, error) {
	token, err := c.token(ctx)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = defaultHTTPClient
	}
	return httpClient.Do(req)
}

func (c *Client) GetFolder(ctx context.Context, folderID string) (File, error) {
	u := fmt.Sprintf("%s/files/%s?fields=id,name,mimeType,parents&supportsAllDrives=true", c.APIBase, url.PathEscape(folderID))
	resp, err := c.request(ctx, http.MethodGet, u, nil, "")
	if err != nil {
		return File{}, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return File{}, &APIError{Op: "get Drive folder", Status: resp.Status, StatusCode: resp.StatusCode}
	}
	var f File
	return f, json.NewDecoder(resp.Body).Decode(&f)
}

func (c *Client) CheckFolder(ctx context.Context, folderID string) error {
	u := fmt.Sprintf("%s/files/%s?fields=id,name,mimeType&supportsAllDrives=true", c.APIBase, url.PathEscape(folderID))
	resp, err := c.request(ctx, http.MethodGet, u, nil, "")
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{Op: "check Drive folder", Status: resp.Status, StatusCode: resp.StatusCode}
	}
	var f File
	if err := json.NewDecoder(resp.Body).Decode(&f); err != nil {
		return err
	}
	if f.MimeType != folderMimeType {
		return fmt.Errorf("drive target is not a folder")
	}
	return nil
}

func (c *Client) ListChildren(ctx context.Context, parentID string) ([]File, error) {
	var all []File
	pageToken := ""
	for {
		q := fmt.Sprintf("'%s' in parents and trashed = false", escapeQuery(parentID))
		values := url.Values{
			"q":                         {q},
			"pageSize":                  {"1000"},
			"fields":                    {"nextPageToken,files(id,name,mimeType,parents,size,md5Checksum)"},
			"supportsAllDrives":         {"true"},
			"includeItemsFromAllDrives": {"true"},
		}
		if pageToken != "" {
			values.Set("pageToken", pageToken)
		}
		u := c.APIBase + "/files?" + values.Encode()
		resp, err := c.request(ctx, http.MethodGet, u, nil, "")
		if err != nil {
			return nil, err
		}
		var listed struct {
			NextPageToken string `json:"nextPageToken"`
			Files         []File `json:"files"`
		}
		decodeErr := json.NewDecoder(resp.Body).Decode(&listed)
		_ = resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("list Drive children: HTTP %d", resp.StatusCode)
		}
		if decodeErr != nil {
			return nil, decodeErr
		}
		all = append(all, listed.Files...)
		if listed.NextPageToken == "" {
			break
		}
		pageToken = listed.NextPageToken
	}
	return all, nil
}

func (c *Client) FindChild(ctx context.Context, parentID, name string, wantFolder bool) (File, bool, error) {
	q := fmt.Sprintf("name = '%s' and '%s' in parents and trashed = false", escapeQuery(name), escapeQuery(parentID))
	if wantFolder {
		q += " and mimeType = '" + folderMimeType + "'"
	}
	values := url.Values{
		"q":                         {q},
		"pageSize":                  {"10"},
		"fields":                    {"files(id,name,mimeType,parents,size,md5Checksum)"},
		"supportsAllDrives":         {"true"},
		"includeItemsFromAllDrives": {"true"},
	}
	u := c.APIBase + "/files?" + values.Encode()
	resp, err := c.request(ctx, http.MethodGet, u, nil, "")
	if err != nil {
		return File{}, false, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return File{}, false, fmt.Errorf("list Drive files: %s", resp.Status)
	}
	var listed listResponse
	if err := json.NewDecoder(resp.Body).Decode(&listed); err != nil {
		return File{}, false, err
	}
	if len(listed.Files) == 0 {
		return File{}, false, nil
	}
	return listed.Files[0], true, nil
}

func (c *Client) CreateFolder(ctx context.Context, parentID, name string) (File, error) {
	payload := map[string]any{
		"name":     name,
		"mimeType": folderMimeType,
		"parents":  []string{parentID},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return File{}, err
	}
	u := c.APIBase + "/files?fields=id,name,mimeType,parents&supportsAllDrives=true"
	resp, err := c.request(ctx, http.MethodPost, u, bytes.NewReader(body), "application/json")
	if err != nil {
		return File{}, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return File{}, fmt.Errorf("create Drive folder: %s", resp.Status)
	}
	var file File
	return file, json.NewDecoder(resp.Body).Decode(&file)
}

func (c *Client) EnsureFolder(ctx context.Context, parentID, name string) (File, error) {
	if file, ok, err := c.FindChild(ctx, parentID, name, true); err != nil {
		return File{}, err
	} else if ok {
		return file, nil
	}
	return c.CreateFolder(ctx, parentID, name)
}

func (c *Client) DeleteFile(ctx context.Context, fileID string) error {
	u := fmt.Sprintf("%s/files/%s?supportsAllDrives=true", c.APIBase, url.PathEscape(fileID))
	resp, err := c.request(ctx, http.MethodDelete, u, nil, "")
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("delete Drive file: %s", resp.Status)
	}
	return nil
}

func (c *Client) UploadFile(ctx context.Context, parentID, name, localPath, contentType string) (File, error) {
	file, err := os.Open(localPath) // #nosec G304 -- localPath is an app-managed upload path supplied by callers after validation.
	if err != nil {
		return File{}, err
	}
	defer func() {
		_ = file.Close()
	}()
	if contentType == "" {
		contentType = detectContentType(localPath)
	}
	info, err := file.Stat()
	if err != nil {
		return File{}, err
	}
	meta := map[string]any{
		"name":    name,
		"parents": []string{parentID},
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return File{}, err
	}
	return c.uploadResumable(ctx, http.MethodPost, c.UploadBase+"/files?uploadType=resumable&fields=id,name,mimeType,parents,size,md5Checksum&supportsAllDrives=true", metaJSON, contentType, info.Size(), file)
}

func (c *Client) uploadResumable(ctx context.Context, method, rawURL string, metadata []byte, contentType string, size int64, file *os.File) (File, error) {
	token, err := c.token(ctx)
	if err != nil {
		return File{}, err
	}
	req, err := http.NewRequestWithContext(ctx, method, rawURL, bytes.NewReader(metadata))
	if err != nil {
		return File{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("X-Upload-Content-Type", contentType)
	req.Header.Set("X-Upload-Content-Length", fmt.Sprintf("%d", size))
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = defaultHTTPClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return File{}, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return File{}, fmt.Errorf("start Drive upload: %s", resp.Status)
	}
	location := resp.Header.Get("Location")
	if location == "" {
		return File{}, fmt.Errorf("drive upload session did not return a location header")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return File{}, err
	}
	putReq, err := http.NewRequestWithContext(ctx, http.MethodPut, location, file)
	if err != nil {
		return File{}, err
	}
	putReq.Header.Set("Authorization", "Bearer "+token)
	putReq.Header.Set("Content-Type", contentType)
	putReq.ContentLength = size
	putResp, err := httpClient.Do(putReq)
	if err != nil {
		return File{}, err
	}
	defer func() {
		_ = putResp.Body.Close()
	}()
	if putResp.StatusCode < 200 || putResp.StatusCode >= 300 {
		return File{}, fmt.Errorf("upload Drive file: %s", putResp.Status)
	}
	var out File
	return out, json.NewDecoder(putResp.Body).Decode(&out)
}

func (c *Client) UploadFileChunked(ctx context.Context, parentID, name, localPath, contentType string, chunkSize int64) (File, error) {
	file, err := os.Open(localPath) // #nosec G304 -- localPath is an app-managed upload path supplied by callers after validation.
	if err != nil {
		return File{}, err
	}
	defer func() {
		_ = file.Close()
	}()
	if contentType == "" {
		contentType = detectContentType(localPath)
	}
	info, err := file.Stat()
	if err != nil {
		return File{}, err
	}
	totalSize := info.Size()

	meta := map[string]any{
		"name":    name,
		"parents": []string{parentID},
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return File{}, err
	}

	token, err := c.token(ctx)
	if err != nil {
		return File{}, err
	}

	uploadURL := c.UploadBase + "/files?uploadType=resumable&fields=id,name,mimeType,parents,size,md5Checksum&supportsAllDrives=true"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bytes.NewReader(metaJSON))
	if err != nil {
		return File{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("X-Upload-Content-Type", contentType)
	req.Header.Set("X-Upload-Content-Length", fmt.Sprintf("%d", totalSize))

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = defaultHTTPClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return File{}, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return File{}, fmt.Errorf("start chunked Drive upload: %s", resp.Status)
	}

	location := resp.Header.Get("Location")
	if location == "" {
		return File{}, fmt.Errorf("drive chunked upload session did not return a location header")
	}

	var uploaded int64
	for uploaded < totalSize {
		chunkEnd := uploaded + chunkSize
		if chunkEnd > totalSize {
			chunkEnd = totalSize
		}

		chunkLen := chunkEnd - uploaded

		putReq, err := http.NewRequestWithContext(ctx, http.MethodPut, location, io.LimitReader(file, chunkLen))
		if err != nil {
			return File{}, err
		}
		putReq.Header.Set("Authorization", "Bearer "+token)
		putReq.Header.Set("Content-Type", contentType)
		putReq.ContentLength = chunkLen

		if chunkEnd < totalSize {
			putReq.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", uploaded, chunkEnd-1, totalSize))
		} else {
			putReq.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", uploaded, chunkEnd-1, totalSize))
		}

		putResp, err := httpClient.Do(putReq)
		if err != nil {
			return File{}, fmt.Errorf("upload chunk %d-%d: %w", uploaded, chunkEnd-1, err)
		}

		body, _ := io.ReadAll(putResp.Body)
		_ = putResp.Body.Close()

		if putResp.StatusCode == 308 {
			rangeHeader := putResp.Header.Get("Range")
			if rangeHeader != "" {
				var lastByte int64
				_, err = fmt.Sscanf(rangeHeader, "bytes=0-%d", &lastByte)
				if err == nil {
					uploaded = lastByte + 1
					if _, err := file.Seek(uploaded, io.SeekStart); err != nil {
						return File{}, fmt.Errorf("seek to resume point: %w", err)
					}
					continue
				}
			}
			uploaded = chunkEnd
		} else if putResp.StatusCode >= 200 && putResp.StatusCode < 300 {
			var out File
			if err := json.Unmarshal(body, &out); err != nil {
				return File{}, fmt.Errorf("parse upload response: %w", err)
			}
			return out, nil
		} else {
			return File{}, fmt.Errorf("chunk upload failed with status %s: %s", putResp.Status, string(body))
		}
	}

	return File{}, fmt.Errorf("chunked upload completed without final response")
}

func (c *Client) DownloadFile(ctx context.Context, fileID, localPath string) error {
	if err := os.MkdirAll(filepath.Dir(localPath), 0o750); err != nil {
		return err
	}
	tmp := localPath + ".tmp"
	u := fmt.Sprintf("%s/files/%s?alt=media&supportsAllDrives=true", c.APIBase, url.PathEscape(fileID))
	resp, err := c.request(ctx, http.MethodGet, u, nil, "")
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download Drive file: %s", resp.Status)
	}
	out, err := os.Create(tmp) // #nosec G304 -- tmp is derived from caller-provided localPath plus a fixed temporary suffix.
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(out, resp.Body)
	if copyErr == nil && resp.ContentLength >= 0 && written != resp.ContentLength {
		copyErr = fmt.Errorf("download truncated: expected %d bytes, got %d", resp.ContentLength, written)
	}
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, localPath)
}

func escapeQuery(value string) string {
	return strings.ReplaceAll(value, "'", "\\'")
}

func detectContentType(path string) string {
	if ext := strings.ToLower(filepath.Ext(path)); ext != "" {
		if ct := mime.TypeByExtension(ext); ct != "" {
			return ct
		}
	}
	return "application/octet-stream"
}
