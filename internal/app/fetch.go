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
	"fmt"

	"github.com/godynheil/kk/internal/core"
	"github.com/godynheil/kk/internal/git"
)

func (a App) Fetch(args []string) error {
	client := git.New(a.Root)
	if err := client.EnsureRepository(); err != nil {
		return err
	}

	cfg, err := core.ReadConfig(a.Root)
	if err != nil {
		return err
	}

	if !hasNoGitRemote(cfg) {
		return client.Run(append([]string{"fetch"}, args...)...)
	}

	if len(cfg.PullRemotes()) == 0 {
		return fmt.Errorf("kk fetch: no pull-enabled remotes configured; run 'kk setup gdrive' or 'kk remote add ...' first")
	}

	fmt.Println("kk: fetching history from remote(s)...")
	result, err := fetchHistory(a.Root, cfg)
	if err != nil {
		return fmt.Errorf("kk fetch: %w", err)
	}

	if !result.Changed {
		fmt.Println("kk: already up to date")
		return nil
	}

	defaultBranch := result.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = "main"
	}
	fmt.Printf("kk: fetch complete (default branch: %s)\n", defaultBranch)
	fmt.Println("kk: to merge into the current branch, run: kk pull")
	return nil
}
