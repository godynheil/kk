package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

func HashFile(path string) (oid string, size int64, err error) {
	f, err := os.Open(path) // #nosec G304 -- hashing intentionally opens caller-selected app-managed files.
	if err != nil {
		return "", 0, err
	}
	defer func() {
		_ = f.Close()
	}()
	h := sha256.New()
	size, err = io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), size, nil
}

func HashBytes(data []byte) (oid string, size int64) {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), int64(len(data))
}
