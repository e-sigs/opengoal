package main

// Tests for roadmap helpers: active-roadmap state machine, migration of
// orphan list_ids, and the case-insensitive findRoadmap lookup.

import (
	"testing"
)

func TestEnsureActiveRoadmap_NoRoadmapsClearsActive(t *testing.T) {
	d := &GoalsData{ActiveRoadmapID: "stale-id"}
	mutated := ensureActiveRoadmap(d)
	if !mutated {
		t.Fatalf("expected mutation when active id refers to nothing")
	}
	if d.ActiveRoadmapID != "" {
		t.Fatalf("expected ActiveRoadmapID cleared, got %q", d.ActiveRoadmapID)
	}
}

func TestEnsureActiveRoadmap_NoChangeWhenValid(t *testing.T) {
	d := &GoalsData{
		Roadmaps:        []Roadmap{{ID: "a", Name: "a"}, {ID: "b", Name: "b"}},
		ActiveRoadmapID: "b",
	}
	if mutated := ensureActiveRoadmap(d); mutated {
		t.Fatalf("expected no mutation when active id is valid")
	}
	if d.ActiveRoadmapID != "b" {
		t.Fatalf("expected b, got %q", d.ActiveRoadmapID)
	}
}

func TestEnsureActiveRoadmap_FallsBackToFirst(t *testing.T) {
	d := &GoalsData{
		Roadmaps:        []Roadmap{{ID: "a"}, {ID: "b"}},
		ActiveRoadmapID: "ghost",
	}
	if mutated := ensureActiveRoadmap(d); !mutated {
		t.Fatalf("expected mutation when active id is dangling")
	}
	if d.ActiveRoadmapID != "a" {
		t.Fatalf("expected fallback to first roadmap 'a', got %q", d.ActiveRoadmapID)
	}
}

func TestMigrateOrphanRoadmapIDs(t *testing.T) {
	d := &GoalsData{
		Roadmaps:        []Roadmap{{ID: "active"}},
		ActiveRoadmapID: "active",
		MainGoals:       []MainGoal{{ID: "mg1"}, {ID: "mg2", RoadmapID: "other"}},
		SubGoals:        []SubGoal{{ID: "sg1"}},
		Tasks:           []Task{{ID: "t1"}, {ID: "t2", RoadmapID: "kept"}},
	}
	mutated := migrateOrphanRoadmapIDs(d)
	if !mutated {
		t.Fatalf("expected mutation since orphans existed")
	}
	if d.MainGoals[0].RoadmapID != "active" {
		t.Fatalf("mg1 should be migrated, got %q", d.MainGoals[0].RoadmapID)
	}
	if d.MainGoals[1].RoadmapID != "other" {
		t.Fatalf("mg2 should be untouched, got %q", d.MainGoals[1].RoadmapID)
	}
	if d.SubGoals[0].RoadmapID != "active" {
		t.Fatalf("sg1 should be migrated, got %q", d.SubGoals[0].RoadmapID)
	}
	if d.Tasks[0].RoadmapID != "active" {
		t.Fatalf("t1 should be migrated, got %q", d.Tasks[0].RoadmapID)
	}
	if d.Tasks[1].RoadmapID != "kept" {
		t.Fatalf("t2 should be untouched, got %q", d.Tasks[1].RoadmapID)
	}

	// Idempotent on a second pass.
	if migrateOrphanRoadmapIDs(d) {
		t.Fatalf("second migration pass should be a no-op")
	}
}

func TestMigrateOrphanRoadmapIDs_NoActiveDoesNothing(t *testing.T) {
	d := &GoalsData{Tasks: []Task{{ID: "t1"}}}
	if migrateOrphanRoadmapIDs(d) {
		t.Fatalf("should not migrate when there is no active roadmap")
	}
	if d.Tasks[0].RoadmapID != "" {
		t.Fatalf("RoadmapID should remain empty, got %q", d.Tasks[0].RoadmapID)
	}
}

func TestFindRoadmap(t *testing.T) {
	d := GoalsData{
		Roadmaps: []Roadmap{
			{ID: "id-1", Name: "Work"},
			{ID: "id-2", Name: "Personal"},
		},
	}

	if got := findRoadmap(d, "id-2"); got != 1 {
		t.Fatalf("by id: expected 1, got %d", got)
	}
	if got := findRoadmap(d, "personal"); got != 1 {
		t.Fatalf("case-insensitive name: expected 1, got %d", got)
	}
	if got := findRoadmap(d, "WORK"); got != 0 {
		t.Fatalf("upper-case name: expected 0, got %d", got)
	}
	if got := findRoadmap(d, "missing"); got != -1 {
		t.Fatalf("unknown: expected -1, got %d", got)
	}
}

func TestActiveRoadmapName(t *testing.T) {
	d := GoalsData{
		Roadmaps:        []Roadmap{{ID: "a", Name: "Alpha"}},
		ActiveRoadmapID: "a",
	}
	if got := activeRoadmapName(d); got != "Alpha" {
		t.Fatalf("expected Alpha, got %q", got)
	}

	d.ActiveRoadmapID = "missing"
	if got := activeRoadmapName(d); got != "(none)" {
		t.Fatalf("expected (none) for missing active id, got %q", got)
	}
}

func TestFilterByRoadmap(t *testing.T) {
	tasks := []Task{
		{ID: "1", RoadmapID: "a"},
		{ID: "2", RoadmapID: "b"},
		{ID: "3", RoadmapID: "a"},
	}
	got := filterByRoadmap(tasks, func(t Task) string { return t.RoadmapID }, "a")
	if len(got) != 2 || got[0].ID != "1" || got[1].ID != "3" {
		t.Fatalf("expected ids 1 and 3, got %+v", got)
	}
}
