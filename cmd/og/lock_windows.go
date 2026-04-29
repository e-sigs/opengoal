//go:build windows

package main

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// withLock takes an exclusive lock on goals.json.lock for the duration of fn,
// using LockFileEx on Windows. Blocks until the lock is acquired.
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

	handle := windows.Handle(f.Fd())
	overlapped := new(windows.Overlapped)

	// LOCKFILE_EXCLUSIVE_LOCK with no LOCKFILE_FAIL_IMMEDIATELY ⇒ blocking.
	if err := windows.LockFileEx(
		handle,
		windows.LOCKFILE_EXCLUSIVE_LOCK,
		0,
		^uint32(0), ^uint32(0),
		overlapped,
	); err != nil {
		fatalf("Error acquiring lock: %v", err)
	}
	defer windows.UnlockFileEx(handle, 0, ^uint32(0), ^uint32(0), overlapped)

	fn()
}
