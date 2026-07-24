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

package core

import "testing"

func TestIsValidBundleName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid full", "full.bundle", true},
		{"valid inc", "inc-000001.bundle", true},
		{"valid inc high sequence", "inc-999999.bundle", true},
		{"invalid prefix", "in-000001.bundle", false},
		{"invalid suffix", "inc-000001.bundl", false},
		{"invalid length", "inc-00001.bundle", false},
		{"invalid chars", "inc-000a01.bundle", false},
		{"empty", "", false},
		{"path traversal attempt", "../full.bundle", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidBundleName(tt.input); got != tt.want {
				t.Errorf("IsValidBundleName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsValidBranchName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"simple", "main", true},
		{"with slash", "feature/login", true},
		{"with multiple slashes", "feature/auth/oauth", true},
		{"with dashes and underscores", "bug-fix_123", true},
		{"path traversal dotdot", "../../etc", false},
		{"path traversal backslash", "feature\\login", false},
		{"trailing slash", "feature/", false},
		{"leading slash", "/feature", false},
		{"consecutive slashes", "feature//login", false},
		{"empty name", "", false},
		{"component starting with dot", "feature/.login", false},
		{"component ending with dot", "feature/login.", false},
		{"special git character tilde", "feat~1", false},
		{"special git character caret", "feat^1", false},
		{"special git character space", "feat 1", false},
		{"special git character colon", "feat:1", false},
		{"special git character question", "feat?1", false},
		{"special git character asterisk", "feat*1", false},
		{"special git character openbracket", "feat[1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidBranchName(tt.input); got != tt.want {
				t.Errorf("IsValidBranchName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestRefsSnapshotValidate(t *testing.T) {
	tests := []struct {
		name    string
		snap    RefsSnapshot
		wantErr bool
	}{
		{
			name: "valid snapshot",
			snap: RefsSnapshot{
				DefaultBranch: "main",
				Branches: map[string]BranchHistory{
					"main": {
						Bundles: []string{"full.bundle", "inc-000001.bundle"},
					},
					"feature/login": {
						Bundles: []string{"full.bundle"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid default branch",
			snap: RefsSnapshot{
				DefaultBranch: "main/../etc",
				Branches: map[string]BranchHistory{
					"main": {
						Bundles: []string{"full.bundle"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid branch name",
			snap: RefsSnapshot{
				DefaultBranch: "main",
				Branches: map[string]BranchHistory{
					"main/../../etc": {
						Bundles: []string{"full.bundle"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid bundle name",
			snap: RefsSnapshot{
				DefaultBranch: "main",
				Branches: map[string]BranchHistory{
					"main": {
						Bundles: []string{"full.bundle", "../../etc"},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.snap.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("RefsSnapshot.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
