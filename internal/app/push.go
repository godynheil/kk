package app

import (
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

func (a App) Push(args []string) error {
	client := git.New(a.Root)
	if err := client.EnsureRepository(); err != nil {
		return err
	}
	fmt.Printf("kk: processing push on %s...\n", currentBranchLabel(client))
	remoteNames, all, syncWorkingDir, workers, gitArgs := ParsePushArgs(args)

	cfg, err := core.ReadConfig(a.Root)
	if err != nil {
		return err
	}
	if upgraded, upgradeErr := a.enableLegacyClonedStorageOriginPush(cfg); upgradeErr != nil {
		return upgradeErr
	} else if upgraded {
		cfg, err = core.ReadConfig(a.Root)
		if err != nil {
			return err
		}
	}
	if len(cfg.Remotes) == 0 && len(gitArgs) == 0 {
		return fmt.Errorf("no push destination configured; run 'kk setup gdrive' or 'kk remote add ...' first")
	}

	a.pushGitRemotes(client)

	if err := a.pushObjects(remoteNames, all, syncWorkingDir, workers); err != nil {
		return err
	}

	if hasNoGitRemote(cfg) {
		targets, terr := cfg.PushRemotes(remoteNames, all)
		if terr == nil && len(targets) > 0 {
			if histErr := a.pushHistory(targets); histErr != nil {
				fmt.Printf("kk: warning: history push failed: %v\n", histErr)
			}
		}
	}

	if len(gitArgs) > 0 {
		if err := runGitPush(client, gitArgs); err != nil {
			if strings.HasPrefix(err.Error(), "no KK history push destination configured") {
				return err
			}
			return fmt.Errorf(
				"push failed; if the remote has new commits run 'kk pull' first, then retry 'kk push': %w",
				err,
			)
		}
	}

	return nil
}

func (a App) enableLegacyClonedStorageOriginPush(cfg core.LocalConfig) (bool, error) {
	if cfg.DefaultRemote != "origin" {
		return false, nil
	}
	origin, ok := cfg.Remotes["origin"]
	if !ok || origin.Push || origin.Type == "git" {
		return false, nil
	}
	for _, remoteCfg := range cfg.Remotes {
		if remoteCfg.Type != "git" && remoteCfg.Push {
			return false, nil
		}
	}
	origin.Push = true
	cfg.Remotes["origin"] = origin
	if err := core.WriteConfig(a.Root, cfg); err != nil {
		return false, fmt.Errorf("enabling push for cloned storage origin: %w", err)
	}
	fmt.Println("kk: enabled push for cloned storage remote origin")
	return true, nil
}

func runGitPush(client git.Client, gitArgs []string) error {
	stdout, stderr, err := client.Combined(append([]string{"push"}, gitArgs...)...)
	if err == nil {
		if stdout != "" {
			fmt.Print(stdout)
		}
		if stderr != "" {
			_, _ = fmt.Fprint(os.Stderr, stderr)
		}
		return nil
	}
	if strings.Contains(stderr, "No configured push destination") {
		return fmt.Errorf("no KK history push destination configured; add a KK remote with 'kk setup gdrive' or pass an explicit push target")
	}
	if stdout != "" {
		fmt.Print(stdout)
	}
	if stderr != "" {
		_, _ = fmt.Fprint(os.Stderr, sanitizeKKErrorText(stderr))
	}
	return fmt.Errorf("%s", sanitizeKKErrorText(err.Error()))
}

func sanitizeKKErrorText(msg string) string {
	msg = strings.ReplaceAll(msg, "Git", "KK")
	msg = strings.ReplaceAll(msg, "git", "kk")
	return msg
}

func (a App) pushGitRemotes(client git.Client) {
	cfg, err := core.ReadConfig(a.Root)
	if err != nil {
		return
	}
	gitRemotes := cfg.GitRemotes()
	if len(gitRemotes) == 0 {
		return
	}
	for _, r := range gitRemotes {
		if !r.Config.Push {
			continue
		}
		fmt.Printf("kk: [%s] syncing pointer history to git remote...\n", r.Name)
		stdout, stderr, pushErr := pushGitRemote(client, r.Name)
		if pushErr != nil {
			if isNoUpstreamError(stderr) {
				fmt.Printf("kk: warning: [%s] no upstream branch set — pointer history not synced.\n", r.Name)
				fmt.Printf("kk:          To set one, run once:\n")
				fmt.Printf("kk:          git --git-dir=.kk/git push -u %s <branch>\n", r.Name)
			} else {
				msg := strings.TrimSpace(stderr)
				if msg == "" {
					msg = pushErr.Error()
				}
				fmt.Printf("kk: warning: [%s] pointer history sync failed: %s\n", r.Name, msg)
			}
			fmt.Printf("kk: continuing without pointer sync for %s\n", r.Name)
			continue
		}
		if stdout != "" {
			fmt.Print(stdout)
		}
		if stderr != "" {
			_, _ = fmt.Fprint(os.Stderr, stderr)
		}
		fmt.Printf("kk: [%s] pointer history synced\n", r.Name)
	}
}

func pushGitRemote(client git.Client, remoteName string) (stdout string, stderr string, err error) {
	stdout, stderr, err = client.Combined("push", remoteName)
	if err == nil || !isNoUpstreamError(stderr) {
		return stdout, stderr, err
	}

	branch := client.CurrentBranch()
	if branch == "" {
		return stdout, stderr, err
	}

	fallbackStdout, fallbackStderr, fallbackErr := client.Combined("push", "-u", remoteName, branch)
	return fallbackStdout, fallbackStderr, fallbackErr
}

func isNoUpstreamError(msg string) bool {
	return strings.Contains(msg, "no upstream branch") ||
		strings.Contains(msg, "no tracking information") ||
		strings.Contains(msg, "No configured push destination") ||
		strings.Contains(msg, "has no tracking")
}

func (a App) pushObjects(remoteNames []string, all bool, syncWorkingDir bool, flagWorkers int) error {
	cfg, err := core.ReadConfig(a.Root)
	if err != nil {
		return err
	}
	targets, err := cfg.PushRemotes(remoteNames, all)
	if err != nil {

		if len(cfg.Remotes) == 0 || cfg.HasOnlyGitRemotes() {
			if len(cfg.GitRemotes()) > 0 {
				fmt.Println("kk: no object remotes configured — pointer history synced to git remote(s)")
			} else {
				fmt.Println("kk: no object remotes configured; skipping object upload")
			}
			return nil
		}
		return err
	}
	if len(targets) == 0 {
		if len(cfg.GitRemotes()) > 0 {
			fmt.Println("kk: no object remotes to push to — pointer history synced to git remote(s)")
		} else {
			fmt.Println("kk: no push-enabled object remotes found; skipping object upload")
		}
		return nil
	}
	store := storage.New(a.Root)
	info, err := core.ReadRepoInfo(a.Root)
	if err != nil {
		return err
	}

	w := resolveWorkers(flagWorkers)

	state, err := readPushState(a.Root)
	if err != nil {
		return fmt.Errorf("reading push state: %w", err)
	}
	gitClient := git.New(a.Root)
	headCommit := ""
	if gitClient.HasHEAD() {
		headCommit, err = gitClient.HeadCommit()
		if err != nil {
			return fmt.Errorf("reading current commit: %w", err)
		}
		headCommit = strings.TrimSpace(headCommit)
	}

	for _, target := range targets {
		driver, err := remote.New(target.Name, target.Config)
		if err != nil {
			return fmt.Errorf("remote %s: failed to initialise: %w", target.Name, err)
		}
		if err := driver.Check(); err != nil {
			return fmt.Errorf("remote %s: check failed: %w", target.Name, err)
		}

		if err := checkProjectConflict(driver, info); err != nil {
			return fmt.Errorf("remote %s: %w", target.Name, err)
		}

		filesToSync, objects, fullSnapshot, err := a.pushInputsForTarget(target.Name, state, syncWorkingDir, headCommit)
		if err != nil {
			return err
		}
		if syncWorkingDir {
			fmt.Printf("kk: [%s] --sync-working-dir: %d working-directory file(s) will be checked\n", target.Name, len(filesToSync))
		} else if fullSnapshot {
			fmt.Printf("kk: [%s] first push or unknown remote state: %d committed file(s) will be checked\n", target.Name, len(filesToSync))
		} else {
			fmt.Printf("kk: [%s] %d changed file(s) since last push will be checked\n", target.Name, len(filesToSync))
		}

		syncStats := remote.SyncStats{}
		if len(filesToSync) == 0 {
			fmt.Printf("kk: [%s] no changed files to sync\n", target.Name)
		} else {
			err = func() error {
				syncRoot, prepareErr := a.prepareSyncRoot(filesToSync, gitClient)
				if prepareErr != nil {
					return prepareErr
				}
				defer func() {
					_ = os.RemoveAll(syncRoot)
				}()

				fmt.Printf("kk: [%s] syncing %d file(s) (%d workers)...\n", target.Name, len(filesToSync), w)
				syncBar := newMultiProgressBar(len(filesToSync), "syncing", w)
				fileStart := func(workerID int, file string) {
					syncBar.SetSlot(workerID, file)
				}
				fileDone := func(workerID, done, total int, file string) {
					syncBar.Complete(workerID, file)
				}
				var syncErr error
				syncStats, syncErr = driver.SyncProjectFiles(info, syncRoot, filesToSync, w, fileStart, fileDone)
				if syncErr != nil {
					syncBar.Finish("")
					return fmt.Errorf("remote %s: file sync failed: %w", target.Name, syncErr)
				}
				if syncStats.Changed == 0 {
					syncBar.Finish(fmt.Sprintf("kk: [%s] files already current — %d checked", target.Name, len(filesToSync)))
				} else {
					syncBar.Finish(fmt.Sprintf("kk: [%s] synced %d file(s), %d already current",
						target.Name, syncStats.Changed, syncStats.Skipped))
				}
				return nil
			}()
			if err != nil {
				return err
			}
		}

		manifest := core.Manifest{}
		if len(objects) > 0 {
			var err error
			manifest, err = driver.ReadManifest(info)
			if err != nil {
				return fmt.Errorf("remote %s: reading manifest: %w", target.Name, err)
			}
		}

		total := len(objects)
		var toUpload []core.RequiredObject
		skipped := 0
		if total > 0 {
			fmt.Printf("kk: [%s] checking %d object(s)...\n", target.Name, total)
			checkBar := newProgressBar(total, "checking")

			remoteObjects := make(map[string]bool, len(manifest.Objects))
			for _, mObj := range manifest.Objects {
				remoteObjects[mObj.OID] = true
			}

			type checkJob struct {
				idx int
				obj core.RequiredObject
			}
			type checkResult struct {
				idx    int
				exists bool
				pulled bool
				err    error
			}

			jobs := make(chan checkJob, total)
			results := make(chan checkResult, total)

			var checkWg sync.WaitGroup
			for i := 0; i < w; i++ {
				checkWg.Add(1)
				go func() {
					defer checkWg.Done()
					for job := range jobs {
						obj := job.obj
						p := core.Pointer{OID: obj.OID, Size: obj.Size}
						exists := remoteObjects[p.OID]
						var err error
						if !exists {
							exists, err = driver.HasObject(info, p)
							if err != nil {
								results <- checkResult{idx: job.idx, err: fmt.Errorf("remote check failed for %s: %w", p.OID, err)}
								continue
							}
						}

						if exists {
							results <- checkResult{idx: job.idx, exists: true}
						} else {
							pulled := false
							if !store.HasObject(p) {
								downloaded, downloadErr := a.pullObjectForPush(p, store, info)
								if downloadErr != nil {
									results <- checkResult{idx: job.idx, err: fmt.Errorf("object %s is missing locally and on remote (pull failed: %w)", p.OID, downloadErr)}
									continue
								}
								if !downloaded {
									results <- checkResult{idx: job.idx, err: fmt.Errorf("object %s is missing locally and could not be pulled from any remote", p.OID)}
									continue
								}
								pulled = true
							}
							results <- checkResult{idx: job.idx, exists: false, pulled: pulled}
						}
					}
				}()
			}

			for i, obj := range objects {
				jobs <- checkJob{idx: i, obj: obj}
			}
			close(jobs)

			go func() {
				checkWg.Wait()
				close(results)
			}()

			resultsSlice := make([]checkResult, total)
			var checkErr error
			for r := range results {
				if r.err != nil {
					if checkErr == nil {
						checkErr = r.err
					}
					continue
				}
				resultsSlice[r.idx] = r
			}

			if checkErr != nil {
				checkBar.Finish("")
				return fmt.Errorf("remote %s: %w", target.Name, checkErr)
			}

			for i, r := range resultsSlice {
				obj := objects[i]
				checkBar.Tick(obj.OID[:12])
				if r.exists {
					p := core.Pointer{OID: obj.OID, Size: obj.Size}
					manifest = remote.UpsertManifestObject(manifest, p)
					skipped++
				} else {
					if r.pulled {
						fmt.Printf("kk: [%s] pulled missing object %s from remote\n", target.Name, obj.OID[:12])
					}
					toUpload = append(toUpload, obj)
				}
			}

			checkBar.Finish(fmt.Sprintf("kk: [%s] %d to upload, %d already on remote",
				target.Name, len(toUpload), skipped))
		}

		uploaded := 0
		if len(toUpload) > 0 {
			fmt.Printf("kk: [%s] uploading %d object(s) (%d workers)...\n",
				target.Name, len(toUpload), w)
			uploadBar := newMultiProgressBar(len(toUpload), "uploading", w)

			uploadedObjs, uploadErr := runObjectUploads(driver, info, toUpload, store, uploadBar, w)
			if uploadErr != nil {
				uploadBar.Finish("")
				return fmt.Errorf("remote %s: %w", target.Name, uploadErr)
			}
			for _, obj := range uploadedObjs {
				manifest = remote.UpsertManifestObject(manifest, core.Pointer{OID: obj.OID, Size: obj.Size})
			}
			uploaded = len(uploadedObjs)
			uploadBar.Finish(fmt.Sprintf("kk: [%s] uploaded %d object(s)", target.Name, uploaded))
		}

		if total > 0 {
			fmt.Printf("kk: [%s] objects complete — %d uploaded, %d skipped\n",
				target.Name, uploaded, skipped)
		}

		if total > 0 {
			if err := driver.WriteManifest(info, manifest); err != nil {
				return fmt.Errorf("remote %s: writing manifest: %w", target.Name, err)
			}
		}
		if syncStats.Changed == 0 && uploaded == 0 {
			fmt.Printf("kk: [%s] already up to date\n", target.Name)
		}
		if !syncWorkingDir && headCommit != "" {
			state.Remotes[target.Name] = remotePushState{HeadCommit: headCommit, UpdatedAt: time.Now().UTC()}
			if err := writePushState(a.Root, state); err != nil {
				return fmt.Errorf("writing push state: %w", err)
			}
		}
	}
	return nil
}

func (a App) pushInputsForTarget(targetName string, state pushState, syncWorkingDir bool, headCommit string) ([]string, []core.RequiredObject, bool, error) {
	if syncWorkingDir {
		files, err := remote.WalkProjectFiles(a.Root)
		if err != nil {
			return nil, nil, false, fmt.Errorf("listing project files: %w", err)
		}
		objects, err := a.requiredObjectsForHeadFiles(files)
		return files, objects, true, err
	}

	gitClient := git.New(a.Root)
	previous := strings.TrimSpace(state.Remotes[targetName].HeadCommit)
	fullSnapshot := previous == "" || headCommit == "" || !gitClient.IsAncestor(previous, headCommit)

	if fullSnapshot {
		headFiles, err := gitClient.HeadFiles()
		if err != nil {
			return nil, nil, true, fmt.Errorf("reading committed files: %w", err)
		}
		files := remote.CommittedProjectFiles(a.Root, headFiles)
		objects, err := a.requiredObjectsForHeadFiles(files)
		return files, objects, true, err
	}

	changed, err := gitClient.ChangedFiles(previous, headCommit)
	if err != nil {
		return nil, nil, false, fmt.Errorf("reading changed files: %w", err)
	}
	files := remote.ExistingProjectFiles(a.Root, changed)
	for _, meta := range []string{".kk/repo.json", ".kk/config.json", ".kk/tracks.json"} {
		if _, err := os.Stat(filepath.Join(a.Root, filepath.FromSlash(meta))); err == nil {
			files = append(files, meta)
		}
	}
	objects, err := a.requiredObjectsForHeadFiles(files)
	return files, objects, false, err
}

func runObjectUploads(
	driver remote.Driver,
	info core.RepoInfo,
	toUpload []core.RequiredObject,
	store storage.Store,
	bar *MultiProgressBar,
	w int,
) ([]core.RequiredObject, error) {
	if w <= 0 {
		w = 1
	}
	type uploadJob struct {
		workerID int
		obj      core.RequiredObject
	}
	type uploadResult struct {
		obj core.RequiredObject
		err error
	}

	jobs := make(chan uploadJob)
	results := make(chan uploadResult, len(toUpload))

	var wg sync.WaitGroup
	for i := 0; i < w; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				p := core.Pointer{OID: job.obj.OID, Size: job.obj.Size}
				bar.SetSlot(job.workerID, fmt.Sprintf("%s (%s)", job.obj.OID[:12], formatBytes(job.obj.Size)))
				err := driver.PutObject(info, p, store.ObjectPath(job.obj.OID))
				bar.Complete(job.workerID, fmt.Sprintf("%s (%s)", job.obj.OID[:12], formatBytes(job.obj.Size)))
				results <- uploadResult{obj: job.obj, err: err}
			}
		}()
	}

	go func() {
		for idx, obj := range toUpload {
			jobs <- uploadJob{workerID: idx % w, obj: obj}
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	var uploaded []core.RequiredObject
	var firstErr error
	for r := range results {
		if r.err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("upload failed for %s: %w", r.obj.OID, r.err)
			}
			continue
		}
		uploaded = append(uploaded, r.obj)
	}
	return uploaded, firstErr
}

func (a App) pullObjectForPush(p core.Pointer, store storage.Store, info core.RepoInfo) (bool, error) {
	cfg, err := core.ReadConfig(a.Root)
	if err != nil {
		return false, err
	}
	targets := cfg.PullRemotes()
	if len(targets) == 0 {
		return false, nil
	}

	localPath := store.ObjectPath(p.OID)
	for _, target := range targets {
		driver, err := remote.New(target.Name, target.Config)
		if err != nil {
			continue
		}
		ok, err := driver.HasObject(info, p)
		if err != nil || !ok {
			continue
		}
		if err := driver.GetObject(info, p, localPath); err != nil {
			continue
		}
		if err := store.VerifyObject(p); err != nil {
			continue
		}
		return true, nil
	}
	return false, nil
}

func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}

	size := float64(n)
	exp := 0
	for size >= float64(unit) && exp < 6 {
		size /= float64(unit)
		exp++
	}
	return fmt.Sprintf("%.1f %cB", size, "KMGTPE"[exp])
}

func checkProjectConflict(driver remote.Driver, info core.RepoInfo) error {
	existing, ok, err := driver.ReadRemoteRepoInfo(info)
	if err != nil {
		return fmt.Errorf("reading remote project info: %w", err)
	}
	if ok && existing.RepoID != info.RepoID {
		return fmt.Errorf(
			"remote folder %q already belongs to project %q (repo_id: %s); rename your project or use a different remote root",
			info.Name, existing.Name, existing.RepoID,
		)
	}
	return nil
}

func ParsePushArgs(args []string) (remoteNames []string, all bool, syncWorkingDir bool, workers int, gitArgs []string) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all-remotes":
			all = true
		case "--sync-working-dir":
			syncWorkingDir = true
		case "--workers":
			if i+1 < len(args) {
				if n, err := strconv.Atoi(args[i+1]); err == nil && n > 0 {
					workers = n
				}
				i++
			}
		case "--remote":
			if i+1 < len(args) {
				remoteNames = append(remoteNames, args[i+1])
				i++
			}
		default:
			if strings.HasPrefix(args[i], "--remote=") {
				name := strings.TrimSpace(strings.TrimPrefix(args[i], "--remote="))
				if name != "" {
					remoteNames = append(remoteNames, name)
				}
				continue
			}
			if strings.HasPrefix(args[i], "--workers=") {
				val := strings.TrimPrefix(args[i], "--workers=")
				if n, err := strconv.Atoi(val); err == nil && n > 0 {
					workers = n
				}
				continue
			}
			gitArgs = append(gitArgs, args[i])
		}
	}
	return remoteNames, all, syncWorkingDir, workers, gitArgs
}

func (a App) prepareSyncRoot(files []string, client git.Client) (string, error) {
	tracks, err := core.ReadTracks(a.Root)
	if err != nil {
		return "", err
	}
	tempRoot, err := os.MkdirTemp(filepath.Join(a.Root, ".kk", "tmp"), "push-sync-*")
	if err != nil {
		return "", err
	}

	success := false
	defer func() {
		if !success {
			_ = os.RemoveAll(tempRoot)
		}
	}()

	for _, rel := range files {
		destPath := filepath.Join(tempRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(destPath), 0o750); err != nil {
			return "", err
		}
		if core.ShouldTrack(rel, tracks) {
			var pointerText string
			if indexBytes, err := client.ShowIndexFile(rel); err == nil && len(indexBytes) > 0 {
				if _, ok := core.ParsePointerBytes(indexBytes); ok {
					pointerText = string(indexBytes)
				}
			}
			if pointerText == "" {
				if headText, err := client.ShowHeadFile(rel); err == nil && headText != "" {
					if _, ok := core.ParsePointerText(headText); ok {
						pointerText = headText
					}
				}
			}
			if pointerText == "" {
				if workingPointer, ok, _ := pointerFromWorkingFile(a.Root, rel); ok {
					pointerText = core.FormatPointer(workingPointer)
				} else {
					filePath := filepath.Join(a.Root, filepath.FromSlash(rel))
					oid, size, err := storage.HashFile(filePath)
					if err != nil {
						return "", err
					}
					pointerText = core.FormatPointer(core.Pointer{OID: oid, Size: size})
				}
			}
			if err := os.WriteFile(destPath, []byte(pointerText), 0o644); err != nil {
				return "", err
			}
		} else {
			srcPath := filepath.Join(a.Root, filepath.FromSlash(rel))
			if err := linkOrCopy(srcPath, destPath); err != nil {
				return "", err
			}
		}
	}
	success = true
	return tempRoot, nil
}

func linkOrCopy(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to copy symlink: %s", src)
	}
	if err := os.Link(src, dst); err == nil {
		return nil
	}
	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		_ = in.Close()
	}()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()
	_, err = io.Copy(out, in)
	return err
}
