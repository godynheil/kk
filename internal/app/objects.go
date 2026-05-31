package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/godynheil/kk/internal/core"
	"github.com/godynheil/kk/internal/git"
	"github.com/godynheil/kk/internal/remote"
	"github.com/godynheil/kk/internal/storage"
)

func (a App) requiredObjectsInHEAD() ([]core.RequiredObject, error) {
	client := git.New(a.Root)
	if !client.HasHEAD() {
		return []core.RequiredObject{}, nil
	}
	files, err := client.HeadFiles()
	if err != nil {
		return nil, err
	}
	contents, err := client.ShowHeadFilesBatch(files)
	if err != nil {
		return nil, err
	}
	seen := map[string]core.RequiredObject{}
	for _, file := range files {
		content, ok := contents[file]
		if !ok {
			continue
		}
		p, ok := core.ParsePointerText(content)
		if !ok {
			continue
		}
		key := fmt.Sprintf("%s:%d", p.OID, p.Size)
		if existing, ok := seen[key]; ok {
			var pathBuilder strings.Builder
			pathBuilder.WriteString(existing.Path)
			pathBuilder.WriteString(", ")
			pathBuilder.WriteString(file)
			existing.Path = pathBuilder.String()
			seen[key] = existing
		} else {
			seen[key] = core.RequiredObject{Path: file, OID: p.OID, Size: p.Size}
		}
	}
	var out []core.RequiredObject
	for _, obj := range seen {
		out = append(out, obj)
	}
	return out, nil
}

func (a App) requiredObjectsForHeadFiles(files []string) ([]core.RequiredObject, error) {
	client := git.New(a.Root)
	contents, err := client.ShowHeadFilesBatch(files)
	if err != nil {
		return nil, err
	}
	seen := map[string]core.RequiredObject{}
	for _, file := range files {
		content, ok := contents[file]
		if !ok {
			continue
		}
		p, ok := core.ParsePointerText(content)
		if !ok {
			continue
		}
		key := fmt.Sprintf("%s:%d", p.OID, p.Size)
		if existing, ok := seen[key]; ok {
			var pathBuilder strings.Builder
			pathBuilder.WriteString(existing.Path)
			pathBuilder.WriteString(", ")
			pathBuilder.WriteString(file)
			existing.Path = pathBuilder.String()
			seen[key] = existing
		} else {
			seen[key] = core.RequiredObject{Path: file, OID: p.OID, Size: p.Size}
		}
	}
	var out []core.RequiredObject
	for _, obj := range seen {
		out = append(out, obj)
	}
	return out, nil
}

func pointerFromWorkingFile(root, rel string) (core.Pointer, bool, error) {
	path := filepath.Join(root, rel)
	info, err := os.Stat(path)
	if err != nil {
		return core.Pointer{}, false, err
	}
	if info.Size() > 4096 {
		return core.Pointer{}, false, nil
	}
	f, err := os.Open(path) // #nosec G304 -- path is joined from repository root and a git-reported relative path.
	if err != nil {
		return core.Pointer{}, false, err
	}
	defer func() {
		_ = f.Close()
	}()
	buf := make([]byte, 1024)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return core.Pointer{}, false, err
	}
	p, ok := core.ParsePointerText(string(buf[:n]))
	return p, ok, nil
}

func (a App) Objects(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: kk objects <live|refs|prune|sync>")
	}
	switch args[0] {
	case "live":
		return a.ObjectsLive(hasFlag(args[1:], "--json"))
	case "refs":
		if len(args) < 2 {
			return fmt.Errorf("usage: kk objects refs <sha256> [--json]")
		}
		return a.ObjectRefs(args[1], hasFlag(args[2:], "--json"))
	case "prune":
		return a.ObjectsPrune(hasFlag(args[1:], "--dry-run"), hasFlag(args[1:], "--json"))
	case "sync":
		return a.ObjectsSync(args[1:])
	default:
		return fmt.Errorf("unknown objects command: %s", args[0])
	}
}

func (a App) ObjectsLive(jsonOut bool) error {
	live, err := a.liveObjectsAcrossAllRefs()
	if err != nil {
		return err
	}
	if jsonOut {
		b, _ := json.MarshalIndent(live, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	keys := sortedLiveKeys(live)
	if len(keys) == 0 {
		fmt.Println("no live kk objects found")
		return nil
	}
	for _, oid := range keys {
		obj := live[oid]
		fmt.Printf("%s size=%d refs=%d\n", obj.OID, obj.Size, len(obj.Refs))
	}
	return nil
}

func (a App) ObjectRefs(oid string, jsonOut bool) error {
	live, err := a.liveObjectsAcrossAllRefs()
	if err != nil {
		return err
	}
	obj, ok := live[oid]
	if !ok {
		obj = core.LiveObject{OID: oid, Refs: []core.ObjectRef{}}
	}
	if jsonOut {
		b, _ := json.MarshalIndent(obj, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	if len(obj.Refs) == 0 {
		fmt.Printf("%s has no reachable refs\n", oid)
		return nil
	}
	fmt.Printf("%s is referenced by:\n", oid)
	for _, ref := range obj.Refs {
		fmt.Printf("  %s  %s\n", ref.Commit, ref.Path)
	}
	return nil
}

func (a App) ObjectsPrune(dryRun bool, jsonOut bool) error {
	live, err := a.liveObjectsAcrossAllRefs()
	if err != nil {
		return err
	}
	store := storage.New(a.Root)
	cached, err := store.ListObjects()
	if err != nil {
		return err
	}
	var candidates []core.PruneCandidate
	for oid, path := range cached {
		if _, ok := live[oid]; !ok {
			candidates = append(candidates, core.PruneCandidate{OID: oid, Path: path})
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].OID < candidates[j].OID })
	if jsonOut {
		b, _ := json.MarshalIndent(map[string]any{
			"dry_run":    dryRun,
			"candidates": candidates,
		}, "", "  ")
		fmt.Println(string(b))
	}
	if len(candidates) == 0 {
		if !jsonOut {
			fmt.Println("no local objects to prune")
		}
		return nil
	}
	if dryRun {
		if !jsonOut {
			for _, c := range candidates {
				fmt.Println("would prune", c.OID)
			}
		}
		return nil
	}
	var oids []string
	for _, c := range candidates {
		oids = append(oids, c.OID)
	}
	if err := store.PruneObjects(oids); err != nil {
		return err
	}
	if !jsonOut {
		for _, c := range candidates {
			fmt.Println("pruned", c.OID)
		}
	}
	return nil
}

func (a App) liveObjectsAcrossAllRefs() (map[string]core.LiveObject, error) {
	client := git.New(a.Root)
	out, err := client.Output("rev-list", "--objects", "--all")
	if err != nil {
		return nil, err
	}
	blobToPaths := make(map[string][]string)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		sha := parts[0]
		path := parts[1]
		blobToPaths[sha] = append(blobToPaths[sha], path)
	}
	if len(blobToPaths) == 0 {
		return map[string]core.LiveObject{}, nil
	}
	gitPath, err := git.GitExecutable()
	if err != nil {
		return nil, err
	}
	checkCmd := exec.Command(gitPath, "--git-dir="+filepath.Join(a.Root, core.KKGitDir), "cat-file", "--batch-check=%(objectname) %(objecttype) %(objectsize)") // #nosec G204 -- gitPath is resolved by GitExecutable; arguments are fixed.
	checkStdin, _ := checkCmd.StdinPipe()
	checkStdout, _ := checkCmd.StdoutPipe()
	if err := checkCmd.Start(); err != nil {
		return nil, err
	}
	go func() {
		defer func() {
			_ = checkStdin.Close()
		}()
		for sha := range blobToPaths {
			_, _ = io.WriteString(checkStdin, sha+"\n")
		}
	}()
	var candidates []string
	scanner := bufio.NewScanner(checkStdout)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, " ")
		if len(parts) == 3 && parts[1] == "blob" {
			size, _ := strconv.ParseInt(parts[2], 10, 64)
			if size <= 1024 {
				candidates = append(candidates, parts[0])
			}
		}
	}
	_ = checkCmd.Wait()
	if len(candidates) == 0 {
		return map[string]core.LiveObject{}, nil
	}
	readCmd := exec.Command(gitPath, "--git-dir="+filepath.Join(a.Root, core.KKGitDir), "cat-file", "--batch") // #nosec G204 -- gitPath is resolved by GitExecutable; arguments are fixed.
	readStdin, _ := readCmd.StdinPipe()
	readStdout, _ := readCmd.StdoutPipe()
	if err := readCmd.Start(); err != nil {
		return nil, err
	}
	go func() {
		defer func() {
			_ = readStdin.Close()
		}()
		for _, sha := range candidates {
			_, _ = io.WriteString(readStdin, sha+"\n")
		}
	}()
	live := map[string]core.LiveObject{}
	reader := bufio.NewReader(readStdout)
	for _, sha := range candidates {
		header, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		headerParts := strings.Split(strings.TrimSpace(header), " ")
		if len(headerParts) < 3 {
			continue
		}
		size, _ := strconv.ParseInt(headerParts[2], 10, 64)
		buf := make([]byte, size)
		_, _ = io.ReadFull(reader, buf)
		_, _ = reader.ReadByte()
		p, ok := core.ParsePointerBytes(buf)
		if !ok {
			continue
		}
		obj, exists := live[p.OID]
		if !exists {
			obj = core.LiveObject{OID: p.OID, Size: p.Size, Refs: []core.ObjectRef{}}
		}
		paths := blobToPaths[sha]
		for _, path := range paths {
			obj.Refs = append(obj.Refs, core.ObjectRef{Commit: "reachable", Path: path})
		}
		live[p.OID] = obj
	}
	_ = readCmd.Wait()
	return live, nil
}

func sortedLiveKeys(live map[string]core.LiveObject) []string {
	keys := make([]string, 0, len(live))
	for oid := range live {
		keys = append(keys, oid)
	}
	sort.Strings(keys)
	return keys
}

func RequiredObjectsFromLive(live map[string]core.LiveObject) []core.RequiredObject {
	keys := sortedLiveKeys(live)
	out := make([]core.RequiredObject, 0, len(keys))
	for _, oid := range keys {
		obj := live[oid]
		path := ""
		if len(obj.Refs) > 0 {
			path = obj.Refs[0].Path
		}
		out = append(out, core.RequiredObject{Path: path, OID: obj.OID, Size: obj.Size})
	}
	return out
}

func (a App) ObjectsSync(args []string) error {
	verbose := hasFlag(args, "--verbose")
	args = removeFlags(args, "--verbose")
	workers := 0
	for i := 0; i < len(args); i++ {
		if args[i] == "--workers" && i+1 < len(args) {
			if n, err := strconv.Atoi(args[i+1]); err == nil && n > 0 {
				workers = n
			}
			args = append(args[:i], args[i+2:]...)
			i--
		} else if strings.HasPrefix(args[i], "--workers=") {
			if n, err := strconv.Atoi(strings.TrimPrefix(args[i], "--workers=")); err == nil && n > 0 {
				workers = n
			}
		}
	}
	w := resolveWorkers(workers)

	live, err := a.liveObjectsAcrossAllRefs()
	if err != nil {
		return fmt.Errorf("listing live objects: %w", err)
	}
	if len(live) == 0 {
		fmt.Println("kk: no live objects to sync")
		return nil
	}

	cfg, err := core.ReadConfig(a.Root)
	if err != nil {
		return err
	}
	targets, err := cfg.PushRemotes(nil, true)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return fmt.Errorf("no push-enabled object remotes configured")
	}

	info, err := core.ReadRepoInfo(a.Root)
	if err != nil {
		return err
	}
	store := storage.New(a.Root)

	type remoteState struct {
		name     string
		driver   remote.Driver
		manifest core.Manifest
		hasMap   map[string]bool
		modified bool
	}

	var remoteStates []*remoteState
	for _, target := range targets {
		driver, err := remote.New(target.Name, target.Config)
		if err != nil {
			return fmt.Errorf("failed to initialise remote %s: %w", target.Name, err)
		}
		if err := driver.Check(); err != nil {
			return fmt.Errorf("remote %s check failed: %w", target.Name, err)
		}
		manifest, err := driver.ReadManifest(info)
		if err != nil {
			manifest = remote.EmptyManifest(info)
		}
		hasMap := make(map[string]bool)
		for _, mObj := range manifest.Objects {
			hasMap[mObj.OID] = true
		}
		remoteStates = append(remoteStates, &remoteState{
			name:     target.Name,
			driver:   driver,
			manifest: manifest,
			hasMap:   hasMap,
		})
	}

	type syncTask struct {
		pointer     core.Pointer
		missingFrom []*remoteState
	}

	var tasks []syncTask
	for _, obj := range live {
		p := core.Pointer{OID: obj.OID, Size: obj.Size}
		var missingFrom []*remoteState
		for _, rState := range remoteStates {
			exists := rState.hasMap[p.OID]
			if !exists {
				exists, err = rState.driver.HasObject(info, p)
				if err != nil {
					exists = false
				}
			}
			if !exists {
				missingFrom = append(missingFrom, rState)
			} else {
				rState.hasMap[p.OID] = true
			}
		}
		if len(missingFrom) > 0 {
			tasks = append(tasks, syncTask{
				pointer:     p,
				missingFrom: missingFrom,
			})
		}
	}

	if len(tasks) == 0 {
		fmt.Println("kk: all live objects are already fully synced across remotes")
		return nil
	}

	fmt.Printf("kk: syncing %d object(s) across %d remote(s) (%d workers)...\n", len(tasks), len(remoteStates), w)

	type taskResult struct {
		pointer core.Pointer
		err     error
	}

	jobs := make(chan syncTask)
	results := make(chan taskResult, len(tasks))

	var wg sync.WaitGroup
	for i := 0; i < w; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range jobs {
				p := task.pointer
				localPath := store.ObjectPath(p.OID)

				if !store.HasObject(p) {
					downloaded := false
					for _, rState := range remoteStates {
						if rState.hasMap[p.OID] {
							if verbose {
								fmt.Printf("kk: downloading %s from %s for replication...\n", p.OID[:12], rState.name)
							}
							if err := rState.driver.GetObject(info, p, localPath); err == nil {
								downloaded = true
								break
							}
						}
					}
					if !downloaded {
						results <- taskResult{pointer: p, err: fmt.Errorf("object missing locally and could not be retrieved from any remote")}
						continue
					}
				}

				var uploadErr error
				for _, rState := range task.missingFrom {
					if verbose {
						fmt.Printf("kk: uploading %s to %s...\n", p.OID[:12], rState.name)
					}
					if err := rState.driver.PutObject(info, p, localPath); err != nil {
						uploadErr = fmt.Errorf("replicating to %s failed: %w", rState.name, err)
						break
					}
				}

				if uploadErr != nil {
					results <- taskResult{pointer: p, err: uploadErr}
				} else {
					results <- taskResult{pointer: p}
				}
			}
		}()
	}

	go func() {
		for _, task := range tasks {
			jobs <- task
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	var errors []error
	succeededCount := 0
	for res := range results {
		if res.err != nil {
			errors = append(errors, fmt.Errorf("syncing %s: %w", res.pointer.OID[:12], res.err))
		} else {
			succeededCount++
			for _, task := range tasks {
				if task.pointer.OID == res.pointer.OID {
					for _, rState := range task.missingFrom {
						rState.manifest = remote.UpsertManifestObject(rState.manifest, task.pointer)
						rState.hasMap[task.pointer.OID] = true
						rState.modified = true
					}
				}
			}
		}
	}

	for _, rState := range remoteStates {
		if rState.modified {
			if err := rState.driver.WriteManifest(info, rState.manifest); err != nil {
				errors = append(errors, fmt.Errorf("writing manifest for %s: %w", rState.name, err))
			}
		}
	}

	fmt.Printf("kk: synced %d/%d object(s)\n", succeededCount, len(tasks))
	for _, err := range errors {
		fmt.Printf("kk: warning: %v\n", err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("object sync encountered %d error(s)", len(errors))
	}
	return nil
}
