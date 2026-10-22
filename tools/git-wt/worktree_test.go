package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const porcelainNL = `worktree /r/main
HEAD aaaaaaaa
branch refs/heads/main

worktree /r/.worktrees/det
HEAD bbbbbbbb
detached

worktree /r/.worktrees/lock
HEAD cccccccc
branch refs/heads/feat/x
locked because reasons

worktree /r/.worktrees/gone
HEAD dddddddd
branch refs/heads/gone
prunable gitdir file points to non-existent location

worktree /r/bare
bare
`

func TestParsePorcelainNewline(t *testing.T) {
	got := ParsePorcelain(porcelainNL)
	want := []Worktree{
		{Path: "/r/main", Head: "aaaaaaaa", Branch: "main"},
		{Path: "/r/.worktrees/det", Head: "bbbbbbbb", Detached: true},
		{Path: "/r/.worktrees/lock", Head: "cccccccc", Branch: "feat/x", Locked: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}

func TestParsePorcelainZ(t *testing.T) {
	z := strings.NewReplacer("\n\n", "\x00\x00", "\n", "\x00").Replace(porcelainNL)
	if got, want := ParsePorcelain(z), ParsePorcelain(porcelainNL); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}

func TestParsePorcelainEmpty(t *testing.T) {
	if got := ParsePorcelain(""); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestParseStatus(t *testing.T) {
	out := " M a.go\x00?? new.txt\x00?? dir/\x00R  renamed.go\x00old.go\x00!! ignored\x00A  added.go\x00MM both.go\x00"
	dirty, untracked := parseStatus(out)
	if want := []string{"a.go", "renamed.go", "added.go", "both.go"}; !reflect.DeepEqual(dirty, want) {
		t.Errorf("dirty = %v, want %v", dirty, want)
	}
	if want := []string{"new.txt", "dir/"}; !reflect.DeepEqual(untracked, want) {
		t.Errorf("untracked = %v, want %v", untracked, want)
	}
}

func byName(ws []Worktree) map[string]Worktree {
	m := map[string]Worktree{}
	for _, w := range ws {
		m[w.Name] = w
	}
	return m
}

func TestDetectRepoFixture(t *testing.T) {
	f := newFixtureRepo(t)
	r := f.Repo
	if r.Root != f.Root || r.Toplevel != f.Root {
		t.Errorf("Root/Toplevel = %s/%s, want %s", r.Root, r.Toplevel, f.Root)
	}
	if r.Name != "repo" {
		t.Errorf("Name = %q", r.Name)
	}
	if r.DefaultBranch != "main" || r.Base != "origin/main" {
		t.Errorf("DefaultBranch/Base = %s/%s", r.DefaultBranch, r.Base)
	}

	inWt, err := DetectRepo(f.Paths["clean"])
	if err != nil {
		t.Fatal(err)
	}
	if inWt.Root != f.Root || inWt.Toplevel != f.Paths["clean"] {
		t.Errorf("from worktree: Root/Toplevel = %s/%s", inWt.Root, inWt.Toplevel)
	}

	if _, err := DetectRepo(t.TempDir()); err == nil || err.Error() != "not in a git repository" {
		t.Errorf("err = %v", err)
	}
}

func TestDetectRepoNoOrigin(t *testing.T) {
	dir, _ := filepath.EvalSymlinks(t.TempDir())
	git(t, dir, "init", "-b", "master")
	git(t, dir, "commit", "--allow-empty", "-m", "x")
	r, err := DetectRepo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if r.DefaultBranch != "master" || r.Base != "master" {
		t.Errorf("DefaultBranch/Base = %s/%s", r.DefaultBranch, r.Base)
	}
	if err := Fetch(r); err != nil {
		t.Errorf("Fetch without origin: %v", err)
	}
}

func TestListWorktreesFixture(t *testing.T) {
	f := newFixtureRepo(t)
	ws, err := ListWorktrees(f.Repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(ws) != 9 {
		t.Fatalf("got %d worktrees", len(ws))
	}
	if !ws[0].Main || ws[0].Path != f.Root || ws[0].Name != "repo" {
		t.Errorf("first = %+v", ws[0])
	}
	m := byName(ws)
	for name, path := range f.Paths {
		w, ok := m[name]
		if !ok {
			t.Fatalf("missing worktree %q in %v", name, m)
		}
		if w.Path != path || w.Main || w.SizeKB != -1 || w.Head == "" {
			t.Errorf("%s: %+v", name, w)
		}
		if w.Current {
			t.Errorf("%s Current = %v", name, w.Current)
		}
	}
	if !ws[0].Current {
		t.Errorf("main worktree should be current: %+v", ws[0])
	}
	if m["claude/x"].Branch != "claude-x" {
		t.Errorf("claude/x branch = %q", m["claude/x"].Branch)
	}
}

func TestListBranchesFixture(t *testing.T) {
	f := newFixtureRepo(t)
	ws, err := ListWorktrees(f.Repo)
	if err != nil {
		t.Fatal(err)
	}
	bs, err := ListBranches(f.Repo, ws)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, b := range bs {
		names = append(names, b.Name)
		if !b.BranchOnly() || b.Branch != b.Name || b.Head == "" || b.Key() != "branch:"+b.Name || b.Main || b.Current {
			t.Errorf("branch row = %+v", b)
		}
	}
	if want := []string{"local-branch", "merged-branch", "pushed-branch"}; !reflect.DeepEqual(names, want) {
		t.Errorf("branches = %v, want %v", names, want)
	}
	all, err := ListAll(f.Repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != len(ws)+len(bs) || !all[0].Main {
		t.Errorf("ListAll returned %d rows, first %+v", len(all), all[0])
	}
}

func TestCollectFixture(t *testing.T) {
	f := newFixtureRepo(t)
	ws, err := ListAll(f.Repo)
	if err != nil {
		t.Fatal(err)
	}
	m := byName(ws)

	type want struct {
		upstream        string
		dirty, untr     []string
		lost, ahead     int
		behind          int
		detached, orphn bool
		inBase          bool
	}
	cases := map[string]want{
		"clean":              {upstream: "origin/clean", behind: 1, inBase: true},
		"dirty":              {upstream: "origin/dirty", dirty: []string{"README.md"}, behind: 1, inBase: true},
		"untracked":          {upstream: "origin/untracked", untr: []string{"new.txt"}, behind: 1, inBase: true},
		"unpushed":           {upstream: "origin/unpushed", lost: 1, ahead: 1, behind: 1},
		"detached-reachable": {detached: true, behind: 1},
		"detached-orphaned":  {detached: true, orphn: true, lost: 1, ahead: 1, behind: 1},
		"upstream-gone":      {lost: 1, ahead: 1, behind: 1},
		"claude/x":           {behind: 1, inBase: true},
		"merged-branch":      {behind: 1, inBase: true},
		"pushed-branch":      {upstream: "origin/pushed-branch", ahead: 1, behind: 1},
		"local-branch":       {lost: 1, ahead: 1, behind: 1},
	}
	for name, c := range cases {
		w, ok := m[name]
		if !ok {
			t.Errorf("%s missing", name)
			continue
		}
		if w.Upstream != c.upstream {
			t.Errorf("%s Upstream = %q, want %q", name, w.Upstream, c.upstream)
		}
		if !reflect.DeepEqual(w.DirtyFiles, c.dirty) {
			t.Errorf("%s DirtyFiles = %v, want %v", name, w.DirtyFiles, c.dirty)
		}
		if !reflect.DeepEqual(w.Untracked, c.untr) {
			t.Errorf("%s Untracked = %v, want %v", name, w.Untracked, c.untr)
		}
		if w.Lost != c.lost || w.Ahead != c.ahead || w.Behind != c.behind {
			t.Errorf("%s lost/ahead/behind = %d/%d/%d, want %d/%d/%d", name, w.Lost, w.Ahead, w.Behind, c.lost, c.ahead, c.behind)
		}
		if w.Detached != c.detached || w.Orphaned != c.orphn {
			t.Errorf("%s detached/orphaned = %v/%v, want %v/%v", name, w.Detached, w.Orphaned, c.detached, c.orphn)
		}
		if w.InBase != c.inBase {
			t.Errorf("%s InBase = %v, want %v", name, w.InBase, c.inBase)
		}
		if w.LastActivity.IsZero() {
			t.Errorf("%s LastActivity is zero", name)
		}
	}

	main := m["repo"]
	if main.Upstream != "origin/main" || main.Lost != 0 || main.Ahead != 0 || main.Behind != 0 || main.InBase {
		t.Errorf("main = %+v", main)
	}
	if main.Orphaned {
		t.Errorf("main Orphaned")
	}
	if len(main.Untracked) == 0 {
		t.Errorf("main should list worktree dirs as untracked: %+v", main)
	}

	commit := time.Now().Add(-time.Minute)
	if !m["untracked"].LastActivity.After(commit) || !m["dirty"].LastActivity.After(commit) {
		t.Errorf("dirty/untracked LastActivity should reflect file mtimes: %v %v", m["dirty"].LastActivity, m["untracked"].LastActivity)
	}
}

func TestCollectPreservesFields(t *testing.T) {
	f := newFixtureRepo(t)
	ws, _ := ListWorktrees(f.Repo)
	for i := range ws {
		ws[i].Marked = true
		ws[i].SizeKB = 42
		ws[i].Sessions = []Session{{ID: "s"}}
		ws[i].PR = &PR{Number: 7}
	}
	before := append([]Worktree(nil), ws...)
	Collect(f.Repo, ws)
	for i := range ws {
		w, b := ws[i], before[i]
		if !w.Marked || w.SizeKB != 42 || len(w.Sessions) != 1 || w.PR == nil || w.Name != b.Name || w.Main != b.Main || w.Current != b.Current {
			t.Errorf("%s lost preserved fields: %+v", b.Name, w)
		}
	}
}

func TestCollectLostWithPR(t *testing.T) {
	f := newFixtureRepo(t)
	ws, _ := ListAll(f.Repo)
	m := byName(ws)
	var gone *Worktree
	for i := range ws {
		if ws[i].Name == "upstream-gone" {
			gone = &ws[i]
		}
	}
	if m["upstream-gone"].Lost != 1 {
		t.Fatalf("precondition: Lost = %d", m["upstream-gone"].Lost)
	}
	gone.PR = &PR{Number: 1, State: "MERGED", HeadOid: gone.Head}
	Collect(f.Repo, ws)
	if gone.Lost != 0 {
		t.Errorf("Lost with PR at HEAD = %d, want 0", gone.Lost)
	}

	git(t, gone.Path, "commit", "--allow-empty", "-m", "after pr")
	gone.Head = git(t, gone.Path, "rev-parse", "HEAD")
	Collect(f.Repo, ws)
	if gone.Lost != 1 {
		t.Errorf("Lost with one commit past PR = %d, want 1", gone.Lost)
	}

	gone.PR.HeadOid = strings.Repeat("0", 40)
	Collect(f.Repo, ws)
	if gone.Lost != 2 {
		t.Errorf("Lost with unknown PR oid = %d, want 2", gone.Lost)
	}
}

func TestCollectMissingBase(t *testing.T) {
	f := newFixtureRepo(t)
	ws, _ := ListWorktrees(f.Repo)
	r := f.Repo
	r.Base = "origin/nonexistent"
	Collect(r, ws)
	for _, w := range ws {
		if w.Ahead != 0 || w.Behind != 0 {
			t.Errorf("%s ahead/behind = %d/%d", w.Name, w.Ahead, w.Behind)
		}
	}
}

func TestFetchFixture(t *testing.T) {
	f := newFixtureRepo(t)
	if err := Fetch(f.Repo); err != nil {
		t.Fatal(err)
	}
}

func TestDirSizeKB(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f"), make([]byte, 64*1024), 0o644); err != nil {
		t.Fatal(err)
	}
	kb, err := DirSizeKB(dir)
	if err != nil || kb < 64 {
		t.Errorf("kb = %d, err = %v", kb, err)
	}
	if _, err := DirSizeKB(filepath.Join(dir, "missing")); err == nil {
		t.Error("expected error for missing path")
	}
}

func TestLoadDetailFixture(t *testing.T) {
	f := newFixtureRepo(t)
	ws, _ := ListWorktrees(f.Repo)
	m := byName(ws)

	w := m["dirty"]
	d, err := LoadDetail(f.Repo, w)
	if err != nil {
		t.Fatal(err)
	}
	mb, _ := MergeBase(f.Repo, w)
	if d.MergeBase == "" || d.MergeBase != mb {
		t.Errorf("MergeBase = %q, want %q", d.MergeBase, mb)
	}
	if !strings.Contains(d.Stat, "README.md") {
		t.Errorf("Stat should mention the uncommitted change:\n%s", d.Stat)
	}
	if len(d.Commits) != 1 || d.Commits[0].Subject != "initial" || d.Commits[0].Hash == "" || d.Commits[0].When == "" {
		t.Errorf("Commits = %+v", d.Commits)
	}

	d, err = LoadDetail(f.Repo, m["unpushed"])
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Commits) != 2 || d.Commits[0].Subject != "local only" {
		t.Errorf("Commits = %+v", d.Commits)
	}
}

func TestLoadDetailBranchOnly(t *testing.T) {
	f := newFixtureRepo(t)
	all, _ := ListAll(f.Repo)
	m := byName(all)
	b := m["pushed-branch"]
	if !b.BranchOnly() {
		t.Fatalf("precondition: %+v", b)
	}
	tmp := filepath.Join(f.Root, ".worktrees", "tmp")
	git(t, f.Root, "worktree", "add", "-q", "--detach", tmp, "pushed-branch")
	writeFile(t, filepath.Join(tmp, "feature.txt"), "x\n")
	git(t, tmp, "add", "feature.txt")
	git(t, tmp, "commit", "-q", "-m", "file on branch")
	git(t, f.Root, "update-ref", "refs/heads/pushed-branch", git(t, tmp, "rev-parse", "HEAD"))
	git(t, f.Root, "worktree", "remove", tmp)
	all, _ = ListAll(f.Repo)

	b = byName(all)["pushed-branch"]

	d, err := LoadDetail(f.Repo, b)
	if err != nil {
		t.Fatal(err)
	}
	mb, _ := MergeBase(f.Repo, b)
	if d.MergeBase != mb || mb == "" {
		t.Errorf("MergeBase = %q, want %q", d.MergeBase, mb)
	}
	if !strings.Contains(d.Stat, "feature.txt") {
		t.Errorf("Stat should show the branch's committed change:\n%s", d.Stat)
	}
	if len(d.Commits) == 0 || d.Commits[0].Subject != "file on branch" {
		t.Errorf("Commits = %+v", d.Commits)
	}
}
