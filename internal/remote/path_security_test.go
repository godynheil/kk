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
