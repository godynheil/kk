package app

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/godynheil/kk/internal/core"
	"github.com/godynheil/kk/internal/git"
)

type StatusResult struct {
	Initialized bool   `json:"initialized"`
	RepoID      string `json:"repo_id,omitempty"`
	Name        string `json:"name,omitempty"`
	Branch      string `json:"branch,omitempty"`
	Dirty       bool   `json:"dirty"`
	Raw         string `json:"raw,omitempty"`
}

func (a App) Status(jsonOut bool) error {
	client := git.New(a.Root)
	res := StatusResult{Initialized: client.IsInitialized()}
	if res.Initialized {
		if info, err := core.ReadRepoInfo(a.Root); err == nil {
			res.RepoID = info.RepoID
			res.Name = info.Name
			_ = syncProjectRegistry(a.Root)
		}
		res.Branch = client.CurrentBranch()
		raw, _, _ := a.gitStatusOutput([]string{"status", "--short", "--branch"})
		res.Raw = strings.TrimSpace(raw)
		for _, line := range strings.Split(res.Raw, "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "##") {
				res.Dirty = true
				break
			}
		}
	}
	if jsonOut {
		b, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	if !res.Initialized {
		return fmt.Errorf("not a kk repository")
	}
	fmt.Println(res.Raw)
	return nil
}
