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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/godynheil/kk/internal/core"
	"github.com/godynheil/kk/internal/git"
	"github.com/godynheil/kk/internal/remote"
)

func hasNoGitRemote(cfg core.LocalConfig) bool {
	return len(cfg.GitRemotes()) == 0
}

const maxIncrementalBundles = 100

func incrementalBundleName(seq int) string {
	return fmt.Sprintf("inc-%06d.bundle", seq)
}

func readHistoryState(root string) (core.HistoryState, error) {
	state := core.HistoryState{
		Version: core.HistoryStateVersion,
		Remotes: map[string]core.RemoteHistoryState{},
	}
	data, err := os.ReadFile(filepath.Join(root, core.HistoryStateFile)) // #nosec G304 -- history state is read from the caller's repository root.
	if os.IsNotExist(err) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, err
	}
	if state.Remotes == nil {
		state.Remotes = map[string]core.RemoteHistoryState{}
	}
	return state, nil
}

func writeHistoryState(root string, state core.HistoryState) error {
	if state.Version == "" {
		state.Version = core.HistoryStateVersion
	}
	if state.Remotes == nil {
		state.Remotes = map[string]core.RemoteHistoryState{}
	}
	path := filepath.Join(root, core.HistoryStateFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func (a App) pushHistory(targets []core.NamedRemoteConfig) error {
	gitClient := git.New(a.Root)
	if !gitClient.HasHEAD() {
		return nil
	}

	branches, err := gitClient.ListAllBranches()
	if err != nil {
		return fmt.Errorf("history: listing branches: %w", err)
	}
	if len(branches) == 0 {
		return nil
	}

	defaultBranch := gitClient.DefaultBranch()

	branchMap := make(map[string]string, len(branches))
	for _, b := range branches {
		if sha, err := gitClient.BranchCommit(b); err == nil {
			branchMap[b] = strings.TrimSpace(sha)
		}
	}

	info, err := core.ReadRepoInfo(a.Root)
	if err != nil {
		return fmt.Errorf("history: reading repo info: %w", err)
	}

	histState, err := readHistoryState(a.Root)
	if err != nil {
		return fmt.Errorf("history: reading local state: %w", err)
	}

	for _, target := range targets {
		driver, err := remote.New(target.Name, target.Config)
		if err != nil {
			fmt.Printf("kk: [%s] history: failed to open remote: %v\n", target.Name, err)
			continue
		}

		remoteSnap, snapExists, err := driver.GetRefsSnapshot(info)
		if err != nil {
			fmt.Printf("kk: [%s] history: failed to read remote refs: %v\n", target.Name, err)
			continue
		}
		if snapExists {
			if err := remoteSnap.Validate(); err != nil {
				fmt.Printf("kk: [%s] history: invalid remote refs snapshot: %v\n", target.Name, err)
				continue
			}
		}

		if !snapExists {
			remoteSnap = core.RefsSnapshot{
				Version:  core.HistoryVersion,
				Branches: make(map[string]core.BranchHistory),
			}
		}

		newSnap := core.RefsSnapshot{
			Version:       core.HistoryVersion,
			DefaultBranch: defaultBranch,
			Branches:      make(map[string]core.BranchHistory),
			UpdatedAt:     time.Now().UTC(),
		}

		remoteState, ok := histState.Remotes[target.Name]
		if !ok {
			remoteState = core.RemoteHistoryState{
				Branches: make(map[string]core.BranchHistoryState),
			}
		}
		if remoteState.Branches == nil {
			remoteState.Branches = make(map[string]core.BranchHistoryState)
		}

		for branchName, localTip := range branchMap {
			remoteBranchSnap := remoteSnap.Branches[branchName]

			if remoteBranchSnap.Tip == localTip {
				fmt.Printf("kk: [%s] branch %s already up to date\n", target.Name, branchName)
				newSnap.Branches[branchName] = remoteBranchSnap
				continue
			}

			var bundleName, sinceRef string
			var nextBundles []string
			var nextSeq int

			baseExists := false
			if remoteBranchSnap.BaseRef != "" {
				baseExists = gitClient.IsAncestor(remoteBranchSnap.BaseRef, localTip)
			}

			if !baseExists || remoteBranchSnap.NextSeq > maxIncrementalBundles {
				bundleName = "full.bundle"
				sinceRef = ""
				nextBundles = []string{bundleName}
				nextSeq = 1
			} else {
				bundleName = incrementalBundleName(remoteBranchSnap.NextSeq)
				sinceRef = remoteBranchSnap.BaseRef
				nextBundles = append(append([]string{}, remoteBranchSnap.Bundles...), bundleName)
				nextSeq = remoteBranchSnap.NextSeq + 1
			}

			tmpDir, err := os.MkdirTemp("", "kk-bundle-*")
			if err != nil {
				return fmt.Errorf("history: creating temp dir: %w", err)
			}
			localBundlePath := filepath.Join(tmpDir, bundleName)

			fmt.Printf("kk: [%s] creating history bundle (%s) for branch %s...\n", target.Name, bundleName, branchName)
			if err := gitClient.CreateBundle(localBundlePath, sinceRef, branchName); err != nil {
				fmt.Printf("kk: [%s] history: bundle creation failed for %s: %v\n", target.Name, branchName, err)
				_ = os.RemoveAll(tmpDir)

				newSnap.Branches[branchName] = remoteBranchSnap
				continue
			}

			fmt.Printf("kk: [%s] uploading history bundle for %s...\n", target.Name, branchName)
			if err := driver.PutHistoryBundle(info, branchName, bundleName, localBundlePath); err != nil {
				fmt.Printf("kk: [%s] history: upload failed for %s: %v\n", target.Name, branchName, err)
				_ = os.RemoveAll(tmpDir)
				newSnap.Branches[branchName] = remoteBranchSnap
				continue
			}
			_ = os.RemoveAll(tmpDir)

			newSnap.Branches[branchName] = core.BranchHistory{
				BaseRef: localTip,
				NextSeq: nextSeq,
				Bundles: nextBundles,
				Tip:     localTip,
			}

			remoteState.Branches[branchName] = core.BranchHistoryState{
				LastAppliedBundle: bundleName,
				LastAppliedRef:    localTip,
			}

			fmt.Printf("kk: [%s] history pushed (%s, branch %s)\n", target.Name, bundleName, branchName)
		}

		remoteState.UpdatedAt = time.Now().UTC()
		histState.Remotes[target.Name] = remoteState
		if err := writeHistoryState(a.Root, histState); err != nil {
			fmt.Printf("kk: warning: failed to save history state: %v\n", err)
		}

		if err := driver.PutRefsSnapshot(info, newSnap); err != nil {
			fmt.Printf("kk: [%s] history: failed to write refs snapshot: %v\n", target.Name, err)
			continue
		}
	}
	return nil
}

type fetchHistoryResult struct {
	DefaultBranch string
	Changed       bool
}

func fetchHistory(root string, cfg core.LocalConfig) (fetchHistoryResult, error) {
	targets := cfg.PullRemotes()
	if len(targets) == 0 {
		return fetchHistoryResult{}, fmt.Errorf("no pull-enabled remotes configured")
	}

	info, err := core.ReadRepoInfo(root)
	if err != nil {
		return fetchHistoryResult{}, fmt.Errorf("history: reading repo info: %w", err)
	}

	histState, err := readHistoryState(root)
	if err != nil {
		return fetchHistoryResult{}, fmt.Errorf("history: reading local state: %w", err)
	}

	gitClient := git.New(root)
	defaultBranch := ""
	checkedAny := false

	for _, target := range targets {
		driver, err := remote.New(target.Name, target.Config)
		if err != nil {
			fmt.Printf("kk: [%s] history: failed to open remote: %v\n", target.Name, err)
			continue
		}

		remoteSnap, snapExists, err := driver.GetRefsSnapshot(info)
		if err != nil || !snapExists {
			if err != nil {
				fmt.Printf("kk: [%s] history: failed to read remote refs: %v\n", target.Name, err)
			}
			continue
		}
		if err := remoteSnap.Validate(); err != nil {
			fmt.Printf("kk: [%s] history: invalid remote refs snapshot: %v\n", target.Name, err)
			continue
		}
		checkedAny = true
		if defaultBranch == "" {
			defaultBranch = remoteSnap.DefaultBranch
		}

		if dlErr := driver.DownloadFiles(info, root, []string{core.TracksFile}, 1); dlErr != nil {
			fmt.Printf("kk: warning: [%s] failed to download tracks.json: %v\n", target.Name, dlErr)
		}

		localState := histState.Remotes[target.Name]
		if localState.Branches == nil {
			localState.Branches = make(map[string]core.BranchHistoryState)
		}

		changedAny := false
		var fetchedBranches []string

		for branchName, remoteBranchSnap := range remoteSnap.Branches {
			branchLocalState := localState.Branches[branchName]
			toApply := bundlesToApply(remoteBranchSnap.Bundles, branchLocalState.LastAppliedBundle)
			if len(toApply) == 0 {
				continue
			}

			fmt.Printf("kk: [%s] fetching %d history bundle(s) for branch %s...\n", target.Name, len(toApply), branchName)
			tmpDir, err := os.MkdirTemp("", "kk-fetch-*")
			if err != nil {
				return fetchHistoryResult{}, fmt.Errorf("history: creating temp dir: %w", err)
			}

			appliedBundle := branchLocalState.LastAppliedBundle
			for i, bundleName := range toApply {
				localPath := filepath.Join(tmpDir, bundleName)
				fmt.Printf("kk: [%s] downloading bundle %d/%d (%s) for branch %s...\n",
					target.Name, i+1, len(toApply), bundleName, branchName)
				if err := driver.GetHistoryBundle(info, branchName, bundleName, localPath); err != nil {
					_ = os.RemoveAll(tmpDir)
					return fetchHistoryResult{}, fmt.Errorf("history: downloading %s for %s: %w", bundleName, branchName, err)
				}
				if err := gitClient.ApplyBundle(localPath); err != nil {
					_ = os.RemoveAll(tmpDir)
					return fetchHistoryResult{}, fmt.Errorf("history: applying %s for %s: %w", bundleName, branchName, err)
				}
				appliedBundle = bundleName
			}
			_ = os.RemoveAll(tmpDir)

			localState.Branches[branchName] = core.BranchHistoryState{
				LastAppliedBundle: appliedBundle,
				LastAppliedRef:    remoteBranchSnap.BaseRef,
			}
			changedAny = true
			fetchedBranches = append(fetchedBranches, branchName)
		}

		if !changedAny {
			fmt.Printf("kk: [%s] history already up to date\n", target.Name)
			continue
		}

		localState.UpdatedAt = time.Now().UTC()
		histState.Remotes[target.Name] = localState
		if err := writeHistoryState(root, histState); err != nil {
			fmt.Printf("kk: warning: failed to save history state: %v\n", err)
		}

		fmt.Printf("kk: [%s] history fetched:\n", target.Name)
		sort.Strings(fetchedBranches)
		for _, b := range fetchedBranches {
			sha := remoteSnap.Branches[b].Tip
			short := sha
			if len(short) > 12 {
				short = short[:12]
			}
			fmt.Printf("kk:   refs/remotes/kk-history/%s → %s\n", b, short)
		}

		return fetchHistoryResult{
			DefaultBranch: remoteSnap.DefaultBranch,
			Changed:       true,
		}, nil
	}

	if checkedAny {
		return fetchHistoryResult{
			DefaultBranch: defaultBranch,
			Changed:       false,
		}, nil
	}

	return fetchHistoryResult{}, nil
}

func bundlesToApply(allBundles []string, lastApplied string) []string {
	if lastApplied == "" {
		return allBundles
	}
	for i, b := range allBundles {
		if b == lastApplied {
			return allBundles[i+1:]
		}
	}

	return allBundles
}
