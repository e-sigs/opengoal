package main

// Tests for claim/dep helpers. These exercise the pure logic (claimActive,
// blockedDeps, claimStatusLabel) without invoking the *Task* commands
// that call os.Exit on errors.

import (
	"testing"
	"time"
)

func TestClaimActive(t *testing.T) {
	ttl := 30 * time.Minute

	tests := []struct {
		name string
		t    Task
		want bool
	}{
		{
			name: "completed task is not active",
			t:    Task{Completed: true, Assignee: "a", ClaimedAt: ptrTime(time.Now())},
			want: false,
		},
		{
			name: "no assignee is not active",
			t:    Task{ClaimedAt: ptrTime(time.Now())},
			want: false,
		},
		{
			name: "no claimed_at is not active",
			t:    Task{Assignee: "a"},
			want: false,
		},
		{
			name: "fresh claim is active",
			t:    Task{Assignee: "a", ClaimedAt: ptrTime(time.Now().Add(-1 * time.Minute))},
			want: true,
		},
		{
			name: "expired (stale) claim is not active",
			t:    Task{Assignee: "a", ClaimedAt: ptrTime(time.Now().Add(-2 * time.Hour))},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := claimActive(tt.t, ttl)
			if got != tt.want {
				t.Fatalf("claimActive(%+v) = %v, want %v", tt.t, got, tt.want)
			}
		})
	}
}

func TestClaimTTLEnvOverride(t *testing.T) {
	t.Setenv("OPENGOAL_CLAIM_TTL", "60")
	if got := claimTTL(); got != 60*time.Second {
		t.Fatalf("expected 60s, got %v", got)
	}
}

func TestClaimTTLDefaultsOnInvalid(t *testing.T) {
	t.Setenv("OPENGOAL_CLAIM_TTL", "garbage")
	if got := claimTTL(); got != 1800*time.Second {
		t.Fatalf("expected default 1800s on invalid input, got %v", got)
	}

	t.Setenv("OPENGOAL_CLAIM_TTL", "0")
	if got := claimTTL(); got != 1800*time.Second {
		t.Fatalf("expected default 1800s on zero input, got %v", got)
	}

	t.Setenv("OPENGOAL_CLAIM_TTL", "-5")
	if got := claimTTL(); got != 1800*time.Second {
		t.Fatalf("expected default 1800s on negative input, got %v", got)
	}
}

func TestAgentIDEnvOverride(t *testing.T) {
	t.Setenv("OPENGOAL_AGENT", "  alice  ") // whitespace trimmed
	if got := agentID(); got != "alice" {
		t.Fatalf("expected alice, got %q", got)
	}
}

func TestAgentIDFallback(t *testing.T) {
	t.Setenv("OPENGOAL_AGENT", "")
	got := agentID()
	if got == "" {
		t.Fatalf("agentID fallback should never be empty")
	}
}

func TestBlockedDeps(t *testing.T) {
	tasks := []Task{
		{ID: "a", Completed: true},
		{ID: "b", Completed: false, Title: "b"},
		{ID: "c", Completed: false, Title: "c", Assignee: "worker", ClaimedAt: ptrTime(time.Now())},
	}

	t.Run("no deps means not blocked", func(t *testing.T) {
		got := blockedDeps(Task{}, tasks)
		if len(got) != 0 {
			t.Fatalf("expected no blocking, got %v", got)
		}
	})

	t.Run("completed dep is satisfied", func(t *testing.T) {
		got := blockedDeps(Task{DependsOn: []string{"a"}}, tasks)
		if len(got) != 0 {
			t.Fatalf("expected no blocking, got %v", got)
		}
	})

	t.Run("pending dep blocks", func(t *testing.T) {
		got := blockedDeps(Task{DependsOn: []string{"b"}}, tasks)
		if len(got) != 1 {
			t.Fatalf("expected one blocker, got %v", got)
		}
	})

	t.Run("missing dep blocks with marker", func(t *testing.T) {
		got := blockedDeps(Task{DependsOn: []string{"zzz"}}, tasks)
		if len(got) != 1 {
			t.Fatalf("expected one blocker, got %v", got)
		}
		if !contains(got[0], "missing") {
			t.Fatalf("expected 'missing' annotation, got %q", got[0])
		}
	})

	t.Run("claim on dep does not satisfy dep", func(t *testing.T) {
		// Only Completed satisfies a dep — having a live claim on it is
		// not enough; the work must be done.
		got := blockedDeps(Task{DependsOn: []string{"c"}}, tasks)
		if len(got) != 1 {
			t.Fatalf("expected dep with live claim still to block, got %v", got)
		}
	})

	t.Run("multiple deps with mixed status", func(t *testing.T) {
		got := blockedDeps(Task{DependsOn: []string{"a", "b", "zzz"}}, tasks)
		if len(got) != 2 {
			t.Fatalf("expected exactly 2 unsatisfied (b + zzz), got %v", got)
		}
	})
}

func TestFindTaskByID(t *testing.T) {
	tasks := []Task{
		{ID: "a", Title: "first"},
		{ID: "b", Title: "second"},
	}
	if got := findTaskByID(tasks, "b"); got == nil || got.Title != "second" {
		t.Fatalf("expected to find b, got %+v", got)
	}
	if got := findTaskByID(tasks, "missing"); got != nil {
		t.Fatalf("expected nil for missing, got %+v", got)
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func contains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && indexOf(s, substr) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
