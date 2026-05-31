package app

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/godynheil/kk/internal/git"
)

var BuildVersion = "dev"

var BuildDate = ""

var BuildDateLocal = ""

func (a App) Version() error {
	display := strings.TrimSpace(BuildVersion)
	if display == "" {
		display = "dev"
	}

	if display == "dev" {
		if gp, err := git.GitExecutable(); err == nil {
			if out, err := exec.Command(gp, "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil { // #nosec G204 -- gp is resolved by GitExecutable; arguments are fixed.
				branch := strings.TrimSpace(string(out))
				if branch != "" {
					display = strings.ReplaceAll(branch, "/", "-")
				}
			}
		}
	}

	display = strings.ReplaceAll(display, "/", "-")

	utc := strings.TrimSpace(BuildDate)
	if utc != "" {
		utcFmt := strings.Replace(utc, "T", " ", 1)
		utcFmt = strings.TrimSuffix(utcFmt, "Z") + " UTC"

		local := strings.TrimSpace(BuildDateLocal)
		if local != "" {
			localFmt := strings.Replace(local, "T", " ", 1)
			fmt.Printf("kk %s (built %s | %s)\n", display, utcFmt, localFmt)
		} else {
			fmt.Printf("kk %s (built %s)\n", display, utcFmt)
		}
	} else {
		fmt.Printf("kk %s\n", display)
	}
	if gv, err := gitVersion(); err == nil {
		fmt.Println(gv)
	}
	return nil
}

func gitVersion() (string, error) {
	gitPath, err := git.GitExecutable()
	if err != nil {
		return "", fmt.Errorf("get kk history engine version: %w", err)
	}
	out, err := exec.Command(gitPath, "--version").Output() // #nosec G204 -- gitPath is resolved by GitExecutable; arguments are fixed.
	if err != nil {
		return "", fmt.Errorf("get kk history engine version: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
