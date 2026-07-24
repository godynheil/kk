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
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/godynheil/kk/internal/core"
	"github.com/godynheil/kk/internal/git"
	"github.com/godynheil/kk/internal/remote"
	"github.com/godynheil/kk/internal/storage"
)

type downloadResult struct {
	Remote string
	Local  bool
}

func (a App) Pull(args []string) error {
	client := git.New(a.Root)
	if err := client.EnsureRepository(); err != nil {
		return err
	}
	fmt.Printf("kk: processing pull on %s...\n", currentBranchLabel(client))

	noMerge, args := extractFlag(args, "--no-merge")
	syncRemotes, args := extractFlag(args, "--sync")
	workers, gitArgs := parsePullWorkers(args)

	cfg, err := core.ReadConfig(a.Root)
	if err != nil {
		return err
	}

	if !client.HasRemotes() {
		if hasNoGitRemote(cfg) {
			return a.pullViaHistory(client, cfg, noMerge, workers, syncRemotes)
		}
		fmt.Println("kk: no git history remotes configured — skipping history pull")
		fmt.Println("    (to sync large files use 'kk pull-file .' or 'kk pull-file --all')")
		return nil
	}

	if noMerge {
		err := client.Run(append([]string{"fetch"}, gitArgs...)...)
		if err != nil {
			return err
		}
		fmt.Println("kk: history fetched (--no-merge) — run 'kk pull' without --no-merge to merge")
		return nil
	}

	pullErr := a.runWithTemporaryDematerialization(client, func() error {
		return client.Run(append([]string{"pull"}, gitArgs...)...)
	})
	if pullErr != nil {
		if client.HasMergeConflicts() {
			return fmt.Errorf(
				"kk pull left merge conflicts — resolve all conflicts, "+
					"stage the results with 'kk stage', then run 'kk commit' to complete the merge: %w",
				pullErr,
			)
		}

		errMsg := pullErr.Error()
		if strings.Contains(errMsg, "no tracking information") ||
			strings.Contains(errMsg, "specify which branch") {
			branch := client.CurrentBranch()
			if branch == "" {
				branch = "<branch>"
			}
			return fmt.Errorf(
				"kk: branch %q has no upstream tracking branch.\n"+
					"  Set one with:\n"+
					"    git --git-dir=.kk/git --work-tree=. branch --set-upstream-to=<remote>/%s %s\n"+
					"  Or pull explicitly:\n"+
					"    kk pull <remote> %s",
				branch, branch, branch, branch,
			)
		}
		return pullErr
	}

	if err := a.materializeNewPointers(workers, syncRemotes); err != nil {
		fmt.Printf("kk: warning: could not materialise all pointer files: %v\n", err)
	}

	return a.Fsck(false)
}

func (a App) pullViaHistory(client git.Client, cfg core.LocalConfig, noMerge bool, workers int, syncRemotes bool) error {
	result, err := fetchHistory(a.Root, cfg)
	if err != nil {
		return fmt.Errorf("kk: could not fetch history from remote: %w", err)
	}

	if !result.Changed {
		if !noMerge {
			currentBranch := client.CurrentBranch()
			if currentBranch != "" {
				ref := "refs/remotes/kk-history/" + currentBranch
				tracking, refErr := client.Output("rev-parse", "--verify", ref)
				head, headErr := client.HeadCommit()
				tracking = strings.TrimSpace(tracking)
				head = strings.TrimSpace(head)
				if refErr == nil && headErr == nil && tracking != "" && head != "" && tracking != head && client.IsAncestor(head, tracking) {
					fmt.Printf("kk: merging %s into %s...\n", ref, currentBranch)
					mergeErr := a.runWithTemporaryDematerialization(client, func() error {
						return client.MergeHistoryBranch(currentBranch)
					})
					if mergeErr != nil {
						if client.HasMergeConflicts() {
							return fmt.Errorf(
								"kk: merge conflict after history fetch - resolve all conflicts,\n"+
									"  stage the results with 'kk stage', then run 'kk commit' to complete the merge.\n"+
									"  To fetch without merging next time, use: kk pull --no-merge\n"+
									"  Original error: %w",
								mergeErr,
							)
						}
						return fmt.Errorf("kk: history merge failed: %w", mergeErr)
					}
				}
			}
		}
		if err := a.materializeNewPointers(workers, syncRemotes); err != nil {
			fmt.Printf("kk: warning: could not materialise all pointer files: %v\n", err)
		}
		return a.Fsck(false)
	}

	if noMerge {
		fmt.Println("kk: history fetched (--no-merge) — run 'kk pull' without --no-merge to merge")
		return nil
	}

	currentBranch := client.CurrentBranch()
	if currentBranch == "" {
		fmt.Println("kk: warning: HEAD is detached — skipping auto-merge; run 'kk merge' manually")
		return nil
	}

	fmt.Printf("kk: merging refs/remotes/kk-history/%s into %s...\n", currentBranch, currentBranch)
	mergeErr := a.runWithTemporaryDematerialization(client, func() error {
		return client.MergeHistoryBranch(currentBranch)
	})
	if mergeErr != nil {
		if client.HasMergeConflicts() {
			return fmt.Errorf(
				"kk: merge conflict after history fetch — resolve all conflicts,\n"+
					"  stage the results with 'kk stage', then run 'kk commit' to complete the merge.\n"+
					"  To fetch without merging next time, use: kk pull --no-merge\n"+
					"  Original error: %w",
				mergeErr,
			)
		}
		return fmt.Errorf("kk: history merge failed: %w", mergeErr)
	}

	if err := a.materializeNewPointers(workers, syncRemotes); err != nil {
		fmt.Printf("kk: warning: could not materialise all pointer files: %v\n", err)
	}

	return a.Fsck(false)
}

func parsePullWorkers(args []string) (workers int, rest []string) {
	for i := 0; i < len(args); i++ {
		if args[i] == "--workers" && i+1 < len(args) {
			if n, err := strconv.Atoi(args[i+1]); err == nil && n > 0 {
				workers = n
			}
			i++
			continue
		}
		if strings.HasPrefix(args[i], "--workers=") {
			if n, err := strconv.Atoi(strings.TrimPrefix(args[i], "--workers=")); err == nil && n > 0 {
				workers = n
			}
			continue
		}
		rest = append(rest, args[i])
	}
	return
}

func extractFlag(args []string, flag string) (found bool, rest []string) {
	for _, a := range args {
		if a == flag {
			found = true
		} else {
			rest = append(rest, a)
		}
	}
	return
}

func (a App) materializeNewPointers(flagWorkers int, syncRemotes bool) error {
	objects, err := a.requiredObjectsInHEAD()
	if err != nil {
		return err
	}

	var targets []string
	for _, obj := range objects {
		if _, ok, err := pointerFromWorkingFile(a.Root, obj.Path); err == nil && ok {
			targets = append(targets, obj.Path)
		}
	}

	if len(targets) == 0 {
		return nil
	}

	fmt.Printf("kk: %d pointer file(s) need materialising after pull\n", len(targets))
	w := resolveWorkers(flagWorkers)
	if w > len(targets) {
		w = len(targets)
	}
	bar := newMultiProgressBar(len(targets), "pulling", w)

	type matResult struct {
		slotID int
		path   string
		err    error
	}

	jobs := make(chan string)
	results := make(chan matResult, len(targets))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < w; i++ {
		slotID := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case path, ok := <-jobs:
					if !ok {
						return
					}
					bar.SetSlot(slotID, path)
					matErr := a.materialize(path, false, false, syncRemotes)
					select {
					case results <- matResult{slotID: slotID, path: path, err: matErr}:
					case <-ctx.Done():
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		for _, f := range targets {
			select {
			case jobs <- f:
			case <-ctx.Done():
				return
			}
		}
		close(jobs)
	}()
	go func() {
		wg.Wait()
		close(results)
	}()

	var warnings []string
	for r := range results {
		bar.Complete(r.slotID, r.path)
		if r.err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", r.path, r.err))
		}
	}
	succeeded := len(targets) - len(warnings)
	bar.Finish(fmt.Sprintf("kk: materialised %d/%d pointer file(s)", succeeded, len(targets)))
	for _, w := range warnings {
		fmt.Printf("kk: warning: could not materialise %s\n", w)
	}
	if len(warnings) > 0 {
		return fmt.Errorf("%d pointer file(s) could not be materialised", len(warnings))
	}
	return nil
}

func (a App) PullFile(args []string) error {
	syncRemotes := hasFlag(args, "--sync")
	force := hasFlag(args, "--force")
	allFlag := hasFlag(args, "--all")
	args = removeFlags(args, "--force", "--all", "--sync")

	workers := 0
	for i := 0; i < len(args); i++ {
		if args[i] == "--workers" && i+1 < len(args) {
			if n, err := strconv.Atoi(args[i+1]); err == nil && n > 0 {
				workers = n
			}
			args = append(args[:i], args[i+2:]...)
			i--
		}
	}

	if len(args) == 0 && !allFlag {
		return fmt.Errorf(
			"usage: kk pull-file [--force] [--workers N] [--sync] <file...>\n" +
				"       kk pull-file [--force] [--workers N] [--sync] .         (materialize all)\n" +
				"       kk pull-file [--force] [--workers N] [--sync] --all     (materialize all)",
		)
	}

	expandAll := allFlag
	if !expandAll {
		for _, f := range args {
			if f == "." {
				expandAll = true
				break
			}
			_, abs, pathErr := safeRepoPath(a.Root, f)
			if pathErr != nil {
				return pathErr
			}
			if info, err := os.Stat(abs); err == nil && info.IsDir() {
				expandAll = true
				break
			}
		}
	}

	if !expandAll && len(args) == 1 {
		return a.materialize(args[0], force, true, syncRemotes)
	}

	var targets []string

	if expandAll {
		objects, err := a.requiredObjectsInHEAD()
		if err != nil {
			return err
		}

		var prefixes []string
		if !allFlag {
			for _, f := range args {
				if f != "." {
					rel, _, err := safeRepoPath(a.Root, f)
					if err != nil {
						return err
					}
					prefixes = append(prefixes, filepath.Clean(filepath.FromSlash(rel))+string(filepath.Separator))
				}
			}
		}
		matchesPrefix := func(path string) bool {
			if len(prefixes) == 0 {
				return true
			}
			for _, pfx := range prefixes {
				if strings.HasPrefix(filepath.Clean(path)+string(filepath.Separator), pfx) ||
					strings.HasPrefix(path, pfx) {
					return true
				}
			}
			return false
		}

		for _, obj := range objects {
			if !matchesPrefix(obj.Path) {
				continue
			}
			if force {

				targets = append(targets, obj.Path)
			} else {
				if _, ok, err := pointerFromWorkingFile(a.Root, obj.Path); err == nil && ok {
					targets = append(targets, obj.Path)
				}
			}
		}

		if len(targets) == 0 {
			if force {
				fmt.Println("kk: no pointer files found in HEAD")
			} else {
				fmt.Println("kk: all large files are already materialized")
			}
			return nil
		}
		fmt.Printf("kk: %d pointer file(s) to materialize\n", len(targets))
	} else {
		for _, arg := range args {
			rel, _, err := safeRepoPath(a.Root, arg)
			if err != nil {
				return err
			}
			targets = append(targets, rel)
		}
	}

	w := resolveWorkers(workers)
	if w > len(targets) {
		w = len(targets)
	}
	bar := newMultiProgressBar(len(targets), "pulling", w)

	type matResult struct {
		slotID int
		path   string
		err    error
	}

	jobs := make(chan string)
	results := make(chan matResult, len(targets))

	var wg sync.WaitGroup
	for i := 0; i < w; i++ {
		slotID := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				bar.SetSlot(slotID, path)
				err := a.materialize(path, force, false, syncRemotes)
				results <- matResult{slotID: slotID, path: path, err: err}
			}
		}()
	}

	go func() {
		for _, f := range targets {
			jobs <- f
		}
		close(jobs)
	}()
	go func() {
		wg.Wait()
		close(results)
	}()

	var warnings []string
	for r := range results {
		bar.Complete(r.slotID, r.path)
		if r.err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", r.path, r.err))
		}
	}
	succeeded := len(targets) - len(warnings)
	bar.Finish(fmt.Sprintf("kk: materialized %d/%d file(s)", succeeded, len(targets)))
	for _, warn := range warnings {
		fmt.Printf("kk: warning: could not materialize %s\n", warn)
	}
	if len(warnings) > 0 {
		return fmt.Errorf("%d file(s) could not be materialized", len(warnings))
	}
	return nil
}

func (a App) materialize(rel string, force bool, verbose bool, syncRemotes ...bool) error {
	var err error
	rel, _, err = safeRepoPath(a.Root, rel)
	if err != nil {
		return err
	}
	p, ok, err := a.pointerForMaterialize(rel)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%s is not a kk pointer file", rel)
	}
	store := storage.New(a.Root)
	source := downloadResult{Local: true}
	if force {
		if verbose {
			fmt.Printf("kk: forcing re-download of %s (%s)\n", rel, p.OID)
		}
		var err error
		source, err = a.downloadObject(p, store.ObjectPath(p.OID), verbose)
		if err != nil {
			return err
		}
	} else if !store.HasObject(p) {
		var err error
		source, err = a.downloadObject(p, store.ObjectPath(p.OID), verbose)
		if err != nil {
			return err
		}
	}
	if err := store.VerifyObject(p); err != nil {
		return err
	}
	in, err := os.Open(store.ObjectPath(p.OID))
	if err != nil {
		return err
	}
	defer func() {
		_ = in.Close()
	}()
	tmpDir := filepath.Join(a.Root, core.TmpDir)
	if err := os.MkdirAll(tmpDir, 0o750); err != nil {
		return err
	}
	out, err := os.CreateTemp(tmpDir, "mat-*.tmp")
	if err != nil {
		return err
	}
	tmp := out.Name()
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	_, dst, err := safeRepoPath(a.Root, rel)
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if verbose {
		if source.Local {
			fmt.Printf("materialized %s using local cache\n", rel)
		} else {
			fmt.Printf("materialized %s using remote %s\n", rel, source.Remote)
		}
	}

	if len(syncRemotes) > 0 && syncRemotes[0] {
		if err := a.SyncObjectAcrossRemotes(p, verbose); err != nil {
			if verbose {
				fmt.Printf("kk: warning: sync failed for %s: %v\n", rel, err)
			}
		}
	}

	return nil
}

func (a App) pointerForMaterialize(rel string) (core.Pointer, bool, error) {
	p, ok, err := pointerFromWorkingFile(a.Root, rel)
	if err == nil {
		return p, ok, nil
	}
	client := git.New(a.Root)
	content, headErr := client.ShowHeadFile(rel)
	if headErr != nil {
		return core.Pointer{}, false, headErr
	}
	p, ok = core.ParsePointerText(content)
	return p, ok, nil
}

func (a App) downloadObject(p core.Pointer, localPath string, verbose bool) (downloadResult, error) {
	cfg, err := core.ReadConfig(a.Root)
	if err != nil {
		return downloadResult{}, err
	}
	info, err := core.ReadRepoInfo(a.Root)
	if err != nil {
		return downloadResult{}, err
	}
	targets := cfg.PullRemotes()
	if len(targets) == 0 {
		return downloadResult{}, fmt.Errorf("object %s is missing locally and no pull-enabled remotes are configured", p.OID)
	}
	if verbose {
		fmt.Printf("kk: object %s not in local cache; checking %d pull remote(s)\n", p.OID, len(targets))
	}
	var checked []string
	var unavailable []string
	for i, target := range targets {
		if verbose {
			fmt.Printf("kk: [%d/%d] checking remote %s\n", i+1, len(targets), target.Name)
		}
		driver, err := remote.New(target.Name, target.Config)
		if err != nil {
			if verbose {
				fmt.Printf("kk: remote %s unavailable: %v\n", target.Name, err)
			}
			unavailable = append(unavailable, target.Name)
			continue
		}
		ok, err := driver.HasObject(info, p)
		if err != nil {
			if verbose {
				fmt.Printf("kk: remote %s check failed: %v\n", target.Name, err)
			}
			unavailable = append(unavailable, target.Name)
			continue
		}
		checked = append(checked, target.Name)
		if !ok {
			if verbose {
				fmt.Printf("kk: remote %s does not have %s\n", target.Name, p.OID)
			}
			continue
		}
		if verbose {
			fmt.Printf("kk: downloading %s from %s\n", p.OID, target.Name)
		}
		if err := driver.GetObject(info, p, localPath); err != nil {
			if verbose {
				fmt.Printf("kk: download from %s failed: %v\n", target.Name, err)
			}
			continue
		}
		if verbose {
			fmt.Printf("downloaded %s from %s\n", p.OID, target.Name)
		}
		return downloadResult{Remote: target.Name}, nil
	}
	if len(checked) > 0 {
		return downloadResult{}, fmt.Errorf("object %s is missing locally; checked pull remotes: %s", p.OID, strings.Join(checked, ", "))
	}
	return downloadResult{}, fmt.Errorf("object %s is missing locally and no pull remote could be used: %s", p.OID, strings.Join(unavailable, ", "))
}

func (a App) Dematerialize(files []string) error {
	if len(files) == 0 {
		return fmt.Errorf("usage: kk dematerialize <file...>")
	}
	client := git.New(a.Root)
	for _, file := range files {
		rel, abs, err := safeRepoPath(a.Root, file)
		if err != nil {
			return err
		}
		file = rel
		content, err := client.ShowHeadFile(file)
		if err != nil {
			return err
		}
		headPointer, ok := core.ParsePointerText(content)
		if !ok {
			return fmt.Errorf("%s is not a kk pointer in HEAD", file)
		}
		current, ok, err := pointerFromWorkingFile(a.Root, file)
		if err == nil && ok {
			if current.OID == headPointer.OID && current.Size == headPointer.Size {
				fmt.Println("already dematerialized", file)
				continue
			}
			return fmt.Errorf("working pointer for %s does not match HEAD", file)
		}
		oid, size, err := storage.HashFile(abs)
		if err != nil {
			return err
		}
		if oid != headPointer.OID || size != headPointer.Size {
			return fmt.Errorf("refusing to dematerialize modified file: %s", file)
		}
		if err := client.Run("restore", "--", file); err != nil {
			return err
		}
		fmt.Println("dematerialized", file)
	}
	return nil
}

func (a App) runWithTemporaryDematerialization(client git.Client, action func() error) error {
	cleanMaterialized, err := a.findCleanMaterializedFiles(client)
	if err != nil {
		fmt.Printf("kk: warning: failed to scan for clean materialized files: %v\n", err)
	}

	if len(cleanMaterialized) > 0 {
		var filesToRestore []string
		for file := range cleanMaterialized {
			filesToRestore = append(filesToRestore, file)
		}
		if restoreErr := client.Run(append([]string{"restore", "--"}, filesToRestore...)...); restoreErr != nil {
			return fmt.Errorf("failed to temporarily dematerialize files: %w", restoreErr)
		}
	}

	actionErr := action()

	if actionErr != nil {

		for restoredFile := range cleanMaterialized {
			_ = a.materialize(restoredFile, false, false)
		}
	}
	return actionErr
}

var syncMutex sync.Mutex

func (a App) SyncObjectAcrossRemotes(p core.Pointer, verbose bool) error {
	cfg, err := core.ReadConfig(a.Root)
	if err != nil {
		return err
	}
	info, err := core.ReadRepoInfo(a.Root)
	if err != nil {
		return err
	}
	targets, err := cfg.PushRemotes(nil, true)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return nil
	}

	store := storage.New(a.Root)
	if !store.HasObject(p) {
		return fmt.Errorf("object %s not present locally in cache, cannot sync", p.OID)
	}
	localPath := store.ObjectPath(p.OID)

	for _, target := range targets {
		driver, err := remote.New(target.Name, target.Config)
		if err != nil {
			if verbose {
				fmt.Printf("kk: sync: remote %s initialization failed: %v\n", target.Name, err)
			}
			continue
		}

		syncMutex.Lock()
		manifest, err := driver.ReadManifest(info)
		if err != nil {
			manifest = remote.EmptyManifest(info)
		}

		exists := false
		for _, mObj := range manifest.Objects {
			if mObj.OID == p.OID {
				exists = true
				break
			}
		}

		if !exists {
			exists, err = driver.HasObject(info, p)
			if err != nil {
				exists = false
			}
		}

		if exists {
			syncMutex.Unlock()
			continue
		}

		syncMutex.Unlock()

		if verbose {
			fmt.Printf("kk: sync: replicating %s to remote %s...\n", p.OID[:12], target.Name)
		}

		if err := driver.PutObject(info, p, localPath); err != nil {
			if verbose {
				fmt.Printf("kk: sync: uploading %s to remote %s failed: %v\n", p.OID[:12], target.Name, err)
			}
			continue
		}

		syncMutex.Lock()
		manifest, err = driver.ReadManifest(info)
		if err != nil {
			manifest = remote.EmptyManifest(info)
		}
		manifest = remote.UpsertManifestObject(manifest, p)
		if err := driver.WriteManifest(info, manifest); err != nil {
			if verbose {
				fmt.Printf("kk: sync: failed to write manifest for %s: %v\n", target.Name, err)
			}
		} else if verbose {
			fmt.Printf("kk: sync: successfully replicated %s to %s\n", p.OID[:12], target.Name)
		}
		syncMutex.Unlock()
	}

	return nil
}
