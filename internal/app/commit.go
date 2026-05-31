package app

import (
	"fmt"

	"github.com/godynheil/kk/internal/core"
	"github.com/godynheil/kk/internal/git"
)

func (a App) Commit(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: kk commit -m <message>")
	}
	client := git.New(a.Root)
	if err := client.EnsureRepository(); err != nil {
		return err
	}
	if err := a.validateStagedLargeFiles(); err != nil {
		return err
	}
	return client.Run(append([]string{"commit"}, args...)...)
}

func (a App) validateStagedLargeFiles() error {
	client := git.New(a.Root)
	tracks, err := core.ReadTracks(a.Root)
	if err != nil {
		return err
	}
	files, err := client.StagedFiles()
	if err != nil {
		return err
	}
	for _, file := range files {
		if !core.ShouldTrack(file, tracks) {
			continue
		}
		data, err := client.ShowIndexFile(file)
		if err != nil {
			continue
		}
		if _, ok := core.ParsePointerBytes(data); ok {
			continue
		}
		return fmt.Errorf("staged large file is not a kk pointer: %s; use kk add %s", file, file)
	}
	return nil
}
