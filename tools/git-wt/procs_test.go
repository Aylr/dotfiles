package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

func TestParseProcs(t *testing.T) {
	ws := []Worktree{{Path: "/r/app", Main: true}, {Path: "/r/app/.worktrees/a"}, {Path: "/r/app/.worktrees/b"}, {Branch: "no-worktree"}}
	out := "p100\nR1\nczsh\nfcwd\nn/r/app/.worktrees/a\n" +
		"p200\nR100\ncnode\nfcwd\nn/r/app/.worktrees/a/web/src\n" +
		"p300\nR1\ncvim\nfcwd\nn/r/app\n" +
		"p400\nR300\ncgit-wt\nfcwd\nn/r/app/.worktrees/b\n" +
		"p410\nR400\ncgit\nfcwd\nn/r/app/.worktrees/b\n" +
		"p411\nR410\ncgit-remote-https\nfcwd\nn/r/app/.worktrees/b\n" +
		"p500\nR1\ncbash\nfcwd\nn/elsewhere\n" +
		"p600\nR1\nc2.1.301\nfcwd\nn/r/app/.worktrees/b\n" +
		"p700\nR1\nczsh\nfcwd\nn/r/app/.worktrees/b\n" +
		"p710\nR700\ncgit\nfcwd\nn/r/app/.worktrees/b\n"
	got := parseProcs(out+"p720\nR710\ncgit-wt\nfcwd\nn/r/app/.worktrees/b\n", newAttributor(ws), 720)
	want := map[string][]Proc{
		"/r/app/.worktrees/a": {{100, "zsh"}, {200, "node"}},
		"/r/app":              {{300, "vim"}},
		"/r/app/.worktrees/b": {{400, "git-wt"}, {410, "git"}, {411, "git-remote-https"}, {600, "claude"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestCollectFindsProcessesInWorktree(t *testing.T) {
	f := newFixtureRepo(t)
	clean := f.Paths["clean"]
	old := procSource
	procSource = func() string { return "p4242\nczsh\nfcwd\nn" + filepath.Join(clean, "src") + "\n" }
	t.Cleanup(func() { procSource = old })

	ws, err := ListAll(f.Repo)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range ws {
		want := 0
		if w.Path == clean {
			want = 1
		}
		if len(w.Procs) != want {
			t.Fatalf("%s procs = %v, want %d", w.Name, w.Procs, want)
		}
	}
}

func TestSquashMergedBranch(t *testing.T) {
	f := delSetup(t)
	shipped := f.addWorktree(t, "shipped", false)
	unshipped := f.addWorktree(t, "unshipped", false)
	for _, w := range []Worktree{shipped, unshipped} {
		if err := os.WriteFile(filepath.Join(w.Path, "feature.txt"), []byte(w.Name+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		git(t, w.Path, "add", ".")
		git(t, w.Path, "commit", "-m", "part 1")
		if err := os.WriteFile(filepath.Join(w.Path, "more.txt"), []byte("more\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		git(t, w.Path, "add", ".")
		git(t, w.Path, "commit", "-m", "part 2")
	}
	git(t, f.repo.Root, "merge", "--squash", "shipped")
	git(t, f.repo.Root, "commit", "-m", "Shipped (#1)")
	git(t, f.repo.Root, "push", "origin", "main")
	git(t, f.repo.Root, "fetch", "origin")

	ws := []Worktree{
		{Path: shipped.Path, Branch: "shipped", Head: git(t, shipped.Path, "rev-parse", "HEAD")},
		{Path: unshipped.Path, Branch: "unshipped", Head: git(t, unshipped.Path, "rev-parse", "HEAD")},
	}
	Collect(f.repo, ws)
	if !ws[0].InBase || ws[0].Lost != 0 || !PreMark(ws[0], "main") {
		t.Fatalf("shipped: squash=%v lost=%d premark=%v", ws[0].InBase, ws[0].Lost, PreMark(ws[0], "main"))
	}
	if ws[1].InBase || ws[1].Lost != 2 {
		t.Fatalf("unshipped: squash=%v lost=%d", ws[1].InBase, ws[1].Lost)
	}

	ws[0].PR = &PR{Number: 1, State: "OPEN", HeadOid: ws[0].Head}
	Collect(f.repo, ws[:1])
	if ws[0].InBase {
		t.Fatal("a branch with a PR should defer to the PR")
	}

	bare := []Worktree{{Branch: "shipped", Head: ws[0].Head}, {Branch: "unshipped", Head: ws[1].Head}}
	Collect(f.repo, bare)
	if !bare[0].InBase || bare[0].Lost != 0 || !PreMark(bare[0], "main") {
		t.Fatalf("shipped without worktree: inBase=%v lost=%d", bare[0].InBase, bare[0].Lost)
	}
	if bare[1].InBase || bare[1].Lost != 2 || PreMark(bare[1], "main") {
		t.Fatalf("unshipped without worktree: inBase=%v lost=%d", bare[1].InBase, bare[1].Lost)
	}
}

func TestLoadProcsRealProcesses(t *testing.T) {
	f := newFixtureRepo(t)
	stranger, child := f.Paths["clean"], f.Paths["dirty"]

	pidFile := filepath.Join(t.TempDir(), "pid")
	detach := exec.Command("sh", "-c", "sleep 30 & echo $! > "+pidFile)
	detach.Dir = stranger
	if err := detach.Run(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(pidFile)
	pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	t.Cleanup(func() { syscall.Kill(pid, syscall.SIGKILL) })

	own := exec.Command("sleep", "30")
	own.Dir = child
	if err := own.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { own.Process.Kill(); own.Wait() })

	ws, err := ListWorktrees(f.Repo)
	if err != nil {
		t.Fatal(err)
	}
	procs := LoadProcs(ws)
	if got := procs[stranger]; len(got) != 1 || got[0].PID != pid || got[0].Command != "sleep" {
		t.Fatalf("stranger procs = %v, want sleep %d", got, pid)
	}
	if got := procs[child]; len(got) != 0 {
		t.Fatalf("git wt's own child counted as in use: %v", got)
	}
}
