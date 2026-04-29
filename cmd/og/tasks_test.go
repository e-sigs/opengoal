package main

// Tests for task-level pure helpers: priority sorting, calculateProgress,
// and the bulkArgs flag parser.

import (
	"testing"
	"time"
)

func TestSortTasksForDisplay(t *testing.T) {
	t0 := time.Now()
	tasks := []Task{
		{ID: "a", Priority: PriorityLow, Created: t0.Add(0)},
		{ID: "b", Priority: PriorityHigh, Created: t0.Add(2 * time.Second)},
		{ID: "c", Priority: PriorityMedium, Created: t0.Add(1 * time.Second)},
		{ID: "d", Priority: "", Created: t0.Add(3 * time.Second)},
		{ID: "e", Priority: PriorityHigh, Created: t0.Add(1 * time.Second)},
	}
	sortTasksForDisplay(tasks)

	wantOrder := []string{"e", "b", "c", "a", "d"} // high(by created) → med → low → none
	for i, exp := range wantOrder {
		if tasks[i].ID != exp {
			ids := make([]string, len(tasks))
			for j, x := range tasks {
				ids[j] = x.ID
			}
			t.Fatalf("position %d: expected %q, got order %v", i, exp, ids)
		}
	}
}

func TestCalculateProgress(t *testing.T) {
	d := GoalsData{
		MainGoals: []MainGoal{{ID: "mg1"}},
		SubGoals: []SubGoal{
			{ID: "s1", ParentID: "mg1", Status: StatusCompleted},
			{ID: "s2", ParentID: "mg1", Status: StatusPending},
			{ID: "s3", ParentID: "mg1", Status: StatusCompleted},
			{ID: "s4", ParentID: "other", Status: StatusCompleted},
		},
	}
	if got := calculateProgress("mg1", d); got != 66 {
		t.Fatalf("expected 66 (2 of 3 done, integer trunc), got %d", got)
	}
	if got := calculateProgress("no-children", d); got != 0 {
		t.Fatalf("expected 0 for no children, got %d", got)
	}
}

func TestParseBulkArgs(t *testing.T) {
	t.Run("flags + ids", func(t *testing.T) {
		b := parseBulkArgs([]string{"task-1", "-y", "task-2", "--priority", "high"}, nil)
		if !b.yes {
			t.Errorf("expected yes")
		}
		if b.priority != PriorityHigh {
			t.Errorf("expected priority high, got %q", b.priority)
		}
		if len(b.ids) != 2 || b.ids[0] != "task-1" || b.ids[1] != "task-2" {
			t.Errorf("ids: %v", b.ids)
		}
	})

	t.Run("--all", func(t *testing.T) {
		b := parseBulkArgs([]string{"--all"}, nil)
		if !b.all {
			t.Errorf("expected all")
		}
	})

	t.Run("allowed filter", func(t *testing.T) {
		b := parseBulkArgs([]string{"--completed"}, map[string]bool{"completed": true})
		if b.filter != "completed" {
			t.Errorf("filter: %q", b.filter)
		}
	})
}
