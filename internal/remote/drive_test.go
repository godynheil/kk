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
	"context"
	"testing"
	"time"

	"github.com/godynheil/kk/internal/gdrive"
)

func TestWaitForDriveObjectRetriesUntilVisible(t *testing.T) {
	attempts := 0
	file, ok, err := waitForDriveObject(context.Background(), time.Second, time.Millisecond, func(context.Context) (gdrive.File, bool, error) {
		attempts++
		if attempts < 3 {
			return gdrive.File{}, false, nil
		}
		return gdrive.File{ID: "file-1", Name: "object-1"}, true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected object to become visible")
	}
	if file.ID != "file-1" {
		t.Fatalf("unexpected file ID: %q", file.ID)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestWaitForDriveObjectStopsWhenTimeoutExpires(t *testing.T) {
	attempts := 0
	_, ok, err := waitForDriveObject(context.Background(), 5*time.Millisecond, time.Millisecond, func(context.Context) (gdrive.File, bool, error) {
		attempts++
		return gdrive.File{}, false, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("did not expect object to be found")
	}
	if attempts < 2 {
		t.Fatalf("expected retry attempts, got %d", attempts)
	}
}
