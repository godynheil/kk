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

import "testing"

func TestShouldShowActivity(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{name: "push", args: []string{"push"}, want: true},
		{name: "status json", args: []string{"status", "--json"}, want: false},
		{name: "help", args: []string{"help"}, want: false},
		{name: "version", args: []string{"version"}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldShowActivity(tc.args); got != tc.want {
				t.Fatalf("shouldShowActivity(%v) = %v, want %v", tc.args, got, tc.want)
			}
		})
	}
}
