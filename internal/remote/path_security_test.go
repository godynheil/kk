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

package remote

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSafeRemoteRel(t *testing.T) {
	tests := []struct {
		input   string
		wantOk  bool
		wantErr string
	}{
		{"foo", true, ""},
		{"foo/bar", true, ""},
		{"foo/../bar", true, ""},
		{"", false, "unsafe remote path"},
		{"/foo", false, "unsafe remote path"},
		{"../foo", false, "unsafe remote path"},
		{"foo/../../bar", false, "unsafe remote path"},
	}

	for _, tt := range tests {
		got, err := safeRemoteRel(tt.input)
		if tt.wantOk {
			if err != nil {
				t.Errorf("safeRemoteRel(%q) returned error: %v", tt.input, err)
			}

			expectedClean := filepath.Clean(filepath.FromSlash(tt.input))
			if got != expectedClean {
				t.Errorf("safeRemoteRel(%q) = %q, want %q", tt.input, got, expectedClean)
			}
		} else {
			if err == nil {
				t.Errorf("safeRemoteRel(%q) expected error, got nil", tt.input)
			} else if tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("safeRemoteRel(%q) error = %v, want substring %q", tt.input, err, tt.wantErr)
			}
		}
	}
}

func TestSafeJoinUnder(t *testing.T) {
	tempDir := filepath.Clean(t.TempDir())

	tests := []struct {
		name    string
		rel     string
		wantOk  bool
		wantErr string
	}{
		{"safe simple", "foo.txt", true, ""},
		{"safe subfolder", "sub/foo.txt", true, ""},
		{"safe cleaning", "sub/../foo.txt", true, ""},
		{"unsafe escape via parent dotdot", "../foo.txt", false, "unsafe remote path"},
		{"unsafe escape deep", "sub/../../foo.txt", false, "unsafe remote path"},
		{"unsafe absolute path", "/foo.txt", false, "unsafe remote path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := safeJoinUnder(tempDir, tt.rel)
			if tt.wantOk {
				if err != nil {
					t.Fatalf("safeJoinUnder returned error: %v", err)
				}
				if !strings.HasPrefix(got, tempDir) {
					t.Errorf("path %q does not start with root %q", got, tempDir)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error, got nil path %q", got)
				}
				if tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error = %v, want substring %q", err, tt.wantErr)
				}
			}
		})
	}
}
