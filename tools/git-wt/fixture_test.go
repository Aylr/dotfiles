package main

import (
	"os"
	"path/filepath"
	"testing"
)

type fixture struct {
	Root  string
	Repo  Repo
	Paths map[string]string
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newFixtureRepo(t *testing.T) fixture {
	t.Helper()
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	origin := filepath.Join(tmp, "origin.git")
	root := filepath.Join(tmp, "repo")
	for _, d := range []string{origin, root} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	git(t, origin, "init", "--bare", "-b", "main")
	git(t, root, "init", "-b", "main")
	git(t, root, "config", "user.name", "t")
	git(t, root, "config", "user.email", "t@t")
	git(t, root, "remote", "add", "origin", origin)
	writeFile(t, filepath.Join(root, "README.md"), "hello\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "initial")
	git(t, root, "push", "-u", "origin", "main")
	git(t, root, "remote", "set-head", "origin", "-a")

	paths := map[string]string{}
	add := func(name, dir string, args ...string) string {
		p := filepath.Join(root, dir)
		git(t, root, append([]string{"worktree", "add"}, append(args, p)...)...)
		paths[name] = p
		return p
	}
	branch := func(name, dir string) string { return add(name, dir, "-b", name) }
	detach := func(name, dir string) string { return add(name, dir, "--detach") }

	p := branch("clean", ".worktrees/clean")
	git(t, p, "push", "-u", "origin", "clean")

	p = branch("dirty", ".worktrees/dirty")
	git(t, p, "push", "-u", "origin", "dirty")
	writeFile(t, filepath.Join(p, "README.md"), "changed\n")

	p = branch("untracked", ".worktrees/untracked")
	git(t, p, "push", "-u", "origin", "untracked")
	writeFile(t, filepath.Join(p, "new.txt"), "new\n")

	p = branch("unpushed", ".worktrees/unpushed")
	git(t, p, "push", "-u", "origin", "unpushed")
	git(t, p, "commit", "--allow-empty", "-m", "local only")

	detach("detached-reachable", ".worktrees/detached-reachable")

	p = detach("detached-orphaned", ".worktrees/detached-orphaned")
	git(t, p, "commit", "--allow-empty", "-m", "orphan")

	p = branch("upstream-gone", ".worktrees/upstream-gone")
	git(t, p, "commit", "--allow-empty", "-m", "gone work")
	git(t, p, "push", "-u", "origin", "upstream-gone")
	git(t, root, "push", "origin", "--delete", "upstream-gone")
	git(t, root, "fetch", "--prune", "origin")

	add("claude/x", ".claude/worktrees/x", "-b", "claude-x")

	git(t, root, "commit", "--allow-empty", "-m", "advance main")
	git(t, root, "push", "origin", "main")

	r, err := DetectRepo(root)
	if err != nil {
		t.Fatal(err)
	}
	return fixture{Root: root, Repo: r, Paths: paths}
}
