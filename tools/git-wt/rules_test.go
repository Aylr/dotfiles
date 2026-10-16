package main

import (
	"reflect"
	"testing"
)

func mergedClean() Worktree {
	return Worktree{Branch: "feat/x", PR: &PR{Number: 3, State: "MERGED"}}
}

func TestBlockedReasons(t *testing.T) {
	live := []Session{{Live: true}, {Live: true}}
	tests := []struct {
		name string
		w    Worktree
		want []string
	}{
		{"none", Worktree{Branch: "feat/x"}, nil},
		{"main", Worktree{Main: true}, []string{"main worktree"}},
		{"current", Worktree{Current: true}, []string{"you are in this worktree"}},
		{"default branch", Worktree{Branch: "main"}, []string{"default branch"}},
		{"open PR", Worktree{PR: &PR{Number: 618, State: "OPEN"}}, []string{"PR #618 is open"}},
		{"draft PR", Worktree{PR: &PR{Number: 744, State: "OPEN", Draft: true}}, []string{"PR #744 is a draft"}},
		{"merged PR", Worktree{PR: &PR{Number: 1, State: "MERGED"}}, nil},
		{"live session", Worktree{Sessions: live}, []string{"live Claude session"}},
		{"stale session", Worktree{Sessions: []Session{{}}}, nil},
		{"locked", Worktree{Locked: true}, []string{"locked"}},
		{"detached has no default branch", Worktree{Detached: true}, nil},
		{"all in order", Worktree{
			Main: true, Current: true, Branch: "main", Locked: true, Sessions: live,
			PR: &PR{Number: 4, State: "OPEN"},
		}, []string{"main worktree", "you are in this worktree", "default branch", "PR #4 is open", "live Claude session", "locked"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BlockedReasons(tt.w, "main")
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPreMark(t *testing.T) {
	with := func(f func(*Worktree)) Worktree {
		w := mergedClean()
		f(&w)
		return w
	}
	tests := []struct {
		name string
		w    Worktree
		want bool
	}{
		{"merged clean", mergedClean(), true},
		{"closed clean", Worktree{Branch: "x", PR: &PR{State: "CLOSED"}}, true},
		{"merged dirty", with(func(w *Worktree) { w.DirtyFiles = []string{"a"} }), false},
		{"merged untracked", with(func(w *Worktree) { w.Untracked = []string{"a"} }), false},
		{"merged lost", with(func(w *Worktree) { w.Lost = 2 }), false},
		{"detached reachable", Worktree{Detached: true}, true},
		{"detached orphaned", Worktree{Detached: true, Orphaned: true}, false},
		{"open PR", Worktree{Branch: "x", PR: &PR{State: "OPEN"}}, false},
		{"live session", with(func(w *Worktree) { w.Sessions = []Session{{Live: true}} }), false},
		{"main", with(func(w *Worktree) { w.Main = true }), false},
		{"current", with(func(w *Worktree) { w.Current = true }), false},
		{"locked", with(func(w *Worktree) { w.Locked = true }), false},
		{"default branch", with(func(w *Worktree) { w.Branch = "main" }), false},
		{"no PR", Worktree{Branch: "x"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PreMark(tt.w, "main"); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLosses(t *testing.T) {
	tests := []struct {
		name string
		w    Worktree
		want []string
	}{
		{"nothing", Worktree{}, nil},
		{"singular", Worktree{DirtyFiles: []string{"a"}, Untracked: []string{"b"}, Lost: 1},
			[]string{"1 dirty file", "1 untracked file", "1 commit exists only here"}},
		{"plural", Worktree{DirtyFiles: []string{"a", "b"}, Untracked: []string{"c", "d", "e"}, Lost: 4},
			[]string{"2 dirty files", "3 untracked files", "4 commits exist only here"}},
		{"orphaned", Worktree{Detached: true, Orphaned: true}, []string{"orphaned detached HEAD"}},
		{"remote", Worktree{Branch: "foo/bar", Upstream: "origin/foo/bar"}, []string{"remote origin/foo/bar"}},
		{"foreign upstream", Worktree{Branch: "feat", Upstream: "origin/main"}, nil},
		{"order", Worktree{DirtyFiles: []string{"a"}, Untracked: []string{"b"}, Lost: 2, Orphaned: true, Branch: "x", Upstream: "origin/x"},
			[]string{"1 dirty file", "1 untracked file", "2 commits exist only here", "orphaned detached HEAD", "remote origin/x"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Losses(tt.w)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
