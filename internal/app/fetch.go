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
