package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

type delFixture struct {
	repo   Repo
	origin string
}

func delSetup(t *testing.T) delFixture {
	t.Helper()
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	origin := filepath.Join(tmp, "origin.git")
	clone := filepath.Join(tmp, "clone")
	git(t, tmp, "init", "--bare", "-b", "main", origin)
	git(t, tmp, "clone", origin, clone)
	git(t, clone, "checkout", "-B", "main")
	git(t, clone, "commit", "--allow-empty", "-m", "init")
	git(t, clone, "push", "-u", "origin", "main")
	return delFixture{Repo{Root: clone, Name: "clone", DefaultBranch: "main", Base: "origin/main"}, origin}
}

func (f delFixture) addWorktree(t *testing.T, name string, pushed bool) Worktree {
	t.Helper()
	path := filepath.Join(f.repo.Root, ".worktrees", name)
	git(t, f.repo.Root, "worktree", "add", "-b", name, path, "main")
	w := Worktree{Path: path, Name: name, Branch: name}
	if pushed {
		git(t, path, "push", "-u", "origin", name)
		w.Upstream = "origin/" + name
	}
	return w
}

func delRunAll(r Repo, w Worktree) []error {
	var errs []error
	for _, s := range DeleteSteps(w) {
		err := s.Run(r)
		errs = append(errs, err)
		if err != nil {
			break
		}
	}
	return errs
}

func delRequireOK(t *testing.T, errs []error) {
	t.Helper()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
	}
}

func delExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func delHasRef(t *testing.T, dir, ref string) bool {
	t.Helper()
	return git(t, dir, "for-each-ref", ref) != ""
}

func TestDeleteCleanBranchWithUpstream(t *testing.T) {
	f := delSetup(t)
	w := f.addWorktree(t, "feat", true)
	delRequireOK(t, delRunAll(f.repo, w))
	if delExists(w.Path) {
		t.Error("worktree dir still exists")
	}
	if delHasRef(t, f.repo.Root, "refs/heads/feat") {
		t.Error("local branch still exists")
	}
	if delHasRef(t, f.origin, "refs/heads/feat") {
		t.Error("origin branch still exists")
	}
}

func TestDeleteSlashedBranchUpstream(t *testing.T) {
	f := delSetup(t)
	w := f.addWorktree(t, "user/feat", true)
	delRequireOK(t, delRunAll(f.repo, w))
	if delHasRef(t, f.origin, "refs/heads/user/feat") {
		t.Error("origin branch still exists")
	}
}

func TestDeleteDirtyWorktreeForced(t *testing.T) {
	f := delSetup(t)
	w := f.addWorktree(t, "dirty", false)
	if err := os.WriteFile(filepath.Join(w.Path, "new.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, w.Path, "add", "new.txt")
	git(t, w.Path, "commit", "-m", "add")
	if err := os.WriteFile(filepath.Join(w.Path, "new.txt"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(w.Path, "untracked.txt"), []byte("u"), 0o644); err != nil {
		t.Fatal(err)
	}
	w.DirtyFiles = []string{"new.txt"}
	w.Untracked = []string{"untracked.txt"}
	delRequireOK(t, delRunAll(f.repo, w))
	if delExists(w.Path) {
		t.Error("worktree dir still exists")
	}
}

func TestDeleteUpstreamAlreadyGone(t *testing.T) {
	f := delSetup(t)
	w := f.addWorktree(t, "gone", true)
	git(t, f.origin, "branch", "-D", "gone")
	git(t, f.repo.Root, "fetch", "--prune")
	steps := DeleteSteps(w)
	if len(steps) != 3 {
		t.Fatalf("got %d steps", len(steps))
	}
	delRequireOK(t, delRunAll(f.repo, w))
}

func TestDeleteDetachedWorktree(t *testing.T) {
	f := delSetup(t)
	path := filepath.Join(f.repo.Root, ".worktrees", "det")
	git(t, f.repo.Root, "worktree", "add", "--detach", path, "main")
	w := Worktree{Path: path, Name: "det", Detached: true}
	steps := DeleteSteps(w)
	if len(steps) != 1 || steps[0].Label != "worktree" {
		t.Fatalf("steps = %+v", steps)
	}
	delRequireOK(t, delRunAll(f.repo, w))
	if delExists(path) {
		t.Error("worktree dir still exists")
	}
}

func TestDeleteBranchCheckedOutElsewhere(t *testing.T) {
	f := delSetup(t)
	first := f.addWorktree(t, "shared", false)
	second := filepath.Join(f.repo.Root, ".worktrees", "shared2")
	git(t, f.repo.Root, "worktree", "add", "--force", second, "shared")
	other := f.addWorktree(t, "other", true)

	errs := delRunAll(f.repo, first)
	if len(errs) != 2 || errs[0] != nil || errs[1] == nil {
		t.Fatalf("errs = %v", errs)
	}
	msg := errs[1].Error()
	if msg == "" || msg != firstLine(msg) {
		t.Errorf("error not a single line: %q", msg)
	}
	delRequireOK(t, delRunAll(f.repo, other))
	if delExists(other.Path) {
		t.Error("following item not deleted")
	}
}

func TestStepLabels(t *testing.T) {
	tests := []struct {
		name string
		w    Worktree
		want []string
	}{
		{"detached", Worktree{Detached: true}, []string{"worktree"}},
		{"branch only", Worktree{Branch: "a"}, []string{"worktree", "branch"}},
		{"branch and upstream", Worktree{Branch: "a", Upstream: "origin/a"}, []string{"worktree", "branch", "origin/a"}},
		{"tracks another branch", Worktree{Branch: "a", Upstream: "origin/main"}, []string{"worktree", "branch"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			for _, s := range DeleteSteps(tt.w) {
				got = append(got, s.Label)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeleteConcurrentItems(t *testing.T) {
	f := delSetup(t)
	var ws []Worktree
	for i := range 12 {
		ws = append(ws, f.addWorktree(t, fmt.Sprintf("feat-%d", i), true))
	}
	git(t, f.repo.Root, "pack-refs", "--all")
	git(t, f.origin, "pack-refs", "--all")

	errs := make([][]error, len(ws))
	var wg sync.WaitGroup
	for i, w := range ws {
		wg.Go(func() { errs[i] = delRunAll(f.repo, w) })
	}
	wg.Wait()

	for i, w := range ws {
		delRequireOK(t, errs[i])
		if delExists(w.Path) || delHasRef(t, f.repo.Root, "refs/heads/"+w.Branch) || delHasRef(t, f.origin, "refs/heads/"+w.Branch) {
			t.Fatalf("%s not fully deleted", w.Name)
		}
	}
	if !delHasRef(t, f.origin, "refs/heads/main") {
		t.Fatal("origin main deleted")
	}
}
