package app

import (
	"os"
	"strconv"
)

const defaultWorkerCount = 4

func resolveWorkers(flagVal int) int {
	if flagVal > 0 {
		return flagVal
	}
	if v := os.Getenv("KK_WORKERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultWorkerCount
}
