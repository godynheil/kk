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

package remote

import (
	"context"
	"fmt"
	"sync"
)

const SyncWorkers = 4

func ConcurrentFiles(
	files []string,
	workers int,
	onStart func(workerID int, file string),
	onDone func(workerID, done, total int, file string),
	fn func(rel string) error,
) error {
	return ConcurrentFilesWithContext(context.Background(), files, workers, onStart, onDone, fn)
}

func ConcurrentFilesWithContext(
	ctx context.Context,
	files []string,
	workers int,
	onStart func(workerID int, file string),
	onDone func(workerID, done, total int, file string),
	fn func(rel string) error,
) error {
	total := len(files)
	if total == 0 {
		return nil
	}
	if workers <= 0 {
		workers = SyncWorkers
	}
	if workers > total {
		workers = total
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type result struct {
		workerID int
		rel      string
		err      error
	}

	jobs := make(chan string)
	results := make(chan result, workers*2)

	var wg sync.WaitGroup

	for id := 0; id < workers; id++ {
		id := id
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case rel, ok := <-jobs:
					if !ok {
						return
					}
					if onStart != nil {
						onStart(id, rel)
					}

					var err error
					func() {
						defer func() {
							if r := recover(); r != nil {
								err = fmt.Errorf("panic processing %s: %v", rel, r)
							}
						}()
						err = fn(rel)
					}()

					select {
					case <-ctx.Done():
						return
					case results <- result{workerID: id, rel: rel, err: err}:
					}
				}
			}
		}()
	}

	var feederWg sync.WaitGroup
	feederWg.Add(1)
	go func() {
		defer feederWg.Done()
		defer close(jobs)
		for _, f := range files {
			select {
			case <-ctx.Done():
				return
			case jobs <- f:
			}
		}
	}()

	go func() {
		feederWg.Wait()
		wg.Wait()
		close(results)
	}()

	var (
		firstErr error
		done     int
	)

	for {
		select {
		case <-ctx.Done():

			if firstErr != nil {
				return firstErr
			}
			return ctx.Err()
		case r, ok := <-results:
			if !ok {

				return firstErr
			}
			if r.err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("syncing %s: %w", r.rel, r.err)
					cancel()
				}
				continue
			}
			done++
			if onDone != nil {
				onDone(r.workerID, done, total, r.rel)
			}
		}
	}
}
