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

package app

import (
	"testing"

	"github.com/godynheil/kk/internal/core"
)

func TestRequiredObjectsFromLive(t *testing.T) {
	live := map[string]core.LiveObject{
		"bbb": {
			OID:  "bbb",
			Size: 20,
			Refs: []core.ObjectRef{{Commit: "c2", Path: "assets/b.bin"}},
		},
		"aaa": {
			OID:  "aaa",
			Size: 10,
			Refs: []core.ObjectRef{{Commit: "c1", Path: "assets/a.bin"}, {Commit: "c3", Path: "copy/a.bin"}},
		},
	}

	got := RequiredObjectsFromLive(live)
	if len(got) != 2 {
		t.Fatalf("expected 2 objects, got %d", len(got))
	}
	if got[0].OID != "aaa" || got[0].Size != 10 || got[0].Path != "assets/a.bin" {
		t.Fatalf("unexpected first object: %+v", got[0])
	}
	if got[1].OID != "bbb" || got[1].Size != 20 || got[1].Path != "assets/b.bin" {
		t.Fatalf("unexpected second object: %+v", got[1])
	}
}
