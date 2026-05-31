package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/godynheil/kk/internal/core"
)

type pushState struct {
	Version string                     `json:"version"`
	Remotes map[string]remotePushState `json:"remotes"`
}

type remotePushState struct {
	HeadCommit string    `json:"head_commit"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func readPushState(root string) (pushState, error) {
	state := pushState{
		Version: "kk-push-state-1.0.0",
		Remotes: map[string]remotePushState{},
	}
	data, err := os.ReadFile(filepath.Join(root, core.PushStateFile)) // #nosec G304 -- push state is read from the caller's repository root.
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
		state.Remotes = map[string]remotePushState{}
	}
	return state, nil
}

func writePushState(root string, state pushState) error {
	if state.Version == "" {
		state.Version = "kk-push-state-1.0.0"
	}
	if state.Remotes == nil {
		state.Remotes = map[string]remotePushState{}
	}
	path := filepath.Join(root, core.PushStateFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}
