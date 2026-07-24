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
	"errors"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestConcurrentFiles_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		files       []string
		workers     int
		fn          func(rel string) error
		expectError bool
		expectedErr error
	}{
		{
			name:        "Empty files list",
			files:       []string{},
			workers:     2,
			fn:          func(rel string) error { return nil },
			expectError: false,
		},
		{
			name:        "Fewer workers than total files",
			files:       []string{"file1", "file2", "file3", "file4"},
			workers:     2,
			fn:          func(rel string) error { return nil },
			expectError: false,
		},
		{
			name:        "More workers than total files",
			files:       []string{"file1", "file2"},
			workers:     10,
			fn:          func(rel string) error { return nil },
			expectError: false,
		},
		{
			name:        "Negative workers defaults to default count",
			files:       []string{"file1", "file2"},
			workers:     -1,
			fn:          func(rel string) error { return nil },
			expectError: false,
		},
		{
			name:    "Single file failure returns error",
			files:   []string{"file1"},
			workers: 2,
			fn: func(rel string) error {
				return errors.New("failed")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var startCount int32
			var doneCount int32
			var mu sync.Mutex
			startedFiles := make(map[string]bool)
			doneFiles := make(map[string]bool)

			onStart := func(workerID int, file string) {
				atomic.AddInt32(&startCount, 1)
				mu.Lock()
				startedFiles[file] = true
				mu.Unlock()
			}

			onDone := func(workerID, done, total int, file string) {
				atomic.AddInt32(&doneCount, 1)
				mu.Lock()
				doneFiles[file] = true
				mu.Unlock()
			}

			err := ConcurrentFiles(tt.files, tt.workers, onStart, onDone, tt.fn)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if int(startCount) != len(tt.files) {
					t.Errorf("expected %d started files, got %d", len(tt.files), startCount)
				}
				if int(doneCount) != len(tt.files) {
					t.Errorf("expected %d done files, got %d", len(tt.files), doneCount)
				}
				for _, f := range tt.files {
					if !startedFiles[f] {
						t.Errorf("file %s was not started", f)
					}
					if !doneFiles[f] {
						t.Errorf("file %s was not done", f)
					}
				}
			}
		})
	}
}

func TestConcurrentFiles_FastFail(t *testing.T) {
	t.Parallel()

	files := make([]string, 100)
	for i := range files {
		files[i] = "file_" + strconv.Itoa(i)
	}

	var callCount int32
	targetErr := errors.New("fatal sync error")

	err := ConcurrentFiles(files, 4, nil, nil, func(rel string) error {
		val := atomic.AddInt32(&callCount, 1)
		// Fail on the second call
		if val == 2 {
			return targetErr
		}
		// Introduce a tiny delay so context cancellation has time to propagate
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, targetErr) && err.Error() != "syncing file_1: fatal sync error" {
		t.Errorf("expected wrapped targetErr, got: %v", err)
	}

	finalCount := atomic.LoadInt32(&callCount)
	if finalCount >= 100 {
		t.Errorf("expected fast-fail to prevent processing all files, but processed %d files", finalCount)
	}
}

func TestConcurrentFiles_PanicSafety(t *testing.T) {
	t.Parallel()

	files := []string{"file1", "panic_file", "file2"}

	err := ConcurrentFiles(files, 2, nil, nil, func(rel string) error {
		if rel == "panic_file" {
			panic("something went critically wrong")
		}
		return nil
	})

	if err == nil {
		t.Fatal("expected error from panic recovery, got nil")
	}
	if !strings.Contains(err.Error(), "something went critically wrong") {
		t.Errorf("expected error message to contain panic context, got: %q", err.Error())
	}
}

func TestConcurrentFiles_ContextCancellation(t *testing.T) {
	t.Parallel()

	files := []string{"file1", "file2", "file3"}
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	err := ConcurrentFilesWithContext(ctx, files, 2, nil, nil, func(rel string) error {
		return nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func BenchmarkConcurrentFiles(b *testing.B) {
	files := make([]string, 10)
	for i := range files {
		files[i] = "file_" + strconv.Itoa(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ConcurrentFilesWithContext(context.Background(), files, 4,
			func(workerID int, file string) {
				// Step 6 — Profile: Place pprof labels inside callbacks/workers to identify hot paths
				pprof.Do(context.Background(), pprof.Labels("worker_id", strconv.Itoa(workerID), "file", file), func(ctx context.Context) {
					// Simulated CPU task inside pprof labeled region
				})
			},
			nil,
			func(rel string) error {
				return nil
			},
		)
	}
}

func FuzzConcurrentFiles(f *testing.F) {
	f.Add(1, 1)
	f.Add(4, 10)
	f.Add(-1, 5)

	f.Fuzz(func(t *testing.T, workers int, fileCount int) {
		if fileCount < 0 || fileCount > 100 {
			return
		}
		files := make([]string, fileCount)
		for i := range files {
			files[i] = "file_" + strconv.Itoa(i)
		}

		_ = ConcurrentFiles(files, workers, nil, nil, func(rel string) error {
			return nil
		})
	})
}
