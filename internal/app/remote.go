package app

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/godynheil/kk/internal/core"
	"github.com/godynheil/kk/internal/git"
	"github.com/godynheil/kk/internal/remote"
)

func (a App) Remote(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: kk remote <add|list|set-default|check|remove|rename|migrate>")
	}
	switch args[0] {
	case "add":
		return a.remoteAdd(args[1:])
	case "list":
		return a.remoteList(hasFlag(args[1:], "--json"))
	case "set-default":
		return a.remoteSetDefault(args[1:])
	case "check":
		return a.remoteCheck(args[1:])
	case "remove":
		return a.remoteRemove(args[1:])
	case "rename":
		return a.remoteRename(args[1:])
	case "migrate":
		return a.remoteMigrate(args[1:])
	default:
		return fmt.Errorf("unknown remote command: %s", args[0])
	}
}

func (a App) remoteAdd(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: kk remote add <local|rclone|git> <name> [options]")
	}
	kind, name := args[0], args[1]
	if err := core.ValidateRemoteName(name); err != nil {
		return err
	}

	if kind == "git" {
		return a.remoteAddGit(name, args[2:])
	}

	cfg, err := core.ReadConfig(a.Root)
	if err != nil {
		return err
	}
	remoteCfg := core.RemoteConfig{Type: kind, ObjectRoot: "objects", ManifestRoot: "manifests", VerifyMode: "local-hash", Priority: 50, Pull: true, Push: true}
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--display-name":
			if err := fatalIfMissingValue(args, i, args[i]); err != nil {
				return err
			}
			remoteCfg.DisplayName = args[i+1]
			i++
		case "--role":
			if err := fatalIfMissingValue(args, i, args[i]); err != nil {
				return err
			}
			remoteCfg.Role = args[i+1]
			i++
		case "--provider":
			if err := fatalIfMissingValue(args, i, args[i]); err != nil {
				return err
			}
			remoteCfg.Provider = args[i+1]
			i++
		case "--path":
			if err := fatalIfMissingValue(args, i, args[i]); err != nil {
				return err
			}
			remoteCfg.Path = args[i+1]
			i++
		case "--binary":
			if err := fatalIfMissingValue(args, i, args[i]); err != nil {
				return err
			}
			remoteCfg.Binary = args[i+1]
			i++
		case "--remote":
			if err := fatalIfMissingValue(args, i, args[i]); err != nil {
				return err
			}
			remoteCfg.Remote = args[i+1]
			i++
		case "--object-root":
			if err := fatalIfMissingValue(args, i, args[i]); err != nil {
				return err
			}
			remoteCfg.ObjectRoot = args[i+1]
			i++
		case "--manifest-root":
			if err := fatalIfMissingValue(args, i, args[i]); err != nil {
				return err
			}
			remoteCfg.ManifestRoot = args[i+1]
			i++
		case "--verify-mode":
			if err := fatalIfMissingValue(args, i, args[i]); err != nil {
				return err
			}
			remoteCfg.VerifyMode = args[i+1]
			i++
		case "--priority":
			if err := fatalIfMissingValue(args, i, args[i]); err != nil {
				return err
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil {
				return err
			}
			remoteCfg.Priority = n
			i++
		case "--pull":
			if err := fatalIfMissingValue(args, i, args[i]); err != nil {
				return err
			}
			remoteCfg.Pull = boolArg(args[i+1], true)
			i++
		case "--push":
			if err := fatalIfMissingValue(args, i, args[i]); err != nil {
				return err
			}
			remoteCfg.Push = boolArg(args[i+1], true)
			i++
		case "--tag":
			if err := fatalIfMissingValue(args, i, args[i]); err != nil {
				return err
			}
			remoteCfg.Tags = append(remoteCfg.Tags, args[i+1])
			i++
		case "--upload-timeout":
			if err := fatalIfMissingValue(args, i, args[i]); err != nil {
				return err
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil {
				return err
			}
			remoteCfg.UploadTimeoutSeconds = n
			i++
		case "--chunk-size":
			if err := fatalIfMissingValue(args, i, args[i]); err != nil {
				return err
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil {
				return err
			}
			remoteCfg.ChunkSizeMB = n
			i++
		case "--buffer-size":
			if err := fatalIfMissingValue(args, i, args[i]); err != nil {
				return err
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil {
				return err
			}
			remoteCfg.BufferSizeMB = n
			i++
		case "--rclone-transfers":
			if err := fatalIfMissingValue(args, i, args[i]); err != nil {
				return err
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil {
				return err
			}
			remoteCfg.RcloneTransfers = n
			i++
		case "--disable-connection-pool":
			remoteCfg.DisableConnectionPool = true
		default:
			return fmt.Errorf("unknown remote add option: %s", args[i])
		}
	}
	if remoteCfg.DisplayName == "" {
		remoteCfg.DisplayName = name
	}
	if remoteCfg.Provider == "" {
		remoteCfg.Provider = kind
	}
	driver, err := remote.New(name, remoteCfg)
	if err != nil {
		return err
	}
	info, err := core.ReadRepoInfo(a.Root)
	if err != nil {
		return err
	}
	if existing, ok, err := driver.ReadRemoteRepoInfo(info); err == nil && ok {
		if existing.RepoID != info.RepoID {
			fmt.Printf("kk: adopting remote project ID: %s (local was %s)\n", existing.RepoID, info.RepoID)
			if err := core.WriteRepoInfo(a.Root, existing); err != nil {
				return fmt.Errorf("failed to adopt remote project ID: %w", err)
			}
		}
	}

	cfg.Remotes[name] = remoteCfg
	if cfg.DefaultRemote == "" {
		cfg.DefaultRemote = name
	}
	if err := core.WriteConfig(a.Root, cfg); err != nil {
		return err
	}
	fmt.Println("remote added", name)
	printCloneHint(kind, remoteCfg, a.Root)
	return nil
}

func (a App) remoteAddGit(name string, args []string) error {
	gitClient := git.New(a.Root)
	if err := gitClient.EnsureRepository(); err != nil {
		return err
	}

	cfg, err := core.ReadConfig(a.Root)
	if err != nil {
		return err
	}
	if _, exists := cfg.Remotes[name]; exists {
		return fmt.Errorf("remote %q already exists; use 'kk remote remove %s' first", name, name)
	}

	remoteCfg := core.RemoteConfig{
		Type: "git",
		Pull: true,
		Push: true,
		Tags: []string{"git"},
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--url":
			if err := fatalIfMissingValue(args, i, args[i]); err != nil {
				return err
			}
			remoteCfg.URL = args[i+1]
			i++
		case "--display-name":
			if err := fatalIfMissingValue(args, i, args[i]); err != nil {
				return err
			}
			remoteCfg.DisplayName = args[i+1]
			i++
		case "--provider":
			if err := fatalIfMissingValue(args, i, args[i]); err != nil {
				return err
			}
			remoteCfg.Provider = args[i+1]
			i++
		case "--push":
			if err := fatalIfMissingValue(args, i, args[i]); err != nil {
				return err
			}
			remoteCfg.Push = boolArg(args[i+1], true)
			i++
		case "--pull":
			if err := fatalIfMissingValue(args, i, args[i]); err != nil {
				return err
			}
			remoteCfg.Pull = boolArg(args[i+1], true)
			i++
		default:
			if !strings.HasPrefix(args[i], "--") && remoteCfg.URL == "" {
				remoteCfg.URL = args[i]
				continue
			}
			return fmt.Errorf("unknown option for 'remote add git': %s", args[i])
		}
	}

	if remoteCfg.URL == "" {
		return fmt.Errorf("usage: kk remote add git <name> <url>\n" +
			"  or:  kk remote add git <name> --url <url>\n" +
			"  example: kk remote add git origin https://git.example.com/org/repo.git")
	}
	if remoteCfg.DisplayName == "" {
		remoteCfg.DisplayName = name
	}
	if remoteCfg.Provider == "" {
		remoteCfg.Provider = "git"
	}

	fmt.Printf("kk: checking git remote accessibility...\n")
	if err := gitClient.CheckRemoteConnectivity(remoteCfg.URL); err != nil {
		return fmt.Errorf("cannot access git remote %q: %w", remoteCfg.URL, err)
	}

	if gitClient.HasGitRemote(name) {
		return fmt.Errorf("git remote %q already exists in .kk/git; remove it first with:\n"+
			"  git --git-dir=.kk/git remote remove %s", name, name)
	}
	if err := gitClient.AddRemote(name, remoteCfg.URL); err != nil {
		return fmt.Errorf("adding git remote: %w", err)
	}

	cfg.Remotes[name] = remoteCfg
	if err := core.WriteConfig(a.Root, cfg); err != nil {
		_ = gitClient.RemoveRemote(name)
		return err
	}

	fmt.Printf("kk: git remote %q added → %s\n", name, remoteCfg.URL)
	fmt.Printf("kk: pointer history will be pushed to %s on 'kk push'\n", name)
	fmt.Printf("kk: binary objects are stored separately on your KK object remote(s)\n")
	return nil
}

func (a App) remoteList(jsonOut bool) error {
	cfg, err := core.ReadConfig(a.Root)
	if err != nil {
		return err
	}
	if jsonOut {
		b, _ := json.MarshalIndent(cfg, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	fmt.Println("default:", cfg.DefaultRemote)
	for name, r := range cfg.Remotes {
		if r.Type == "git" {
			fmt.Printf("%s type=git provider=%s url=%s push=%v pull=%v\n", name, r.Provider, r.URL, r.Push, r.Pull)
		} else {
			fmt.Printf("%s type=%s provider=%s role=%s pull=%v push=%v priority=%d\n", name, r.Type, r.Provider, r.Role, r.Pull, r.Push, r.Priority)
		}
	}
	return nil
}

func (a App) remoteSetDefault(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: kk remote set-default <name>")
	}
	cfg, err := core.ReadConfig(a.Root)
	if err != nil {
		return err
	}
	if _, ok := cfg.Remotes[args[0]]; !ok {
		return fmt.Errorf("remote not found: %s", args[0])
	}
	cfg.DefaultRemote = args[0]
	if err := core.WriteConfig(a.Root, cfg); err != nil {
		return err
	}
	fmt.Println("default remote:", args[0])
	return nil
}

func (a App) remoteRemove(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: kk remote remove <name>")
	}
	cfg, err := core.ReadConfig(a.Root)
	if err != nil {
		return err
	}
	r, exists := cfg.Remotes[args[0]]
	if !exists {
		return fmt.Errorf("remote not found: %s", args[0])
	}
	if r.Type == "git" {
		gitClient := git.New(a.Root)
		if gitClient.HasGitRemote(args[0]) {
			if err := gitClient.RemoveRemote(args[0]); err != nil {
				return fmt.Errorf("removing git remote from .kk/git: %w", err)
			}
		}
	}
	delete(cfg.Remotes, args[0])
	if cfg.DefaultRemote == args[0] {
		cfg.DefaultRemote = ""
	}
	return core.WriteConfig(a.Root, cfg)
}

func (a App) remoteRename(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: kk remote rename <old> <new>")
	}
	if err := core.ValidateRemoteName(args[1]); err != nil {
		return err
	}
	cfg, err := core.ReadConfig(a.Root)
	if err != nil {
		return err
	}
	r, ok := cfg.Remotes[args[0]]
	if !ok {
		return fmt.Errorf("remote not found: %s", args[0])
	}
	if r.Type == "git" {
		gitClient := git.New(a.Root)
		if gitClient.HasGitRemote(args[0]) {
			if err := gitClient.RenameRemote(args[0], args[1]); err != nil {
				return fmt.Errorf("renaming git remote in .kk/git: %w", err)
			}
		}
	}
	delete(cfg.Remotes, args[0])
	cfg.Remotes[args[1]] = r
	if cfg.DefaultRemote == args[0] {
		cfg.DefaultRemote = args[1]
	}
	return core.WriteConfig(a.Root, cfg)
}

func (a App) remoteMigrate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: kk remote migrate <to-git|to-storage> [options]\n" +
			"  to-git   <name> <url>  — switch from storage bundles to a git hosting remote\n" +
			"  to-storage             — switch from a git remote to storage-bundle history")
	}
	switch args[0] {
	case "to-git":
		return a.remoteMigrateToGit(args[1:])
	case "to-storage":
		return a.remoteMigrateToStorage(args[1:])
	default:
		return fmt.Errorf("unknown migrate direction %q; use 'to-git' or 'to-storage'", args[0])
	}
}

func (a App) remoteMigrateToGit(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: kk remote migrate to-git <name> <url>\n" +
			"  example: kk remote migrate to-git github https://github.com/org/repo.git")
	}

	name := args[0]
	url := args[1]
	push := true
	pull := true
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--push":
			if err := fatalIfMissingValue(args, i, args[i]); err != nil {
				return err
			}
			push = boolArg(args[i+1], true)
			i++
		case "--pull":
			if err := fatalIfMissingValue(args, i, args[i]); err != nil {
				return err
			}
			pull = boolArg(args[i+1], true)
			i++
		default:
			return fmt.Errorf("unknown option for 'remote migrate to-git': %s", args[i])
		}
	}

	if err := core.ValidateRemoteName(name); err != nil {
		return err
	}

	cfg, err := core.ReadConfig(a.Root)
	if err != nil {
		return err
	}

	if len(cfg.GitRemotes()) > 0 {
		var existing []string
		for _, r := range cfg.GitRemotes() {
			existing = append(existing, r.Name)
		}
		fmt.Printf("kk: already in git-remote mode (remote(s): %s) — nothing to migrate\n",
			strings.Join(existing, ", "))
		return nil
	}

	if _, exists := cfg.Remotes[name]; exists {
		return fmt.Errorf("remote %q already exists; choose a different name or remove it first", name)
	}

	gitClient := git.New(a.Root)
	if err := gitClient.EnsureRepository(); err != nil {
		return err
	}

	fmt.Printf("kk: checking git remote accessibility...\n")
	if err := gitClient.CheckRemoteConnectivity(url); err != nil {
		return fmt.Errorf("cannot reach git remote %q: %w", url, err)
	}

	if gitClient.HasGitRemote(name) {
		return fmt.Errorf("git remote %q already exists in .kk/git; remove it first with:\n"+
			"  git --git-dir=.kk/git remote remove %s", name, name)
	}

	if err := gitClient.AddRemote(name, url); err != nil {
		return fmt.Errorf("adding git remote: %w", err)
	}

	remoteCfg := core.RemoteConfig{
		Type:        "git",
		URL:         url,
		DisplayName: name,
		Provider:    "git",
		Push:        push,
		Pull:        pull,
		Tags:        []string{"git"},
	}
	cfg.Remotes[name] = remoteCfg
	if err := core.WriteConfig(a.Root, cfg); err != nil {
		_ = gitClient.RemoveRemote(name)
		return err
	}

	fmt.Printf("kk: pushing all local branches to %q...\n", name)
	stdout, stderr, pushErr := gitClient.Combined("push", "--all", name)
	if stdout != "" {
		fmt.Print(stdout)
	}
	if stderr != "" {
		fmt.Print(stderr)
	}
	if pushErr != nil {

		fmt.Printf("kk: warning: initial git push failed: %v\n", pushErr)
		fmt.Printf("kk:          Once auth is set up, run:\n")
		fmt.Printf("kk:          git --git-dir=.kk/git push --all %s\n", name)
	} else {
		fmt.Printf("kk: [%s] all branches pushed\n", name)
	}

	fmt.Printf("\nkk: migration complete → git remote %q registered (%s)\n", name, url)
	fmt.Printf("kk: future 'kk push' will sync pointer history to %q via git push\n", name)
	fmt.Printf("kk: existing history bundles on object remote(s) are kept but will not be extended\n")
	return nil
}

func (a App) remoteMigrateToStorage(args []string) error {
	var targetRemoteName string
	skipConfirm := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--remote":
			if err := fatalIfMissingValue(args, i, args[i]); err != nil {
				return err
			}
			targetRemoteName = args[i+1]
			i++
		case "--yes", "-y":
			skipConfirm = true
		default:
			return fmt.Errorf("unknown option for 'remote migrate to-storage': %s", args[i])
		}
	}

	cfg, err := core.ReadConfig(a.Root)
	if err != nil {
		return err
	}

	gitRemotes := cfg.GitRemotes()
	if len(gitRemotes) == 0 {
		fmt.Println("kk: already in storage-bundle mode — nothing to migrate")
		return nil
	}

	var toRemove []core.NamedRemoteConfig
	if targetRemoteName != "" {
		r, _, err := cfg.GetRemote(targetRemoteName)
		if err != nil {
			return err
		}
		if r.Type != "git" {
			return fmt.Errorf("remote %q has type %q, not 'git'; use 'kk remote migrate to-storage' without --remote to target all git remotes",
				targetRemoteName, r.Type)
		}
		toRemove = []core.NamedRemoteConfig{{Name: targetRemoteName, Config: r}}
	} else {
		toRemove = gitRemotes
	}

	storageTargets, err := cfg.PushRemotes(nil, true)
	if err != nil || len(storageTargets) == 0 {
		return fmt.Errorf("no push-enabled object remote found; add a local/rclone/drive remote before migrating")
	}

	var removeNames []string
	for _, r := range toRemove {
		removeNames = append(removeNames, fmt.Sprintf("%s (%s)", r.Name, r.Config.URL))
	}
	fmt.Printf("kk: This will:\n")
	fmt.Printf("kk:   1. Upload a full history bundle to: ")
	for i, t := range storageTargets {
		if i > 0 {
			fmt.Printf(", ")
		}
		fmt.Printf("%s", t.Name)
	}
	fmt.Println()
	fmt.Printf("kk:   2. Remove git remote(s): %s\n", strings.Join(removeNames, ", "))
	fmt.Printf("kk:   Future 'kk push' will upload incremental bundles instead of using git push.\n")

	if !skipConfirm {
		fmt.Printf("\nProceed? [y/N] ")
		var answer string
		_, _ = fmt.Fscan(os.Stdin, &answer)
		if !strings.EqualFold(strings.TrimSpace(answer), "y") {
			fmt.Println("kk: migration cancelled")
			return nil
		}
	}

	fmt.Println("kk: creating initial history bundle(s)...")
	if err := a.pushHistory(storageTargets); err != nil {
		return fmt.Errorf("failed to create initial history bundles: %w; migration aborted — no changes were made to your remote configuration", err)
	}

	gitClient := git.New(a.Root)
	for _, r := range toRemove {
		if gitClient.HasGitRemote(r.Name) {
			if err := gitClient.RemoveRemote(r.Name); err != nil {
				fmt.Printf("kk: warning: could not remove git remote %q from .kk/git: %v\n", r.Name, err)
			}
		}
		delete(cfg.Remotes, r.Name)
		if cfg.DefaultRemote == r.Name {
			cfg.DefaultRemote = ""
		}
	}

	if cfg.DefaultRemote == "" && len(storageTargets) > 0 {
		cfg.DefaultRemote = storageTargets[0].Name
		fmt.Printf("kk: default remote set to %q\n", cfg.DefaultRemote)
	}

	if err := core.WriteConfig(a.Root, cfg); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Printf("\nkk: migration complete → commit history will now travel via storage bundles\n")
	fmt.Printf("kk: teammates can restore history with: kk clone <spec> --history\n")
	return nil
}

func printCloneHint(kind string, cfg core.RemoteConfig, root string) {
	switch kind {
	case "rclone":
		if cfg.Remote != "" {
			fmt.Printf("    Teammates can clone with:\n")
			fmt.Printf("    kk clone rclone:%s\n", cfg.Remote)
		}
	case "local":
		if cfg.Path != "" {
			fmt.Printf("    Teammates can clone with:\n")
			fmt.Printf("    kk clone local:%s\n", cfg.Path)
		}
	}
}

type RemoteCheckResult struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

func (a App) remoteCheck(args []string) error {
	jsonOut := hasFlag(args, "--json")
	all := hasFlag(args, "--all")
	args = removeFlags(args, "--json", "--all")
	cfg, err := core.ReadConfig(a.Root)
	if err != nil {
		return err
	}
	var names []string
	if all {
		for name := range cfg.Remotes {
			names = append(names, name)
		}
	} else if len(args) > 0 {
		names = append(names, args[0])
	} else if cfg.DefaultRemote != "" {
		names = append(names, cfg.DefaultRemote)
	} else {
		return fmt.Errorf("no remote specified")
	}
	var results []RemoteCheckResult
	okAll := true
	gitClient := git.New(a.Root)
	for _, name := range names {
		rcfg, _, err := cfg.GetRemote(name)
		if err != nil {
			results = append(results, RemoteCheckResult{Name: name, OK: false, Message: err.Error()})
			okAll = false
			continue
		}
		if rcfg.Type == "git" {
			if !gitClient.HasGitRemote(name) {
				msg := fmt.Sprintf("git remote %q not found in .kk/git (expected URL: %s)", name, rcfg.URL)
				results = append(results, RemoteCheckResult{Name: name, OK: false, Message: msg})
				okAll = false
				continue
			}
			actualURL := gitClient.GetRemoteURL(name)
			if rcfg.URL != "" && actualURL != rcfg.URL {
				msg := fmt.Sprintf("URL mismatch: config=%q .kk/git=%q", rcfg.URL, actualURL)
				results = append(results, RemoteCheckResult{Name: name, OK: false, Message: msg})
				okAll = false
				continue
			}
			results = append(results, RemoteCheckResult{Name: name, OK: true,
				Message: fmt.Sprintf("git remote configured (%s)", actualURL)})
			continue
		}
		driver, err := remote.New(name, rcfg)
		if err != nil {
			results = append(results, RemoteCheckResult{Name: name, OK: false, Message: err.Error()})
			okAll = false
			continue
		}
		if err := driver.Check(); err != nil {
			results = append(results, RemoteCheckResult{Name: name, OK: false, Message: err.Error()})
			okAll = false
			continue
		}
		results = append(results, RemoteCheckResult{Name: name, OK: true})
	}
	if jsonOut {
		b, _ := json.MarshalIndent(results, "", "  ")
		fmt.Println(string(b))
	} else {
		for _, result := range results {
			if result.OK {
				fmt.Println("ok", result.Name)
			} else {
				fmt.Println("bad", result.Name, result.Message)
			}
		}
	}
	if !okAll {
		return fmt.Errorf("one or more remotes failed")
	}
	return nil
}
