package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/godynheil/kk/internal/core"
	"github.com/godynheil/kk/internal/git"
	"github.com/godynheil/kk/internal/storage"
)

func (a App) Stage(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: kk stage <path...>")
	}
	return git.New(a.Root).Run(append([]string{"add", "--"}, args...)...)
}

func (a App) Unstage(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: kk unstage <path...>")
	}
	return git.New(a.Root).Run(append([]string{"restore", "--staged", "--"}, args...)...)
}

func (a App) Discard(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: kk discard <path...>")
	}
	return git.New(a.Root).Run(append([]string{"restore", "--"}, args...)...)
}

func (a App) Git(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: kk <command> [args...]")
	}

	client := git.New(a.Root)
	if isShortStatus(args) {
		return a.gitStatus(args)
	}

	isSwitch := isBranchSwitch(args, client)

	var cleanMaterialized map[string]string
	var oldHead string
	realChangesBeforeSwitch := true
	if isSwitch {
		var err error
		cleanMaterialized, err = a.findCleanMaterializedFiles(client)
		if err != nil {
			fmt.Printf("kk: warning: failed to scan for clean materialized files: %v\n", err)
		} else {
			realChangesBeforeSwitch = a.hasRealStatusChanges(client, cleanMaterialized)
		}
		if len(cleanMaterialized) > 0 {
			for file, content := range cleanMaterialized {
				filePath := filepath.Join(a.Root, file)
				if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
					for restoredFile := range cleanMaterialized {
						_ = a.materialize(restoredFile, false, false)
					}
					return fmt.Errorf("failed to temporarily dematerialize %s: %w", file, err)
				}
			}

			_, _, _ = client.Combined("update-index", "--refresh")
		}
		if client.HasHEAD() {
			oldHead, _ = client.HeadCommit()
		}
	}

	gitErr := client.Run(args...)
	if gitErr != nil && isSwitch && len(cleanMaterialized) > 0 && !realChangesBeforeSwitch {
		gitErr = client.Run(forceBranchSwitchArgs(args)...)
	}

	if isSwitch {
		if gitErr != nil {
			for restoredFile := range cleanMaterialized {
				_ = a.materialize(restoredFile, false, false)
			}
			return gitErr
		}

		materializedCount := 0
		for file := range cleanMaterialized {
			filePath := filepath.Join(a.Root, file)
			if _, err := os.Stat(filePath); err == nil {
				if _, ok, err := pointerFromWorkingFile(a.Root, file); err == nil && ok {
					if err := a.materialize(file, false, true); err == nil {
						materializedCount++
					} else {
						fmt.Printf("kk: warning: failed to materialize %s: %v\n", file, err)
					}
				}
			}
		}

		var newHead string
		if client.HasHEAD() {
			newHead, _ = client.HeadCommit()
		}
		if oldHead != "" && newHead != "" {
			changed, _ := client.ChangedFiles(oldHead, newHead)
			store := storage.New(a.Root)
			for _, file := range changed {
				if _, ok := cleanMaterialized[file]; ok {
					continue
				}
				filePath := filepath.Join(a.Root, file)
				if _, err := os.Stat(filePath); err == nil {
					if p, ok, err := pointerFromWorkingFile(a.Root, file); err == nil && ok {
						if store.HasObject(p) {
							if err := a.materialize(file, false, true); err == nil {
								materializedCount++
							} else {
								fmt.Printf("kk: warning: failed to materialize changed file %s: %v\n", file, err)
							}
						}
					}
				}
			}
		}

		if materializedCount > 0 {
			fmt.Printf("kk: branch switch complete; materialized %d file(s)\n", materializedCount)
		} else {
			fmt.Println("kk: branch switch complete")
		}
		return nil
	}

	if gitErr != nil {
		return gitErr
	}

	if warnsAboutPointers(args) {
		fmt.Println("kk: checkout/switch completed; use `kk pull-file <path>` to materialize large files when needed")
	}
	return nil
}

func (a App) hasRealStatusChanges(client git.Client, cleanMaterialized map[string]string) bool {
	out, _, err := client.Combined("status", "--porcelain")
	if err != nil {
		return true
	}
	filtered := filterCleanMaterializedStatus(out, cleanMaterialized)
	return strings.TrimSpace(filtered) != ""
}

func forceBranchSwitchArgs(args []string) []string {
	for _, arg := range args[1:] {
		if arg == "-f" || arg == "--force" || arg == "--discard-changes" {
			return args
		}
	}
	out := append([]string{}, args...)
	switch args[0] {
	case "checkout":
		return append([]string{"checkout", "-f"}, out[1:]...)
	case "switch":
		return append([]string{"switch", "--discard-changes"}, out[1:]...)
	default:
		return args
	}
}

func isShortStatus(args []string) bool {
	if len(args) == 0 || args[0] != "status" {
		return false
	}
	for _, arg := range args[1:] {
		if arg == "-z" {
			return false
		}
		if arg == "--short" || arg == "-s" || arg == "--porcelain" || strings.HasPrefix(arg, "--porcelain=") {
			return true
		}
	}
	return false
}

func (a App) gitStatus(args []string) error {
	out, stderr, err := a.gitStatusOutput(args)
	if out != "" {
		fmt.Print(out)
	}
	if stderr != "" {
		fmt.Fprint(os.Stderr, stderr)
	}
	return err
}

func (a App) gitStatusOutput(args []string) (string, string, error) {
	client := git.New(a.Root)
	out, stderr, err := client.Combined(args...)
	if err != nil {
		return out, stderr, err
	}
	cleanMaterialized, scanErr := a.findCleanMaterializedFiles(client)
	if scanErr != nil || len(cleanMaterialized) == 0 {
		return out, stderr, nil
	}
	return filterCleanMaterializedStatus(out, cleanMaterialized), stderr, nil
}

func filterCleanMaterializedStatus(raw string, cleanMaterialized map[string]string) string {
	if raw == "" {
		return ""
	}
	hasFinalNewline := strings.HasSuffix(raw, "\n")
	lines := strings.Split(strings.TrimSuffix(raw, "\n"), "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		if shouldHideCleanMaterializedStatusLine(line, cleanMaterialized) {
			continue
		}
		filtered = append(filtered, line)
	}
	if len(filtered) == 0 {
		return ""
	}
	out := strings.Join(filtered, "\n")
	if hasFinalNewline {
		out += "\n"
	}
	return out
}

func shouldHideCleanMaterializedStatusLine(line string, cleanMaterialized map[string]string) bool {
	if len(line) < 4 || strings.HasPrefix(line, "##") {
		return false
	}
	if line[0] != ' ' || line[1] != 'M' {
		return false
	}
	path := strings.TrimSpace(line[3:])
	if path == "" || strings.Contains(path, " -> ") {
		return false
	}
	path = strings.Trim(path, `"`)
	_, ok := cleanMaterialized[filepath.ToSlash(path)]
	return ok
}

func warnsAboutPointers(args []string) bool {
	switch args[0] {
	case "checkout", "switch":
		return true
	default:
		return false
	}
}

func isBranchSwitch(args []string, client git.Client) bool {
	if len(args) == 0 {
		return false
	}
	if args[0] == "switch" {
		return true
	}
	if args[0] != "checkout" {
		return false
	}
	for _, arg := range args[1:] {
		if arg == "-b" || arg == "-B" || arg == "-d" || arg == "--detach" || arg == "--orphan" || arg == "-t" || arg == "--track" {
			return true
		}
		if arg == "--" {
			break
		}
	}

	var nonFlags []string
	for _, arg := range args[1:] {
		if arg == "--" {
			break
		}
		if !strings.HasPrefix(arg, "-") {
			nonFlags = append(nonFlags, arg)
		}
	}

	if len(nonFlags) == 0 {
		return false
	}

	if nonFlags[0] == "-" {
		return true
	}

	if len(nonFlags) > 1 {
		return false
	}

	target := nonFlags[0]
	_, _, err := client.Combined("rev-parse", "--verify", target)
	return err == nil
}

func (a App) findCleanMaterializedFiles(client git.Client) (map[string]string, error) {
	if !client.HasHEAD() {
		return nil, nil
	}
	statusOut, _, err := client.Combined("status", "--porcelain")
	if err != nil {
		return nil, err
	}
	candidates := make(map[string]bool)
	for _, line := range strings.Split(statusOut, "\n") {
		line = strings.TrimSuffix(line, "\r")
		if len(line) < 4 {
			continue
		}
		if line[0] == 'M' || line[1] == 'M' {
			path := strings.TrimSpace(line[3:])
			path = strings.Trim(path, `"`)
			candidates[filepath.ToSlash(path)] = true
		}
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	cleanMaterialized := make(map[string]string)
	for file := range candidates {
		content, err := client.ShowHeadFile(file)
		if err != nil {
			continue
		}
		pointer, ok := core.ParsePointerText(content)
		if !ok {
			continue
		}
		filePath := filepath.Join(a.Root, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			continue
		}
		_, isPointer, err := pointerFromWorkingFile(a.Root, file)
		if err == nil && isPointer {
			continue
		}
		oid, size, err := storage.HashFile(filePath)
		if err != nil {
			continue
		}
		if oid == pointer.OID && size == pointer.Size {
			cleanMaterialized[file] = content
		}
	}
	return cleanMaterialized, nil
}
