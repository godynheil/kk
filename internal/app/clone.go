package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/godynheil/kk/internal/core"
	"github.com/godynheil/kk/internal/gdrive"
	"github.com/godynheil/kk/internal/git"
	"github.com/godynheil/kk/internal/remote"
)

func (a App) Clone(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: kk clone <remote-spec> [<dest>] [--remote-name <n>] [--pull] [--history] [--account <profile>] [--here] [--force]\n" +
			"  remote-spec examples:\n" +
			"    git:https://github.com/org/repo.git\n" +
			"    local:/Volumes/NAS/KK/MyGame\n" +
			"    rclone:gdrive:KK/MyGame\n" +
			"    drive:<project-folder-id>\n" +
			"  --account  rclone or gdrive profile name to use (e.g. 'work', 'personal')\n" +
			"             omit to use 'default'; list profiles with 'kk accounts'\n" +
			"  --history  Download and restore the full commit history from the remote.\n" +
			"             Requires the project to have been pushed at least once with\n" +
			"             a kk build that supports history bundles.\n" +
			"  --here     Clone into the current directory instead of a new subdirectory.\n" +
			"             The directory must be empty except for the kk binary itself.\n" +
			"  --force    Combined with --here: skip the non-empty directory check\n" +
			"  --branch   Git clone only: clone a specific remote branch")
	}

	spec, dest, remoteName, account, branch, doPull, withHistory, here, force, flagWorkers, err := parseCloneArgs(args)
	if err != nil {
		return err
	}

	if here {
		if dest != "" {
			return fmt.Errorf("kk clone: --here and an explicit destination are mutually exclusive")
		}
		cwd, cwdErr := os.Getwd()
		if cwdErr != nil {
			return fmt.Errorf("kk clone: getting current directory: %w", cwdErr)
		}
		dest = cwd
		if checkErr := checkHereDirectory(dest, force); checkErr != nil {
			return checkErr
		}
	}

	remoteType, remoteRoot, projectName, err := parseRemoteSpec(spec)
	if err != nil {
		return err
	}

	cfg, err := a.buildCloneRemoteConfig(remoteType, remoteRoot, account)
	if err != nil {
		return err
	}

	if remoteType == "git" {
		return a.cloneFromGit(remoteRoot, dest, remoteName, account, branch, doPull, withHistory, here, force, flagWorkers, cfg)
	}

	if remoteType == "drive" {
		driveClient := gdrive.NewClient(cfg.DriveAuthPath)
		folder, resolveErr := driveClient.GetFolder(context.Background(), cfg.DriveFolderID)
		if resolveErr != nil {
			return fmt.Errorf("kk clone: resolving drive folder %q: %w", cfg.DriveFolderID, explainDriveFolderResolveError(resolveErr, cfg.DriveAuthPath))
		}
		if folder.MimeType != "application/vnd.google-apps.folder" {
			return fmt.Errorf("kk clone: %q is not a Drive folder (mimeType: %s)", cfg.DriveFolderID, folder.MimeType)
		}
		projectName = folder.Name
		fmt.Printf("kk: resolved folder → project=%q  folder-id=%s\n", projectName, cfg.DriveFolderID)
	}

	if remoteName == "" {
		remoteName = "origin"
	}
	cfg.DisplayName = remoteName
	if cfg.Provider == "" {
		cfg.Provider = remoteType
	}

	driver, err := remote.New(remoteName, cfg)
	if err != nil {
		return fmt.Errorf("kk clone: building remote: %w", err)
	}

	fmt.Printf("kk: checking remote %q (%s) ...\n", remoteName, remoteType)
	if err := driver.Check(); err != nil {
		return fmt.Errorf("kk clone: remote not accessible: %w", err)
	}
	fmt.Printf("kk: remote %q is accessible\n", remoteName)

	stub := core.RepoInfo{Name: projectName}
	info, ok, err := driver.ReadRemoteRepoInfo(stub)
	if err != nil {
		return fmt.Errorf("kk clone: reading remote repo info: %w", err)
	}
	if !ok {
		return fmt.Errorf("kk clone: project %q not found on remote %q\n"+
			"  (expected files at <remote-root>/%s/.kk/repo.json)",
			projectName, remoteName, projectName)
	}
	fmt.Printf("kk: found project %q (repo_id: %s)\n", info.Name, info.RepoID)

	if dest == "" {
		dest = projectName
	}
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return fmt.Errorf("kk clone: resolving destination: %w", err)
	}
	if !here {
		if entries, _ := os.ReadDir(absDest); len(entries) > 0 {
			return fmt.Errorf("kk clone: destination %q already exists and is not empty", absDest)
		}
	}
	if err := os.MkdirAll(absDest, 0o750); err != nil {
		return fmt.Errorf("kk clone: creating destination: %w", err)
	}

	fmt.Printf("kk: downloading project files from %q ...\n", remoteName)
	w := resolveWorkers(flagWorkers)
	var (
		downloadBar   *MultiProgressBar
		downloadTotal int
	)
	onFiles := func(total int) {
		downloadTotal = total
		if total > 0 {
			slots := min(w, total)
			fmt.Printf("kk: [%s] downloading %d file(s) (%d workers)...\n", remoteName, total, slots)
			downloadBar = newMultiProgressBar(total, "downloading", slots)
		}
	}
	onStart := func(workerID int, file string) {
		if downloadBar != nil {
			downloadBar.SetSlot(workerID, file)
		}
	}
	onDone := func(workerID, done, total int, file string) {
		if downloadBar != nil {
			downloadBar.Complete(workerID, file)
		}
	}
	if err := driver.DownloadProjectFiles(info, absDest, w, onFiles, onStart, onDone); err != nil {
		if downloadBar != nil {
			downloadBar.Finish("")
		}
		return fmt.Errorf("kk clone: downloading project files: %w", err)
	}
	if downloadBar != nil {
		downloadBar.Finish(fmt.Sprintf("kk: [%s] downloaded %d file(s)", remoteName, downloadTotal))
	} else {
		fmt.Println("kk: project files downloaded")
	}

	for _, dir := range []string{core.ObjectDir, core.TmpDir, core.LogDir, core.KKGitDir} {
		if err := os.MkdirAll(filepath.Join(absDest, dir), 0o750); err != nil {
			return fmt.Errorf("kk clone: creating %s: %w", dir, err)
		}
	}
	if err := ensureGitExclude(absDest); err != nil {
		return fmt.Errorf("kk clone: history exclude: %w", err)
	}

	gitClient := git.New(absDest)
	if err := gitClient.InitMain(); err != nil {
		return fmt.Errorf("kk clone: history init: %w", err)
	}

	freshCfg := core.DefaultConfig()
	freshCfg.Remotes[remoteName] = cfg
	freshCfg.DefaultRemote = remoteName
	if err := core.WriteConfig(absDest, freshCfg); err != nil {
		return fmt.Errorf("kk clone: writing config: %w", err)
	}

	if err := core.WriteRepoInfo(absDest, info); err != nil {
		return fmt.Errorf("kk clone: writing repo info: %w", err)
	}

	fmt.Println("kk: staging files ...")
	if withHistory {
		if cloneErr := cloneRestoreHistory(driver, info, absDest, gitClient); cloneErr != nil {
			fmt.Printf("kk: warning: could not restore history: %v\n  falling back to initial snapshot\n", cloneErr)
			withHistory = false
		} else {
			_ = gitClient.Run("clean", "-fd")
		}
	}
	if !withHistory {
		if err := gitClient.Run("add", "."); err != nil {
			return fmt.Errorf("kk clone: stage initial snapshot: %w", err)
		}
		if err := gitClient.Run("commit", "--allow-empty", "-m", "kk clone: initial snapshot"); err != nil {
			return fmt.Errorf("kk clone: commit initial snapshot: %w", err)
		}
	}

	if err := syncProjectRegistry(absDest); err != nil {
		fmt.Printf("kk: warning: could not register project: %v\n", err)
	}

	clonedApp := App{Root: absDest}
	fmt.Printf("\nkk: clone complete -> %s\n", absDest)

	if doPull {
		fmt.Println("kk: --pull: materialising large files ...")
		objects, listErr := clonedApp.requiredObjectsInHEAD()
		if listErr != nil {
			return fmt.Errorf("kk clone: listing objects: %w", listErr)
		}

		if len(objects) > 0 {
			bar := newMultiProgressBar(len(objects), "pulling", min(w, len(objects)))

			type matResult struct {
				slotID int
				path   string
				err    error
			}

			jobs := make(chan core.RequiredObject)
			results := make(chan matResult, len(objects))

			var wg sync.WaitGroup
			pullWorkers := min(w, len(objects))
			for i := 0; i < pullWorkers; i++ {
				slotID := i
				wg.Add(1)
				go func() {
					defer wg.Done()
					for obj := range jobs {
						bar.SetSlot(slotID, obj.Path)
						err := clonedApp.materialize(obj.Path, false, false)
						results <- matResult{slotID: slotID, path: obj.Path, err: err}
					}
				}()
			}

			go func() {
				for _, obj := range objects {
					jobs <- obj
				}
				close(jobs)
			}()
			go func() {
				wg.Wait()
				close(results)
			}()

			var matWarnings []string
			for r := range results {
				bar.Complete(r.slotID, r.path)
				if r.err != nil {
					matWarnings = append(matWarnings, fmt.Sprintf("%s: %v", r.path, r.err))
				}
			}
			succeeded := len(objects) - len(matWarnings)
			bar.Finish(fmt.Sprintf("kk: materialised %d/%d object(s)", succeeded, len(objects)))
			for _, w := range matWarnings {
				fmt.Printf("kk: warning: could not materialise %s\n", w)
			}
		}
		fmt.Println("kk: materialisation complete")
	} else {
		fmt.Printf("kk: run 'cd %s && kk fsck'          to check large-file status\n", absDest)
		fmt.Println("    run 'kk pull-file <file>'     to materialise individual files")
		fmt.Println("    run 'kk pull-file .'          to materialise all pointer files at once")
		fmt.Println("    run 'kk pull-file --all'      to materialise everything (alias for .)")
	}

	return nil
}

func parseCloneArgs(args []string) (spec, dest, remoteName, account, branch string, doPull bool, withHistory bool, here bool, force bool, workers int, err error) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--remote-name":
			if err = fatalIfMissingValue(args, i, args[i]); err != nil {
				return
			}
			remoteName = args[i+1]
			i++
		case "--account":
			if err = fatalIfMissingValue(args, i, args[i]); err != nil {
				return
			}
			account = args[i+1]
			i++
		case "--branch":
			if err = fatalIfMissingValue(args, i, args[i]); err != nil {
				return
			}
			branch = args[i+1]
			i++
		case "--pull":
			doPull = true
		case "--history":
			withHistory = true
		case "--here":
			here = true
		case "--force":
			force = true
		case "--workers":
			if err = fatalIfMissingValue(args, i, args[i]); err != nil {
				return
			}
			if n, convErr := strconv.Atoi(args[i+1]); convErr == nil && n > 0 {
				workers = n
			}
			i++
		default:
			if strings.HasPrefix(args[i], "--workers=") {
				val := strings.TrimPrefix(args[i], "--workers=")
				if n, convErr := strconv.Atoi(val); convErr == nil && n > 0 {
					workers = n
				}
				continue
			}
			if strings.HasPrefix(args[i], "-") {
				err = fmt.Errorf("kk clone: unknown flag %q", args[i])
				return
			}
			if spec == "" {
				spec = args[i]
			} else if dest == "" {
				dest = args[i]
			} else {
				err = fmt.Errorf("kk clone: unexpected argument %q", args[i])
				return
			}
		}
	}
	if spec == "" {
		err = fmt.Errorf("kk clone: remote spec is required")
	}
	return
}

func checkHereDirectory(dest string, force bool) error {
	kkBinaries := map[string]bool{
		"kk":              true,
		"kk.exe":          true,
		"kk-portable.exe": true,
	}
	entries, err := os.ReadDir(dest)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("kk clone: reading destination directory: %w", err)
	}
	var blocking []string
	for _, e := range entries {
		if !kkBinaries[e.Name()] {
			blocking = append(blocking, e.Name())
		}
	}
	if len(blocking) == 0 || force {
		return nil
	}
	return fmt.Errorf(
		"kk clone: --here: directory %q is not empty\n"+
			"  unexpected file(s): %s\n"+
			"  Remove them first, or re-run with --force to clone anyway",
		dest, strings.Join(blocking, ", "),
	)
}

func parseRemoteSpec(spec string) (remType, remRoot, project string, err error) {
	switch {
	case strings.HasPrefix(spec, "git:"):
		rest := strings.TrimPrefix(spec, "git:")
		remRoot = rest
		remType = "git"
		idx := strings.LastIndex(rest, "/")
		if idx < 0 {
			err = fmt.Errorf("git spec must contain a valid URL with project name: %q", spec)
			return
		}
		project = rest[idx+1:]
		project = strings.TrimSuffix(project, ".git")

	case strings.HasPrefix(spec, "local:"):
		rest := filepath.FromSlash(strings.TrimPrefix(spec, "local:"))
		project = filepath.Base(rest)
		remRoot = rest
		remType = "local"

	case strings.HasPrefix(spec, "rclone:"):
		rest := strings.TrimPrefix(spec, "rclone:")
		idx := strings.LastIndex(rest, "/")
		if idx < 0 {
			err = fmt.Errorf("rclone spec must end with /<ProjectName>: %q", spec)
			return
		}
		project = rest[idx+1:]
		remRoot = rest
		remType = "rclone"

	case strings.HasPrefix(spec, "drive:"):
		rest := strings.TrimPrefix(spec, "drive:")
		if strings.Contains(rest, "/") {
			err = fmt.Errorf("drive spec must be just the project folder ID: %q\n"+
				"  use: drive:<project-folder-id>  (copy the ID from the Drive URL)", spec)
			return
		}
		remRoot = rest
		remType = "drive"

	default:
		err = fmt.Errorf("unsupported remote spec %q\n"+
			"  use: git:<url>\n"+
			"       local:/path/RemoteRoot/ProjectName\n"+
			"       rclone:<remote>:<path>/ProjectName\n"+
			"       drive:<project-folder-id>", spec)
	}

	if err == nil && project == "" && remType != "drive" {
		err = fmt.Errorf("could not determine project name from spec %q", spec)
	}
	if err == nil && remRoot == "" {
		err = fmt.Errorf("could not determine remote root from spec %q", spec)
	}
	return
}

func (a App) buildCloneRemoteConfig(remType, remRoot, account string) (core.RemoteConfig, error) {
	cfg := core.RemoteConfig{
		Type:         remType,
		ObjectRoot:   "objects",
		ManifestRoot: "manifests",
		VerifyMode:   "local-hash",
		Priority:     50,
		Pull:         true,
		Push:         true,
	}

	switch remType {
	case "local":
		cfg.Path = remRoot
		cfg.VerifyMode = "local-hash"

	case "rclone":
		cfg.Remote = remRoot

	case "drive":
		var authPath string

		if account != "" {
			p := gdrive.DefaultAuthPath(account)
			if _, statErr := os.Stat(p); os.IsNotExist(statErr) {
				accounts, _ := listLocalAccounts()
				var names []string
				for _, a := range accounts {
					names = append(names, a.Name)
				}
				hint := ""
				if len(names) > 0 {
					hint = fmt.Sprintf("\n  available profiles: %s", strings.Join(names, ", "))
				}
				return cfg, fmt.Errorf("account profile %q not found%s\n"+
					"  run 'kk setup gdrive' to add a new account", account, hint)
			}
			authPath = p
			fmt.Printf("kk: using Google Drive account profile %q\n", account)
		} else {
			defaultPath := gdrive.DefaultAuthPath("default")
			if _, statErr := os.Stat(defaultPath); !os.IsNotExist(statErr) {
				authPath = defaultPath
			} else {
				accounts, err := listLocalAccounts()
				if err != nil {
					return cfg, err
				}
				if len(accounts) == 0 {
					return cfg, fmt.Errorf(
						"google drive auth not found at %s\n"+
							"  run 'kk setup gdrive' first to authorise access", defaultPath)
				}

				fmt.Println("Default Google Drive account not found. Choose an account to use:")
				for i, acc := range accounts {
					info := acc.Name
					if acc.Email != "" && acc.DisplayName != "" {
						info = fmt.Sprintf("%s (%s - %s)", acc.Name, acc.Email, acc.DisplayName)
					} else if acc.Email != "" {
						info = fmt.Sprintf("%s (%s)", acc.Name, acc.Email)
					}
					fmt.Printf("  [%d] %s\n", i+1, info)
				}
				fmt.Printf("  [%d] Log in to a new Google Drive account online\n", len(accounts)+1)

				reader := bufio.NewReader(os.Stdin)
				choice, err := promptChoice(reader, len(accounts)+1)
				if err != nil {
					selected := accounts[0]
					fmt.Printf("Non-interactive mode or error reading choice (%v). Selecting first account: %q (%s)\n", err, selected.Name, selected.Email)
					authPath = selected.Path
				} else if choice <= len(accounts) {
					selected := accounts[choice-1]
					fmt.Printf("Selected Google Drive account: %q (%s)\n", selected.Name, selected.Email)
					authPath = selected.Path
				} else {
					fmt.Println("Select Google Drive authorization scope:")
					fmt.Println("  [1] Restricted app-specific scope (drive.file) [Default]")
					fmt.Println("      Access only to files/folders created by KK. More secure.")
					fmt.Println("  [2] Full drive access scope (drive)")
					fmt.Println("      Access to all files/folders. Required for Shared Drives and shared folders.")

					scopeChoice, scopeErr := promptChoice(reader, 2)
					driveScope := "https://www.googleapis.com/auth/drive.file"
					if scopeErr == nil && scopeChoice == 2 {
						driveScope = "https://www.googleapis.com/auth/drive"
					}

					newAuthPath, loginErr := a.loginNewGoogleDriveAccount(reader, driveScope)
					if loginErr != nil {
						return cfg, fmt.Errorf("failed to log in to new account: %w", loginErr)
					}
					authPath = newAuthPath
				}
			}
		}
		cfg.DriveFolderID = remRoot
		cfg.DriveAuthPath = authPath
		cfg.Provider = "google-drive"

	case "git":
		cfg.URL = remRoot
		cfg.Pull = true
		cfg.Push = true
		cfg.Provider = "git"
		fmt.Printf("kk: git remote URL → %s\n", remRoot)

	default:
		return cfg, fmt.Errorf("unsupported remote type %q", remType)
	}

	return cfg, nil
}

func explainDriveFolderResolveError(err error, authPath string) error {
	var apiErr *gdrive.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		return err
	}

	hint := "Google returned 404 for this folder. Confirm the folder ID is correct and that the selected Google account has access."
	auth, authErr := gdrive.LoadAuth(authPath)
	if authErr == nil && !hasGoogleDriveFullScope(auth.Scope) {
		hint += "\n  This account is authorized with the restricted drive.file scope, which cannot access folders shared by other accounts.\n" +
			"  Re-authorize with full Drive scope: kk setup gdrive --auth-only --scope full\n" +
			"  Then retry clone with the matching profile: kk clone drive:<project-folder-id> --account <profile>"
	} else {
		hint += "\n  For folders shared by another account, make sure the share is granted to this Google account.\n" +
			"  If you only have a link-sharing URL with a resourcekey parameter, ask the owner to share the folder directly with your account."
	}

	return fmt.Errorf("%w\n  %s", err, hint)
}

func hasGoogleDriveFullScope(scope string) bool {
	for _, part := range strings.Fields(scope) {
		if part == "https://www.googleapis.com/auth/drive" {
			return true
		}
	}
	return false
}

func cloneRestoreHistory(driver remote.Driver, info core.RepoInfo, localRoot string, gitClient git.Client) error {
	remoteSnap, ok, err := driver.GetRefsSnapshot(info)
	if err != nil {
		return fmt.Errorf("reading remote refs snapshot: %w", err)
	}
	if !ok || len(remoteSnap.Branches) == 0 {
		return fmt.Errorf("no history bundles found on remote (was the project pushed with history enabled?)")
	}
	if err := remoteSnap.Validate(); err != nil {
		return fmt.Errorf("invalid remote refs snapshot: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "kk-clone-history-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	fmt.Printf("kk: [--history] restoring history from %d branch(es)...\n", len(remoteSnap.Branches))

	localStateBranches := make(map[string]core.BranchHistoryState)

	for branchName, branchSnap := range remoteSnap.Branches {
		for i, bundleName := range branchSnap.Bundles {
			localPath := filepath.Join(tmpDir, bundleName)
			fmt.Printf("kk: [--history] downloading bundle %d/%d (%s) for branch %s...\n",
				i+1, len(branchSnap.Bundles), bundleName, branchName)
			if err := driver.GetHistoryBundle(info, branchName, bundleName, localPath); err != nil {
				return fmt.Errorf("downloading %s for %s: %w", bundleName, branchName, err)
			}
			if err := gitClient.UnbundleHistory(localPath); err != nil {
				return fmt.Errorf("applying %s for %s: %w", bundleName, branchName, err)
			}
		}

		if len(branchSnap.Bundles) > 0 {
			localStateBranches[branchName] = core.BranchHistoryState{
				LastAppliedBundle: branchSnap.Bundles[len(branchSnap.Bundles)-1],
				LastAppliedRef:    branchSnap.BaseRef,
			}
		}
	}

	defaultBranch := remoteSnap.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = "main"
	}
	if err := gitClient.SetupFromBundle(defaultBranch); err != nil {
		return fmt.Errorf("setting up from bundle: %w", err)
	}

	histState := core.HistoryState{
		Version: core.HistoryStateVersion,
		Remotes: map[string]core.RemoteHistoryState{
			"origin": {
				Branches:  localStateBranches,
				UpdatedAt: remoteSnap.UpdatedAt,
			},
		},
	}
	if err := writeHistoryState(localRoot, histState); err != nil {
		fmt.Printf("kk: warning: could not save history state: %v\n", err)
	}

	branches, _ := gitClient.ListAllBranches()
	fmt.Printf("kk: [--history] %d branch(es) restored: %s\n",
		len(branches), strings.Join(branches, ", "))
	return nil
}

func (a App) cloneFromGit(url, dest, remoteName, account, branch string, doPull, withHistory, here, force bool, workers int, gitConfig core.RemoteConfig) error {
	if dest == "" {
		idx := strings.LastIndex(url, "/")
		projectName := url[idx+1:]
		projectName = strings.TrimSuffix(projectName, ".git")
		dest = projectName
	}
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return fmt.Errorf("kk clone: resolving destination: %w", err)
	}

	if !here {
		if entries, _ := os.ReadDir(absDest); len(entries) > 0 {
			return fmt.Errorf("kk clone: destination %q already exists and is not empty", absDest)
		}
	}

	_, statErr := os.Stat(absDest)
	destExisted := !os.IsNotExist(statErr)

	if err := os.MkdirAll(absDest, 0o750); err != nil {
		return fmt.Errorf("kk clone: creating destination: %w", err)
	}

	fmt.Printf("kk: cloning from git remote...\n")
	tmpCloneDir, err := os.MkdirTemp("", "kk-git-clone-*")
	if err != nil {
		return fmt.Errorf("kk clone: creating temp clone dir: %w", err)
	}
	cloneArgs := []string{"clone"}
	if branch != "" {
		cloneArgs = append(cloneArgs, "--branch", branch)
	}
	cloneArgs = append(cloneArgs, url, tmpCloneDir)
	cmd := exec.Command("git", cloneArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(tmpCloneDir)
		if !here && !destExisted {
			_ = os.RemoveAll(absDest)
		}
		return fmt.Errorf("kk clone: git clone failed: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpCloneDir) }()

	fmt.Printf("kk: copying project files...\n")
	if err := copyDirContents(tmpCloneDir, absDest); err != nil {
		if !here && !destExisted {
			_ = os.RemoveAll(absDest)
		}
		return fmt.Errorf("kk clone: copying files: %w", err)
	}

	kkDir := filepath.Join(absDest, core.KKDir)
	if _, err := os.Stat(kkDir); os.IsNotExist(err) {
		fmt.Printf("kk: warning: cloned repository does not appear to be a KK repository\n")
		fmt.Printf("kk: initializing KK structure...\n")
		for _, dir := range []string{core.KKDir, core.ObjectDir, core.TmpDir, core.LogDir} {
			if err := os.MkdirAll(filepath.Join(absDest, dir), 0o750); err != nil {
				return fmt.Errorf("kk clone: creating %s: %w", dir, err)
			}
		}
		tmpGitDir := filepath.Join(tmpCloneDir, ".git")
		kkGitDir := filepath.Join(absDest, core.KKGitDir)
		if err := os.Rename(tmpGitDir, kkGitDir); err != nil {
			return fmt.Errorf("kk clone: preserving git history: %w", err)
		}

		gitClient := git.New(absDest)
		if remoteName == "" {
			remoteName = "origin"
		}
		if remoteName != "origin" && gitClient.HasGitRemote("origin") && !gitClient.HasGitRemote(remoteName) {
			if err := gitClient.RenameRemote("origin", remoteName); err != nil {
				return fmt.Errorf("kk clone: renaming git remote: %w", err)
			}
		}
		if !gitClient.HasGitRemote(remoteName) {
			if err := gitClient.AddRemote(remoteName, url); err != nil {
				return fmt.Errorf("kk clone: adding git remote: %w", err)
			}
		}
		if err := ensureGitExclude(absDest); err != nil {
			return fmt.Errorf("kk clone: history exclude: %w", err)
		}

		freshCfg := core.DefaultConfig()
		gitConfig.DisplayName = remoteName
		freshCfg.Remotes[remoteName] = gitConfig
		freshCfg.DefaultRemote = remoteName
		if err := core.WriteConfig(absDest, freshCfg); err != nil {
			return fmt.Errorf("kk clone: writing config: %w", err)
		}

		info, err := core.NewRepoInfo(absDest)
		if err != nil {
			return fmt.Errorf("kk clone: creating repo info: %w", err)
		}
		if err := core.WriteRepoInfo(absDest, info); err != nil {
			return fmt.Errorf("kk clone: writing repo info: %w", err)
		}
		if err := core.WriteTracks(absDest, core.Tracks{Patterns: []string{}}); err != nil {
			return fmt.Errorf("kk clone: writing tracks: %w", err)
		}

		fmt.Printf("\nkk: clone complete -> %s\n", absDest)
		fmt.Println("kk: repository initialized as KK project")
		return nil
	}

	cfg, err := core.ReadConfig(absDest)
	if err != nil {
		return fmt.Errorf("kk clone: reading config: %w", err)
	}

	if remoteName == "" {
		remoteName = "origin"
	}
	gitConfig.DisplayName = remoteName
	cfg.Remotes[remoteName] = gitConfig
	if cfg.DefaultRemote == "" {
		cfg.DefaultRemote = remoteName
	}
	if err := core.WriteConfig(absDest, cfg); err != nil {
		return fmt.Errorf("kk clone: writing config: %w", err)
	}

	kkGitDir := filepath.Join(absDest, core.KKGitDir)
	if _, err := os.Stat(kkGitDir); os.IsNotExist(err) {
		if err := os.MkdirAll(kkGitDir, 0o750); err != nil {
			return fmt.Errorf("kk clone: creating .kk/git: %w", err)
		}
		gitClient := git.New(absDest)
		if err := gitClient.InitMain(); err != nil {
			return fmt.Errorf("kk clone: history init: %w", err)
		}
		if err := gitClient.AddRemote(remoteName, url); err != nil {
			return fmt.Errorf("kk clone: adding git remote: %w", err)
		}
		if err := gitClient.Run("fetch", remoteName); err != nil {
			return fmt.Errorf("kk clone: fetching from git remote: %w", err)
		}
	}

	a.checkAndPromptRemoteSetup(absDest)

	fmt.Printf("\nkk: clone complete -> %s\n", absDest)

	if doPull {
		return a.materializeAfterClone(absDest, workers)
	}

	fmt.Printf("kk: run 'cd %s && kk fsck'          to check large-file status\n", absDest)
	fmt.Println("    run 'kk pull-file <file>'     to materialise individual files")
	fmt.Println("    run 'kk pull-file .'          to materialise all pointer files at once")
	fmt.Println("    run 'kk pull-file --all'      to materialise everything (alias for .)")

	return nil
}

func copyDirContents(src, dest string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if rel == ".git" {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if strings.HasPrefix(filepath.Base(path), ".") && filepath.Base(path) != ".kk" {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(dest, rel), 0o750)
		}
		if d.Type()&os.ModeType != 0 {
			return nil
		}
		in, err := os.Open(path) // #nosec G304,G122 -- path comes from WalkDir under a fresh git clone temp dir; non-regular entries are skipped.
		if err != nil {
			return err
		}
		defer func() {
			_ = in.Close()
		}()
		out, err := os.OpenFile(filepath.Join(dest, rel), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600) // #nosec G304 -- rel is produced by filepath.Rel from the walked source tree.
		if err != nil {
			return err
		}
		_, err = io.Copy(out, in)
		closeErr := out.Close()
		if err != nil {
			return err
		}
		return closeErr
	})
}

func (a App) checkAndPromptRemoteSetup(dest string) {
	cfg, err := core.ReadConfig(dest)
	if err != nil {
		return
	}

	var inaccessibleRemotes []string
	for name, remoteCfg := range cfg.Remotes {
		if remoteCfg.Type == "git" {
			continue
		}
		driver, err := remote.New(name, remoteCfg)
		if err != nil {
			inaccessibleRemotes = append(inaccessibleRemotes, name)
			continue
		}
		if err := driver.Check(); err != nil {
			inaccessibleRemotes = append(inaccessibleRemotes, name)
		}
	}

	if len(inaccessibleRemotes) > 0 {
		fmt.Printf("kk: warning: some object remotes are not accessible:\n")
		for _, name := range inaccessibleRemotes {
			fmt.Printf("  - %s\n", name)
		}
		fmt.Println("\n  To materialize pointer files, configure these remotes:")
		for _, name := range inaccessibleRemotes {
			remoteCfg := cfg.Remotes[name]
			switch remoteCfg.Type {
			case "rclone":
				target := remoteCfg.Remote
				if target == "" {
					target = "<target>"
				}
				binaryPart := ""
				if remoteCfg.Binary != "" {
					binaryPart = fmt.Sprintf(" [--binary %s]", remoteCfg.Binary)
				}
				fmt.Printf("    kk remote add rclone %s --remote %s%s\n", name, target, binaryPart)
			case "drive":
				fmt.Printf("    kk setup gdrive\n")
			case "local":
				fmt.Printf("    kk remote add local %s --path %s\n", name, remoteCfg.Path)
			}
		}
	}
}

func (a App) materializeAfterClone(dest string, workers int) error {
	clonedApp := App{Root: dest}
	fmt.Println("kk: materialising large files ...")

	objects, err := clonedApp.requiredObjectsInHEAD()
	if err != nil {
		return fmt.Errorf("kk clone: listing objects: %w", err)
	}

	if len(objects) == 0 {
		fmt.Println("kk: no pointer files to materialise")
		return nil
	}

	w := resolveWorkers(workers)
	bar := newMultiProgressBar(len(objects), "pulling", min(w, len(objects)))

	type matResult struct {
		slotID int
		path   string
		err    error
	}

	jobs := make(chan core.RequiredObject)
	results := make(chan matResult, len(objects))

	var wg sync.WaitGroup
	pullWorkers := min(w, len(objects))
	for i := 0; i < pullWorkers; i++ {
		slotID := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			for obj := range jobs {
				bar.SetSlot(slotID, obj.Path)
				err := clonedApp.materialize(obj.Path, false, false)
				results <- matResult{slotID: slotID, path: obj.Path, err: err}
			}
		}()
	}

	go func() {
		for _, obj := range objects {
			jobs <- obj
		}
		close(jobs)
	}()
	go func() {
		wg.Wait()
		close(results)
	}()

	var matWarnings []string
	for r := range results {
		bar.Complete(r.slotID, r.path)
		if r.err != nil {
			matWarnings = append(matWarnings, fmt.Sprintf("%s: %v", r.path, r.err))
		}
	}
	succeeded := len(objects) - len(matWarnings)
	bar.Finish(fmt.Sprintf("kk: materialised %d/%d object(s)", succeeded, len(objects)))
	for _, w := range matWarnings {
		fmt.Printf("kk: warning: could not materialise %s\n", w)
	}
	fmt.Println("kk: materialisation complete")
	return nil
}
