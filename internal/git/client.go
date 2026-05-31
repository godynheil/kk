package git

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/godynheil/kk/internal/core"
)

const maxCommandLineLen = 28000

type Client struct {
	Root string
}

func New(root string) Client {
	if root == "" {
		root = "."
	}
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}
	return Client{Root: root}
}

func (c Client) gitDir() string {
	return filepath.Join(c.Root, core.KKGitDir)
}

func (c Client) args(args ...string) []string {
	base := []string{"--git-dir=" + c.gitDir(), "--work-tree=" + c.Root}
	return append(base, args...)
}

func (c Client) Run(args ...string) error {
	gitPath, err := GitExecutable()
	if err != nil {
		return err
	}
	cmd := exec.Command(gitPath, c.args(args...)...) // #nosec G204 -- gitPath is resolved by GitExecutable; arguments are passed without shell expansion.
	cmd.Dir = c.Root
	var out, er bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &er
	cmd.Stdin = os.Stdin
	err = cmd.Run()
	if err == nil {
		if out.Len() > 0 {
			_, _ = os.Stdout.Write(out.Bytes())
		}
		if er.Len() > 0 {
			_, _ = os.Stderr.Write(er.Bytes())
		}
		return nil
	}
	return errors.New(sanitizeFailure(out.String(), er.String(), err))
}

func (c Client) RunBatched(prefix []string, items []string) error {
	return c.runBatched(prefix, items, maxCommandLineLen)
}

func (c Client) runBatched(prefix []string, items []string, maxLen int) error {
	if len(items) == 0 {
		return c.Run(prefix...)
	}
	if maxLen <= 0 {
		maxLen = maxCommandLineLen
	}

	var batch []string
	baseLen := c.commandLineLen(prefix)
	currentLen := baseLen
	for _, item := range items {
		itemLen := commandArgLen(item)
		if len(batch) > 0 && currentLen+itemLen > maxLen {
			if err := c.Run(append(append([]string{}, prefix...), batch...)...); err != nil {
				return err
			}
			batch = batch[:0]
			currentLen = baseLen
		}
		batch = append(batch, item)
		currentLen += itemLen
	}
	if len(batch) > 0 {
		return c.Run(append(append([]string{}, prefix...), batch...)...)
	}
	return nil
}

func (c Client) commandLineLen(args []string) int {
	gitPath, err := GitExecutable()
	if err != nil {
		gitPath = "git"
	}
	total := commandArgLen(gitPath)
	for _, arg := range c.args(args...) {
		total += commandArgLen(arg)
	}
	return total
}

func commandArgLen(arg string) int {
	return len(arg) + 3
}

func (c Client) Output(args ...string) (string, error) {
	gitPath, err := GitExecutable()
	if err != nil {
		return "", err
	}
	cmd := exec.Command(gitPath, c.args(args...)...) // #nosec G204 -- gitPath is resolved by GitExecutable; arguments are passed without shell expansion.
	cmd.Dir = c.Root
	out, err := cmd.Output()
	return string(out), err
}

func (c Client) Combined(args ...string) (stdout string, stderr string, err error) {
	gitPath, err := GitExecutable()
	if err != nil {
		return "", "", err
	}
	cmd := exec.Command(gitPath, c.args(args...)...) // #nosec G204 -- gitPath is resolved by GitExecutable; arguments are passed without shell expansion.
	cmd.Dir = c.Root
	var out, er bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &er
	err = cmd.Run()
	return out.String(), er.String(), err
}

func (c Client) InitMain() error {
	if c.IsInitialized() {
		return nil
	}
	if err := c.Run("init", "--initial-branch=main"); err == nil {
		return nil
	}
	if err := c.Run("init"); err != nil {
		return err
	}
	return c.Run("symbolic-ref", "HEAD", "refs/heads/main")
}

func (c Client) IsInitialized() bool {
	_, err := os.Stat(filepath.Join(c.gitDir(), "HEAD"))
	return err == nil
}

func (c Client) ShowHeadFile(path string) (string, error) {
	return c.Output("show", "HEAD:"+path)
}

func (c Client) ShowIndexFile(path string) ([]byte, error) {
	gitPath, err := GitExecutable()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(gitPath, c.args("show", ":"+path)...) // #nosec G204 -- gitPath is resolved by GitExecutable; path is a git object spec argument, not shell input.
	cmd.Dir = c.Root
	return cmd.Output()
}

func (c Client) CurrentBranch() string {
	out, err := c.Output("branch", "--show-current")
	if err != nil {
		return ""
	}
	return stringTrimSpace(out)
}

func (c Client) StagedFiles() ([]string, error) {
	out, err := c.Output("diff", "--cached", "--name-only")
	if err != nil {
		return nil, err
	}
	return nonEmptyLines(out), nil
}

func (c Client) DeletedFiles(path string) ([]string, error) {
	var args []string
	if path != "" {
		args = []string{"ls-files", "--deleted", "--", path}
	} else {
		args = []string{"ls-files", "--deleted"}
	}
	out, err := c.Output(args...)
	if err != nil {
		return nil, err
	}
	return nonEmptyLines(out), nil
}

func (c Client) HeadFiles() ([]string, error) {
	out, _, err := c.Combined("ls-tree", "-r", "--name-only", "HEAD")
	if err != nil {
		return []string{}, nil
	}
	return nonEmptyLines(out), nil
}

func (c Client) HasHEAD() bool {
	_, _, err := c.Combined("rev-parse", "--verify", "HEAD")
	return err == nil
}

func (c Client) HeadCommit() (string, error) {
	out, err := c.Output("rev-parse", "HEAD")
	return stringTrimSpace(out), err
}

func (c Client) IsAncestor(ancestor, descendant string) bool {
	if ancestor == "" || descendant == "" {
		return false
	}
	_, _, err := c.Combined("merge-base", "--is-ancestor", ancestor, descendant)
	return err == nil
}

func (c Client) ChangedFiles(fromCommit, toCommit string) ([]string, error) {
	if fromCommit == "" || toCommit == "" {
		return []string{}, nil
	}
	out, _, err := c.Combined("diff", "--name-only", "--diff-filter=ACMRT", fromCommit+".."+toCommit)
	if err != nil {
		return nil, err
	}
	return nonEmptyLines(out), nil
}

func (c Client) HasRemotes() bool {
	out, _, err := c.Combined("remote")
	return err == nil && len(nonEmptyLines(out)) > 0
}

func (c Client) EnsureRepository() error {
	if !c.IsInitialized() {
		return errors.New("not a kk repository; run kk init first")
	}
	return nil
}

func (c Client) HasMergeConflicts() bool {
	for _, marker := range []string{"MERGE_HEAD", "rebase-merge", "rebase-apply"} {
		if _, err := os.Stat(filepath.Join(c.gitDir(), marker)); err == nil {
			return true
		}
	}
	return false
}

func (c Client) AddRemote(name, url string) error {
	return c.Run("remote", "add", name, url)
}

func (c Client) RemoveRemote(name string) error {
	return c.Run("remote", "remove", name)
}

func (c Client) RenameRemote(oldName, newName string) error {
	return c.Run("remote", "rename", oldName, newName)
}

func (c Client) GetRemoteURL(name string) string {
	out, err := c.Output("remote", "get-url", name)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

func (c Client) HasGitRemote(name string) bool {
	out, _, err := c.Combined("remote")
	if err != nil {
		return false
	}
	for _, r := range nonEmptyLines(out) {
		if r == name {
			return true
		}
	}
	return false
}

func (c Client) CheckRemoteConnectivity(url string) error {
	tempName := "kk-check-connectivity-tmp"
	if err := c.Run("remote", "add", tempName, url); err != nil {
		return fmt.Errorf("failed to add temporary remote for check: %w", err)
	}
	defer func() {
		_ = c.Run("remote", "remove", tempName)
	}()

	_, stderr, err := c.Combined("ls-remote", tempName)
	if err != nil {
		if strings.Contains(stderr, "authentication failed") || strings.Contains(stderr, "could not read Username") {
			return fmt.Errorf("authentication failed for %s: %s", url, sanitizeFailure("", stderr, err))
		}
		if strings.Contains(stderr, "Could not resolve host") || strings.Contains(stderr, "Could not connect") {
			return fmt.Errorf("cannot connect to %s: %s", url, sanitizeFailure("", stderr, err))
		}
		return fmt.Errorf("remote %s is not accessible: %s", url, sanitizeFailure("", stderr, err))
	}
	return nil
}

func (c Client) CreateBundle(destPath, sinceRef, branch string) error {
	args := []string{"bundle", "create", destPath, branch}
	if sinceRef != "" {
		args = append(args, "^"+sinceRef)
	}
	return c.Run(args...)
}

func (c Client) ApplyBundle(bundlePath string) error {
	abs, err := filepath.Abs(bundlePath)
	if err != nil {
		return err
	}

	fwd := filepath.ToSlash(abs)
	return c.Run("fetch", fwd, "+refs/heads/*:refs/remotes/kk-history/*")
}

func (c Client) UnbundleHistory(bundlePath string) error {
	abs, err := filepath.Abs(bundlePath)
	if err != nil {
		return err
	}
	fwd := filepath.ToSlash(abs)
	return c.Run("fetch", "--update-head-ok", fwd, "+refs/heads/*:refs/heads/*")
}

func (c Client) MergeHistoryBranch(branch string) error {
	ref := "refs/remotes/kk-history/" + branch
	if _, err := c.Output("rev-parse", "--verify", ref); err != nil {
		return errors.New("remote tracking ref " + ref + " not found after fetch")
	}
	return c.Run("merge", "--no-edit", ref)
}

func (c Client) ListAllBranches() ([]string, error) {
	out, err := c.Output("branch", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}
	return nonEmptyLines(out), nil
}

func (c Client) DefaultBranch() string {
	out, err := c.Output("symbolic-ref", "--short", "HEAD")
	if err != nil {
		return "main"
	}
	if b := stringTrimSpace(out); b != "" {
		return b
	}
	return "main"
}

func (c Client) BranchCommit(branch string) (string, error) {
	return c.Output("rev-parse", "refs/heads/"+branch)
}

func (c Client) SetupFromBundle(defaultBranch string) error {
	headRef := "refs/heads/" + defaultBranch
	if _, err := c.Output("rev-parse", "--verify", headRef); err != nil {
		return errors.New("branch " + headRef + " not found after unbundle")
	}

	if err := c.Run("symbolic-ref", "HEAD", "refs/heads/"+defaultBranch); err != nil {
		return err
	}

	_ = c.Run("reset", "--mixed", "HEAD")
	return nil
}

func (c Client) AllCommits() ([]string, error) {
	var args []string
	if c.HasHEAD() {
		args = []string{"rev-list", "--branches", "--tags", "HEAD"}
	} else {
		args = []string{"rev-list", "--branches", "--tags"}
	}
	out, _, err := c.Combined(args...)
	if err != nil {
		return []string{}, nil
	}
	return nonEmptyLines(out), nil
}

func (c Client) FilesAtCommit(commit string) ([]string, error) {
	out, _, err := c.Combined("ls-tree", "-r", "--name-only", commit)
	if err != nil {
		return []string{}, nil
	}
	return nonEmptyLines(out), nil
}

func (c Client) ShowCommitFile(commit string, path string) (string, error) {
	return c.Output("show", commit+":"+path)
}

func stringTrimSpace(s string) string {
	return string(bytes.TrimSpace([]byte(s)))
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, raw := range bytes.Split([]byte(s), []byte("\n")) {
		line := string(bytes.TrimSpace(raw))
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func sanitizeFailure(stdout, stderr string, err error) string {
	msg := stringTrimSpace(stderr)
	if msg == "" {
		msg = stringTrimSpace(stdout)
	}
	if msg == "" && err != nil {
		msg = err.Error()
	} else if err != nil && err.Error() != "" {
		msg += "\n" + err.Error()
	}
	msg = strings.ReplaceAll(msg, "Git", "KK")
	msg = strings.ReplaceAll(msg, "git", "kk")
	return msg
}

func (c Client) ShowHeadFilesBatch(files []string) (map[string]string, error) {
	if len(files) == 0 {
		return map[string]string{}, nil
	}
	gitPath, err := GitExecutable()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(gitPath, c.args("cat-file", "--batch")...) // #nosec G204 -- gitPath is resolved by GitExecutable; arguments are fixed and passed without a shell.
	cmd.Dir = c.Root

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	res := map[string]string{}
	errChan := make(chan error, 1)

	go func() {
		defer func() {
			_ = stdin.Close()
		}()
		for _, file := range files {
			_, err := io.WriteString(stdin, "HEAD:"+file+"\n")
			if err != nil {
				return
			}
		}
	}()

	go func() {
		reader := bufio.NewReader(stdout)
		for _, file := range files {
			header, err := reader.ReadString('\n')
			if err != nil {
				errChan <- err
				return
			}
			header = strings.TrimSpace(header)
			if strings.HasSuffix(header, "missing") {
				continue
			}
			parts := strings.Split(header, " ")
			if len(parts) < 3 {
				errChan <- fmt.Errorf("invalid cat-file header: %s", header)
				return
			}
			sizeStr := parts[2]
			size, err := strconv.ParseInt(sizeStr, 10, 64)
			if err != nil {
				errChan <- err
				return
			}

			buf := make([]byte, size)
			_, err = io.ReadFull(reader, buf)
			if err != nil {
				errChan <- err
				return
			}

			_, _ = reader.ReadByte()

			res[file] = string(buf)
		}
		errChan <- nil
	}()

	readErr := <-errChan
	waitErr := cmd.Wait()
	if readErr != nil {
		return nil, readErr
	}
	if waitErr != nil {
		return nil, waitErr
	}

	return res, nil
}
