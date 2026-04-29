//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"syscall"
)

// withLock takes an exclusive advisory flock on goals.json.lock for the
// duration of fn. The lock file is created if missing and is never
// removed (removing it would race with other holders). Blocks until the
// lock is acquired.
func withLock(fn func()) {
	path := getGoalsFilePath() + ".lock"
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		fatalf("Error creating lock dir: %v", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		fatalf("Error opening lock file: %v", err)
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		fatalf("Error acquiring lock: %v", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	fn()
}
