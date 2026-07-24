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

import (
	"fmt"
	"regexp"
	"strconv"
)

var pointerRE = regexp.MustCompile(`(?m)^version kk-lfs-1\.0\.0\r?\noid sha256:([a-f0-9]{64})\r?\nsize ([0-9]+)\r?\n?$`)

func FormatPointer(p Pointer) string {
	return fmt.Sprintf("version %s\noid sha256:%s\nsize %d\n", PointerVersion, p.OID, p.Size)
}

func ParsePointerText(s string) (Pointer, bool) {
	m := pointerRE.FindStringSubmatch(s)
	if len(m) != 3 {
		return Pointer{}, false
	}
	size, err := strconv.ParseInt(m[2], 10, 64)
	if err != nil {
		return Pointer{}, false
	}
	return Pointer{OID: m[1], Size: size}, true
}

func ParsePointerBytes(b []byte) (Pointer, bool) {
	return ParsePointerText(string(b))
}
