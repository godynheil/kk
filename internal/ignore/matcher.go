package ignore

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/godynheil/kk/internal/core"
)

type Matcher struct {
	patterns []string
}

func Load(root string) Matcher {
	path := filepath.Join(root, core.KKIgnoreFile)
	f, err := os.Open(path) // #nosec G304 -- ignore file is read from the caller's repository root.
	if err != nil {
		return Matcher{}
	}
	defer func() {
		_ = f.Close()
	}()
	m := Matcher{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m.patterns = append(m.patterns, filepath.ToSlash(line))
	}
	return m
}

func (m Matcher) Ignored(path string) bool {
	path = filepath.ToSlash(filepath.Clean(path))
	if core.IsInsideKK(path) {
		return true
	}
	base := filepath.Base(path)
	for _, pattern := range m.patterns {
		if strings.HasSuffix(pattern, "/") {
			prefix := strings.TrimSuffix(pattern, "/")
			if path == prefix || strings.HasPrefix(path, prefix+"/") {
				return true
			}
			continue
		}
		if pattern == path || pattern == base {
			return true
		}
		if ok, _ := filepath.Match(pattern, path); ok {
			return true
		}
		if ok, _ := filepath.Match(pattern, base); ok {
			return true
		}
	}
	return false
}
