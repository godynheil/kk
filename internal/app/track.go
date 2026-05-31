package app

import (
	"fmt"

	"github.com/godynheil/kk/internal/core"
)

func (a App) Track(patterns []string) error {
	if len(patterns) == 0 {
		return fmt.Errorf("usage: kk track <pattern...>")
	}
	tracks, err := core.ReadTracks(a.Root)
	if err != nil {
		return err
	}
	for _, pattern := range patterns {
		tracks = core.AddTrackPattern(tracks, pattern)
		fmt.Println("tracking", pattern)
	}
	return core.WriteTracks(a.Root, tracks)
}

func (a App) Untrack(patterns []string) error {
	if len(patterns) == 0 {
		return fmt.Errorf("usage: kk untrack <pattern...>")
	}
	tracks, err := core.ReadTracks(a.Root)
	if err != nil {
		return err
	}
	for _, pattern := range patterns {
		tracks = core.RemoveTrackPattern(tracks, pattern)
		fmt.Println("untracked", pattern)
	}
	return core.WriteTracks(a.Root, tracks)
}

func (a App) TrackList() error {
	tracks, err := core.ReadTracks(a.Root)
	if err != nil {
		return err
	}
	if len(tracks.Patterns) == 0 {
		fmt.Println("no tracked patterns")
		return nil
	}
	for _, pattern := range tracks.Patterns {
		fmt.Println(pattern)
	}
	return nil
}
