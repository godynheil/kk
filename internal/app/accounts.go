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

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/godynheil/kk/internal/gdrive"
)

type AccountInfo struct {
	Profile      string `json:"profile"`
	Email        string `json:"email,omitempty"`
	DisplayName  string `json:"display_name,omitempty"`
	ClientID     string `json:"client_id"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`
}

func (a App) Accounts(args []string) error {
	jsonOut := hasFlag(args, "--json")
	deleteAll := hasFlag(args, "--delete-all")
	deleteProfile, err := stringFlag(args, "--delete")
	if err != nil {
		return err
	}
	if deleteAll && deleteProfile != "" {
		return fmt.Errorf("--delete and --delete-all cannot be used together")
	}

	if deleteAll {
		return deleteAllAccounts(jsonOut)
	}
	if deleteProfile != "" {
		return deleteAccount(deleteProfile, jsonOut)
	}

	dir := gdrive.AuthDir()
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			if jsonOut {
				fmt.Println("[]")
				return nil
			}
			fmt.Println("No accounts found in cache.")
			return nil
		}
		return fmt.Errorf("read accounts dir: %w", err)
	}

	accounts := []AccountInfo{}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		profileName := strings.TrimSuffix(file.Name(), ".json")
		filePath := filepath.Join(dir, file.Name())

		info := AccountInfo{
			Profile: profileName,
		}

		auth, err := gdrive.LoadAuth(filePath)
		if err != nil {
			info.Status = "Corrupted"
			info.ErrorMessage = err.Error()
			accounts = append(accounts, info)
			continue
		}

		info.ClientID = auth.ClientID
		info.Email = auth.Email
		info.DisplayName = auth.DisplayName

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		client := gdrive.NewClient(filePath)
		userInfo, err := client.FetchUserInfo(ctx)
		cancel()

		if err == nil {
			info.Status = "Active"
			info.Email = userInfo.Email
			info.DisplayName = userInfo.DisplayName
		} else {
			errStr := err.Error()
			isAuthErr := strings.Contains(errStr, "400") ||
				strings.Contains(errStr, "401") ||
				strings.Contains(errStr, "invalid_") ||
				strings.Contains(errStr, "unauthorized")

			if isAuthErr {
				info.Status = "Invalid Credentials"
				info.ErrorMessage = errStr
			} else {
				if time.Now().Before(auth.Expiry.Add(-30*time.Second)) && auth.AccessToken != "" {
					info.Status = "Active (Offline)"
				} else {
					info.Status = "Offline (Expired)"
				}
				info.ErrorMessage = errStr
			}
		}

		accounts = append(accounts, info)
	}

	if jsonOut {
		b, err := json.MarshalIndent(accounts, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}

	if len(accounts) == 0 {
		fmt.Println("No accounts found in cache.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(w, "Profile\tEmail\tDisplay Name\tClient ID\tStatus")
	_, _ = fmt.Fprintln(w, "-------\t-----\t------------\t---------\t------")
	for _, acc := range accounts {
		email := acc.Email
		if email == "" {
			email = "-"
		}
		dn := acc.DisplayName
		if dn == "" {
			dn = "-"
		}
		cid := acc.ClientID
		if len(cid) > 20 {
			cid = cid[:17] + "..."
		} else if cid == "" {
			cid = "-"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", acc.Profile, email, dn, cid, acc.Status)
	}
	_ = w.Flush()

	return nil
}

type deleteAccountsResult struct {
	Deleted int    `json:"deleted"`
	Profile string `json:"profile,omitempty"`
}

func deleteAccount(profile string, jsonOut bool) error {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return fmt.Errorf("--delete requires a profile")
	}
	if strings.ContainsAny(profile, `/\`) || profile == "." || profile == ".." {
		return fmt.Errorf("invalid profile %q", profile)
	}

	path := gdrive.DefaultAuthPath(profile)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("account %q not found", profile)
		}
		return fmt.Errorf("delete account %q: %w", profile, err)
	}

	if jsonOut {
		b, err := json.Marshal(deleteAccountsResult{Deleted: 1, Profile: profile})
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}

	fmt.Printf("Deleted account %q from cache.\n", profile)
	return nil
}

func deleteAllAccounts(jsonOut bool) error {
	dir := gdrive.AuthDir()
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			if jsonOut {
				fmt.Println(`{"deleted":0}`)
			} else {
				fmt.Println("No accounts found in cache.")
			}
			return nil
		}
		return fmt.Errorf("read accounts dir: %w", err)
	}

	deleted := 0
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		if err := os.Remove(filepath.Join(dir, file.Name())); err != nil {
			return fmt.Errorf("delete account %q: %w", strings.TrimSuffix(file.Name(), ".json"), err)
		}
		deleted++
	}

	if jsonOut {
		b, err := json.Marshal(deleteAccountsResult{Deleted: deleted})
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}

	if deleted == 0 {
		fmt.Println("No accounts found in cache.")
		return nil
	}

	fmt.Printf("Deleted %d account(s) from cache.\n", deleted)
	return nil
}
