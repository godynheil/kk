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

package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

var remoteNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}$`)

func DefaultConfig() LocalConfig {
	return LocalConfig{
		Version:       ConfigVersion,
		DefaultRemote: "",
		Remotes:       map[string]RemoteConfig{},
	}
}

func ValidateRemoteName(name string) error {
	if name == "" {
		return fmt.Errorf("remote name is required")
	}

	reserved := []string{".", "..", "main"}
	for _, r := range reserved {
		if name == r {
			return fmt.Errorf("remote name %q is reserved", name)
		}
	}
	if !remoteNameRe.MatchString(name) {
		return fmt.Errorf("invalid remote name %q; use 1-64 characters with letters, numbers, dots, underscores, or dashes", name)
	}
	return nil
}

func ReadConfig(root string) (LocalConfig, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(filepath.Join(root, ConfigFile)) // #nosec G304 -- config is read from the caller's repository root.
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	if cfg.Remotes == nil {
		cfg.Remotes = map[string]RemoteConfig{}
	}
	return cfg, nil
}

func WriteConfig(root string, cfg LocalConfig) error {
	if cfg.Version == "" {
		cfg.Version = ConfigVersion
	}
	if cfg.Remotes == nil {
		cfg.Remotes = map[string]RemoteConfig{}
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(root, ConfigFile), data, 0o600)
}

func (c LocalConfig) GetRemote(name string) (RemoteConfig, string, error) {
	if name == "" {
		name = c.DefaultRemote
	}
	if name == "" {
		return RemoteConfig{}, "", fmt.Errorf("no remote specified and no default_remote configured")
	}
	remote, ok := c.Remotes[name]
	if !ok {
		return RemoteConfig{}, "", fmt.Errorf("remote not found: %s", name)
	}
	return remote, name, nil
}

func (c LocalConfig) GitRemotes() []NamedRemoteConfig {
	var out []NamedRemoteConfig
	for name, cfg := range c.Remotes {
		if cfg.Type == "git" {
			out = append(out, NamedRemoteConfig{Name: name, Config: cfg})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (c LocalConfig) HasOnlyGitRemotes() bool {
	if len(c.Remotes) == 0 {
		return false
	}
	for _, r := range c.Remotes {
		if r.Type != "git" {
			return false
		}
	}
	return true
}

func (c LocalConfig) PullRemotes() []NamedRemoteConfig {
	var out []NamedRemoteConfig
	for name, cfg := range c.Remotes {
		if cfg.Type == "git" {

			continue
		}
		if cfg.Pull {
			out = append(out, NamedRemoteConfig{Name: name, Config: cfg})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Name == c.DefaultRemote {
			return true
		}
		if out[j].Name == c.DefaultRemote {
			return false
		}
		return out[i].Config.Priority < out[j].Config.Priority
	})
	return out
}

func (c LocalConfig) PushRemotes(names []string, all bool) ([]NamedRemoteConfig, error) {
	if all {
		var out []NamedRemoteConfig
		for name, cfg := range c.Remotes {
			if cfg.Type == "git" {

				continue
			}
			if cfg.Push {
				out = append(out, NamedRemoteConfig{Name: name, Config: cfg})
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Config.Priority < out[j].Config.Priority })
		return out, nil
	}

	if len(names) == 0 {
		cfg, name, err := c.GetRemote("")
		if err != nil {
			return nil, err
		}
		if cfg.Type == "git" {

			var out []NamedRemoteConfig
			for n, r := range c.Remotes {
				if r.Type != "git" && r.Push {
					out = append(out, NamedRemoteConfig{Name: n, Config: r})
				}
			}
			sort.Slice(out, func(i, j int) bool { return out[i].Config.Priority < out[j].Config.Priority })
			return out, nil
		}
		if !cfg.Push {
			out := c.pushEnabledObjectRemotesExcept(name)
			if len(out) > 0 {
				return out, nil
			}
			return nil, fmt.Errorf("default remote %s is not push-enabled", name)
		}
		return []NamedRemoteConfig{{Name: name, Config: cfg}}, nil
	}

	var out []NamedRemoteConfig
	for _, name := range names {
		cfg, resolved, err := c.GetRemote(name)
		if err != nil {
			return nil, err
		}
		if cfg.Type == "git" {

			continue
		}
		if !cfg.Push {
			return nil, fmt.Errorf("remote %s is not push-enabled", resolved)
		}
		out = append(out, NamedRemoteConfig{Name: resolved, Config: cfg})
	}
	return out, nil
}

func (c LocalConfig) pushEnabledObjectRemotesExcept(excluded string) []NamedRemoteConfig {
	var out []NamedRemoteConfig
	for name, cfg := range c.Remotes {
		if name == excluded || cfg.Type == "git" || !cfg.Push {
			continue
		}
		out = append(out, NamedRemoteConfig{Name: name, Config: cfg})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == c.DefaultRemote {
			return true
		}
		if out[j].Name == c.DefaultRemote {
			return false
		}
		return out[i].Config.Priority < out[j].Config.Priority
	})
	return out
}
