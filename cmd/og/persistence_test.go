package main

// Tests for the persistence layer: writeGoals/readGoals atomicity and
// the withLock cross-process serializer.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// withTempHome redirects $HOME to a per-test temp directory so the data
// path resolution in getGoalsFilePath() touches an isolated tree. It
// also primes the data directory so writeGoals can create temp files.
func withTempHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	ensureGoalsFile()
	return home
}

func TestWriteReadRoundTrip(t *testing.T) {
	withTempHome(t)

	want := GoalsData{
		Roadmaps: []Roadmap{{ID: "list-1", Name: "test"}},
		ActiveRoadmapID: "list-1",
		MainGoals: []MainGoal{{ID: "mg-1", RoadmapID: "list-1", Title: "ship", Status: StatusInProgress}},
		Tasks: []Task{{ID: "task-1", RoadmapID: "list-1", Title: "do thing"}},
	}
	writeGoals(want)

	got := readGoals()
	if len(got.Roadmaps) != 1 || got.Roadmaps[0].ID != "list-1" {
		t.Fatalf("roadmap not round-tripped: %+v", got.Roadmaps)
	}
	if len(got.MainGoals) != 1 || got.MainGoals[0].Title != "ship" {
		t.Fatalf("main goal not round-tripped: %+v", got.MainGoals)
	}
	if len(got.Tasks) != 1 || got.Tasks[0].Title != "do thing" {
		t.Fatalf("task not round-tripped: %+v", got.Tasks)
	}
}

// TestWriteAtomicNoLeftoverTemps verifies writeGoals() does not leave
// `.tmp` siblings behind in the data directory after a successful write.
func TestWriteAtomicNoLeftoverTemps(t *testing.T) {
	home := withTempHome(t)

	writeGoals(GoalsData{
		Roadmaps:        []Roadmap{{ID: "x", Name: "x"}},
		ActiveRoadmapID: "x",
	})

	dir := filepath.Join(home, ".local", "share", "opencode")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Fatalf("temp file leaked: %s", e.Name())
		}
	}
}

// TestWriteOverwritesAtomically verifies that a second writeGoals fully
// replaces the on-disk file rather than appending.
func TestWriteOverwritesAtomically(t *testing.T) {
	home := withTempHome(t)
	path := filepath.Join(home, ".local", "share", "opencode", "goals.json")

	writeGoals(GoalsData{Roadmaps: []Roadmap{{ID: "a", Name: "first"}}, ActiveRoadmapID: "a"})
	writeGoals(GoalsData{Roadmaps: []Roadmap{{ID: "b", Name: "second"}}, ActiveRoadmapID: "b"})

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("readfile: %v", err)
	}
	var data GoalsData
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("unmarshal: %v\nraw: %s", err, raw)
	}
	if len(data.Roadmaps) != 1 || data.Roadmaps[0].ID != "b" {
		t.Fatalf("expected only second roadmap, got %+v", data.Roadmaps)
	}
}

// TestWithLockSerializesIncrement verifies withLock prevents lost updates
// across many concurrent goroutines doing read-modify-write on the
// shared GoalsData. Without the lock, the count would be < N.
func TestWithLockSerializesIncrement(t *testing.T) {
	withTempHome(t)

	// Seed one roadmap so requireActiveRoadmap-style code paths would work.
	writeGoals(GoalsData{Roadmaps: []Roadmap{{ID: "r", Name: "r"}}, ActiveRoadmapID: "r"})

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			withLock(func() {
				data := readGoals()
				// Use Tasks slice length as a counter.
				data.Tasks = append(data.Tasks, Task{
					ID: generateID("task"), RoadmapID: "r", Title: "x",
				})
				writeGoals(data)
			})
		}()
	}
	wg.Wait()

	final := readGoals()
	if len(final.Tasks) != n {
		t.Fatalf("expected %d tasks after concurrent writes, got %d", n, len(final.Tasks))
	}
}
