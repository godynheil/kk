package storage

import (
	"path/filepath"

	"github.com/godynheil/kk/internal/core"
)

type Store struct {
	Root string
}

func New(root string) Store {
	if root == "" {
		root = "."
	}
	return Store{Root: root}
}

func (s Store) ObjectPath(oid string) string {
	if len(oid) < 4 {
		return filepath.Join(s.Root, core.ObjectDir, "invalid", oid)
	}
	return filepath.Join(s.Root, core.ObjectDir, oid[:2], oid[2:4], oid)
}

func (s Store) TempPath(name string) string {
	return filepath.Join(s.Root, core.TmpDir, name)
}
