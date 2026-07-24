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

package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/godynheil/kk/internal/core"
)

func TestHashFileAndBytes(t *testing.T) {
	tempDir := t.TempDir()
	content := []byte("hello, kk storage engine!")

	// Test HashBytes
	expectedHash := sha256.Sum256(content)
	expectedOID := hex.EncodeToString(expectedHash[:])
	expectedSize := int64(len(content))

	oid, size := HashBytes(content)
	if oid != expectedOID || size != expectedSize {
		t.Errorf("HashBytes() = (%s, %d); expected (%s, %d)", oid, size, expectedOID, expectedSize)
	}

	// Test HashFile success
	filePath := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(filePath, content, 0o644)
	if err != nil {
		t.Fatal(err)
	}

	oid, size, err = HashFile(filePath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if oid != expectedOID || size != expectedSize {
		t.Errorf("HashFile() = (%s, %d); expected (%s, %d)", oid, size, expectedOID, expectedSize)
	}

	// Test HashFile missing file error
	_, _, err = HashFile(filepath.Join(tempDir, "missing-file.txt"))
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestStorePaths(t *testing.T) {
	s := New("virtual-repo")
	if s.Root != "virtual-repo" {
		t.Errorf("expected Root = %q, got %q", "virtual-repo", s.Root)
	}

	// Fallback to "."
	s2 := New("")
	if s2.Root != "." {
		t.Errorf("expected fallback Root = \".\", got %q", s2.Root)
	}

	// ObjectPath boundary check
	shortPath := s.ObjectPath("abc")
	expectedShortPath := filepath.Join("virtual-repo", core.ObjectDir, "invalid", "abc")
	if shortPath != expectedShortPath {
		t.Errorf("expected short OID path to map to %q, got %s", expectedShortPath, shortPath)
	}

	longPath := s.ObjectPath("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	expectedPath := filepath.Join("virtual-repo", core.ObjectDir, "12", "34", "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	if longPath != expectedPath {
		t.Errorf("expected %q, got %q", expectedPath, longPath)
	}

	// TempPath check
	tempPath := s.TempPath("foo.tmp")
	if tempPath != filepath.Join("virtual-repo", core.TmpDir, "foo.tmp") {
		t.Errorf("expected temp path to be virtual-repo/.kk/tmp/foo.tmp, got %s", tempPath)
	}
}

func TestStoreVerifyHasPruneObjects(t *testing.T) {
	tempDir := t.TempDir()
	s := New(tempDir)

	content := []byte("large object content simulation")
	testFile := filepath.Join(tempDir, "asset.dat")
	err := os.WriteFile(testFile, content, 0o644)
	if err != nil {
		t.Fatal(err)
	}

	expectedOID, expectedSize := HashBytes(content)

	// 1. Store the object from file
	ptr, err := s.StoreObjectFromFile(testFile)
	if err != nil {
		t.Fatalf("StoreObjectFromFile failed: %v", err)
	}

	if ptr.OID != expectedOID || ptr.Size != expectedSize {
		t.Errorf("expected pointer (%s, %d), got (%s, %d)", expectedOID, expectedSize, ptr.OID, ptr.Size)
	}

	// 2. HasObject checks
	if !s.HasObject(ptr) {
		t.Error("HasObject returned false for stored object")
	}

	// 3. VerifyObject success
	err = s.VerifyObject(ptr)
	if err != nil {
		t.Errorf("VerifyObject failed: %v", err)
	}

	// 4. Idempotency: Store again shouldn't error or double-allocate
	ptr2, err := s.StoreObjectFromFile(testFile)
	if err != nil {
		t.Fatalf("StoreObjectFromFile second run failed: %v", err)
	}
	if ptr2.OID != ptr.OID {
		t.Error("idempotency check failed")
	}

	// 5. VerifyObject failure cases (mismatch size)
	badPtrSize := core.Pointer{OID: ptr.OID, Size: ptr.Size + 1}
	err = s.VerifyObject(badPtrSize)
	if err == nil {
		t.Error("expected VerifyObject error for size mismatch")
	}

	// 6. VerifyObject failure cases (hash mismatch / corrupted content)
	objPath := s.ObjectPath(ptr.OID)
	err = os.WriteFile(objPath, []byte("corrupted!"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	err = s.VerifyObject(ptr)
	if err == nil {
		t.Error("expected VerifyObject error for corrupted content")
	}

	// Reset to valid state for listing and pruning
	err = os.WriteFile(objPath, content, 0o644)
	if err != nil {
		t.Fatal(err)
	}

	// 7. ListObjects
	objMap, err := s.ListObjects()
	if err != nil {
		t.Fatalf("ListObjects failed: %v", err)
	}
	if len(objMap) != 1 {
		t.Errorf("expected 1 object in store, got %d", len(objMap))
	}
	if _, ok := objMap[ptr.OID]; !ok {
		t.Errorf("expected OID %s in object map", ptr.OID)
	}

	// 8. PruneObject
	err = s.PruneObject(ptr.OID)
	if err != nil {
		t.Errorf("PruneObject failed: %v", err)
	}
	if s.HasObject(ptr) {
		t.Error("object was not removed by PruneObject")
	}

	// Prune already missing object shouldn't error
	err = s.PruneObject(ptr.OID)
	if err != nil {
		t.Errorf("PruneObject on missing object returned error: %v", err)
	}

	// 9. PruneObjects bulk
	// Store again
	_, _ = s.StoreObjectFromFile(testFile)
	if !s.HasObject(ptr) {
		t.Fatal("failed to restore object for bulk prune test")
	}

	err = s.PruneObjects([]string{ptr.OID})
	if err != nil {
		t.Errorf("PruneObjects failed: %v", err)
	}
	if s.HasObject(ptr) {
		t.Error("object was not removed by bulk PruneObjects")
	}
}
