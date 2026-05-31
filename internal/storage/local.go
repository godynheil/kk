package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/godynheil/kk/internal/core"
)

func (s Store) StoreObjectFromFile(path string) (core.Pointer, error) {
	oid, size, err := HashFile(path)
	if err != nil {
		return core.Pointer{}, err
	}
	p := core.Pointer{OID: oid, Size: size}
	obj := s.ObjectPath(oid)
	if fileExists(obj) {
		return p, nil
	}
	if err := os.MkdirAll(filepath.Dir(obj), 0o750); err != nil {
		return core.Pointer{}, err
	}
	out, err := os.CreateTemp(filepath.Dir(obj), "."+filepath.Base(obj)+".*.tmp")
	if err != nil {
		return core.Pointer{}, err
	}
	tmp := out.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = out.Close()
			_ = os.Remove(tmp)
		}
	}()
	in, err := os.Open(path) // #nosec G304 -- path is the source file selected by caller for object storage.
	if err != nil {
		return core.Pointer{}, err
	}
	defer func() {
		_ = in.Close()
	}()
	if _, err := io.Copy(out, in); err != nil {
		return core.Pointer{}, err
	}
	if err := out.Close(); err != nil {
		return core.Pointer{}, err
	}
	cleanup = false
	if err := os.Rename(tmp, obj); err != nil {
		_ = os.Remove(tmp)
		return core.Pointer{}, err
	}
	return p, nil
}

func (s Store) VerifyObject(p core.Pointer) error {
	path := s.ObjectPath(p.OID)
	actualOID, actualSize, err := HashFile(path)
	if err != nil {
		return err
	}
	if actualOID != p.OID {
		return fmt.Errorf("hash mismatch for %s: expected %s, got %s", path, p.OID, actualOID)
	}
	if actualSize != p.Size {
		return fmt.Errorf("size mismatch for %s: expected %d, got %d", path, p.Size, actualSize)
	}
	return nil
}

func (s Store) HasObject(p core.Pointer) bool {
	path := s.ObjectPath(p.OID)
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Size() == p.Size
}

func (s Store) PruneObject(oid string) error {
	path := s.ObjectPath(oid)
	if fileExists(path) {
		return os.Remove(path)
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

var objectOIDRe = regexp.MustCompile(`^[a-f0-9]{64}$`)

func (s Store) ListObjects() (map[string]string, error) {
	objects := map[string]string{}
	root := filepath.Join(s.Root, core.ObjectDir)
	if _, err := os.Stat(root); err != nil {
		return objects, nil
	}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if objectOIDRe.MatchString(name) {
			objects[name] = path
		}
		return nil
	})
	return objects, err
}

func (s Store) PruneObjects(oids []string) error {
	for _, oid := range oids {
		if err := s.PruneObject(oid); err != nil {
			return err
		}
	}
	return nil
}
