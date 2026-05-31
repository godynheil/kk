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
