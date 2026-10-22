package main

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const collectWorkers = 8

func DetectRepo(cwd string) (Repo, error) {
	top, err := runGit(cwd, "rev-parse", "--show-toplevel")
	if err != nil || top == "" {
		return Repo{}, errors.New("not in a git repository")
	}
	top = realPath(top)
	list, err := runGit(top, "worktree", "list", "--porcelain")
	if err != nil {
		return Repo{}, err
	}
	first, _, _ := strings.Cut(list, "\n")
	root, ok := strings.CutPrefix(first, "worktree ")
	if !ok {
		return Repo{}, errors.New("cannot find main worktree")
	}
	root = realPath(root)

	def, err := runGit(root, "symbolic-ref", "-q", "--short", "refs/remotes/origin/HEAD")
	def = strings.TrimPrefix(def, "origin/")
	if err != nil || def == "" {
		def = "master"
		if _, err := runGit(root, "show-ref", "-q", "--verify", "refs/heads/main"); err == nil {
			def = "main"
		}
	}
	base := def
	if _, err := runGit(root, "rev-parse", "-q", "--verify", "origin/"+def); err == nil {
		base = "origin/" + def
	}
	return Repo{Root: root, Name: repoName(root), Toplevel: top, DefaultBranch: def, Base: base}, nil
}

func repoName(root string) string {
	if home, err := os.UserHomeDir(); err == nil {
		rel, err := filepath.Rel(realPath(filepath.Join(home, "repos")), root)
		if err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
			return rel
		}
	}
	return filepath.Base(root)
}

func realPath(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}

func ParsePorcelain(out string) []Worktree {
	sep := "\n"
	if strings.Contains(out, "\x00") {
		sep = "\x00"
	}
	var ws []Worktree
	var cur Worktree
	skip := false
	flush := func() {
		if cur.Path != "" && !skip {
			ws = append(ws, cur)
		}
		cur, skip = Worktree{}, false
	}
	for _, field := range strings.Split(out, sep) {
		if field == "" {
			flush()
			continue
		}
		key, val, _ := strings.Cut(field, " ")
		switch key {
		case "worktree":
			cur.Path = val
		case "HEAD":
			cur.Head = val
		case "branch":
			cur.Branch = strings.TrimPrefix(val, "refs/heads/")
		case "detached":
			cur.Detached = true
		case "locked":
			cur.Locked = true
		case "prunable", "bare":
			skip = true
		}
	}
	flush()
	return ws
}

func ListWorktrees(r Repo) ([]Worktree, error) {
	out, err := runGit(r.Root, "worktree", "list", "--porcelain", "-z")
	if err != nil {
		return nil, err
	}
	ws := ParsePorcelain(out)
	for i := range ws {
		w := &ws[i]
		w.Path = realPath(w.Path)
		w.Main = i == 0
		w.Current = w.Path == r.Toplevel
		w.SizeKB = -1
		w.Name = worktreeName(r.Root, *w)
	}
	return ws, nil
}

func ListBranches(r Repo, ws []Worktree) ([]Worktree, error) {
	out, err := runGit(r.Root, "for-each-ref", "--format=%(refname)%00%(objectname)", "refs/heads")
	if err != nil {
		return nil, err
	}
	checkedOut := map[string]bool{}
	for _, w := range ws {
		checkedOut[w.Branch] = true
	}
	var bs []Worktree
	for _, line := range strings.Split(out, "\n") {
		ref, head, ok := strings.Cut(line, "\x00")
		name := strings.TrimPrefix(ref, "refs/heads/")
		if !ok || checkedOut[name] {
			continue
		}
		bs = append(bs, Worktree{Name: name, Branch: name, Head: head})
	}
	return bs, nil
}

func ListAll(r Repo) ([]Worktree, error) {
	ws, err := ListWorktrees(r)
	if err != nil {
		return nil, err
	}
	bs, err := ListBranches(r, ws)
	if err != nil {
		return nil, err
	}
	ws = append(ws, bs...)
	Collect(r, ws)
	return ws, nil
}

func worktreeName(root string, w Worktree) string {
	if w.Main {
		return filepath.Base(root)
	}
	rel, err := filepath.Rel(root, w.Path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, "../") {
		return filepath.Base(w.Path)
	}
	if after, ok := strings.CutPrefix(rel, ".worktrees/"); ok {
		return after
	}
	if after, ok := strings.CutPrefix(rel, ".claude/worktrees/"); ok {
		return "claude/" + after
	}
	return rel
}

func Collect(r Repo, ws []Worktree) {
	_, err := runGit(r.Root, "rev-parse", "-q", "--verify", r.Base)
	baseOK := err == nil
	procs := LoadProcs(ws)
	for i := range ws {
		ws[i].Procs = procs[ws[i].Path]
	}
	var wg sync.WaitGroup
	sem := make(chan struct{}, collectWorkers)
	for i := range ws {
		wg.Add(1)
		go func(w *Worktree) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			collectOne(r, w, baseOK)
		}(&ws[i])
	}
	wg.Wait()
}

func collectOne(r Repo, w *Worktree, baseOK bool) {
	dir, head := w.Dir(r.Root), w.Head
	w.DirtyFiles, w.Untracked, w.Upstream = nil, nil, ""
	w.Ahead, w.Behind, w.Lost, w.Orphaned, w.InBase = 0, 0, 0, false, false

	if !w.BranchOnly() {
		if out, err := run(dir, "git", "status", "--porcelain=v1", "-z"); err == nil {
			w.DirtyFiles, w.Untracked = parseStatus(out)
		}
	}

	if !w.Detached {
		if up, err := runGit(dir, "rev-parse", "--abbrev-ref", "--symbolic-full-name", w.Branch+"@{u}"); err == nil && up != "" {
			if _, err := runGit(dir, "show-ref", "--verify", "-q", "refs/remotes/"+up); err == nil {
				w.Upstream = up
			}
		}
	}

	if baseOK {
		if out, err := runGit(dir, "rev-list", "--left-right", "--count", r.Base+"..."+head); err == nil {
			left, right, _ := strings.Cut(out, "\t")
			w.Behind, _ = strconv.Atoi(left)
			w.Ahead, _ = strconv.Atoi(right)
		}
	}

	w.LastActivity = lastActivity(dir, head, w.DirtyFiles, w.Untracked)

	if w.Detached {
		if out, err := runGit(dir, "for-each-ref", "--contains", head, "--count=1", "refs/heads", "refs/remotes"); err == nil {
			w.Orphaned = out == ""
		}
	}

	w.Lost = lostCommits(dir, *w)
	if w.PR == nil && !w.Detached && baseOK && w.Branch != r.DefaultBranch {
		switch {
		case isAncestor(dir, head, r.Base):
			w.InBase = true
		case w.Lost > 0 && squashMerged(dir, r.Base, head):
			w.InBase, w.Lost = true, 0
		}
	}
}

func isAncestor(dir, commit, of string) bool {
	_, err := runGit(dir, "merge-base", "--is-ancestor", commit, of)
	return err == nil
}

func squashMerged(p, base, head string) bool {
	mb, err := runGit(p, "merge-base", base, head)
	if err != nil {
		return false
	}
	if _, err := runGit(p, "diff", "--quiet", mb, head); err == nil {
		return false
	}
	squash, err := runGit(p, "commit-tree", head+"^{tree}", "-p", mb, "-m", "_")
	if err != nil {
		return false
	}
	out, err := runGit(p, "cherry", base, squash)
	return err == nil && strings.HasPrefix(out, "-")
}

func lostCommits(p string, w Worktree) int {
	if w.PR != nil && w.PR.HeadOid != "" {
		if _, err := runGit(p, "cat-file", "-e", w.PR.HeadOid+"^{commit}"); err == nil {
			return countRevs(p, w.PR.HeadOid+".."+w.Head)
		}
	}
	if w.Detached {
		return countRevs(p, w.Head, "--not", "--branches", "--remotes")
	}
	return countRevs(p, w.Head, "--not", "--exclude="+w.Branch, "--branches", "--remotes")
}

func countRevs(p string, args ...string) int {
	out, err := runGit(p, append([]string{"rev-list", "--count"}, args...)...)
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(out)
	return n
}

func lastActivity(p, head string, dirty, untracked []string) time.Time {
	var latest time.Time
	if out, err := runGit(p, "log", "-1", "--format=%ct", head); err == nil {
		if sec, err := strconv.ParseInt(out, 10, 64); err == nil {
			latest = time.Unix(sec, 0)
		}
	}
	for _, f := range append(append([]string{}, dirty...), untracked...) {
		if fi, err := os.Lstat(filepath.Join(p, f)); err == nil && fi.ModTime().After(latest) {
			latest = fi.ModTime()
		}
	}
	return latest
}

func parseStatus(out string) (dirty, untracked []string) {
	fields := strings.Split(out, "\x00")
	for i := 0; i < len(fields); i++ {
		e := fields[i]
		if len(e) < 4 {
			continue
		}
		code, path := e[:2], e[3:]
		switch {
		case code == "??":
			untracked = append(untracked, path)
		case code == "!!":
		default:
			dirty = append(dirty, path)
			if strings.ContainsAny(code, "RC") {
				i++
			}
		}
	}
	return dirty, untracked
}

func Fetch(r Repo) error {
	remotes, err := runGit(r.Root, "remote")
	if err != nil {
		return err
	}
	if !strings.Contains("\n"+remotes+"\n", "\norigin\n") {
		return nil
	}
	_, err = runGit(r.Root, "fetch", "--prune", "--quiet", "origin")
	return err
}

func DirSizeKB(path string) (int64, error) {
	out, err := run("", "du", "-sk", path)
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(out)
	if len(fields) == 0 {
		return 0, errors.New("empty du output")
	}
	return strconv.ParseInt(fields[0], 10, 64)
}

func MergeBase(r Repo, w Worktree) (string, error) {
	return runGit(w.Dir(r.Root), "merge-base", r.Base, w.Head)
}

func LoadDetail(r Repo, w Worktree) (Detail, error) {
	mb, err := MergeBase(r, w)
	if err != nil {
		return Detail{}, err
	}
	d := Detail{MergeBase: mb}
	dir := w.Dir(r.Root)
	stat := []string{"diff", "--stat=80", mb}
	if w.BranchOnly() {
		stat = append(stat, w.Head)
	}
	if d.Stat, err = runGit(dir, stat...); err != nil {
		return d, err
	}
	out, err := runGit(dir, "log", "-5", "--format=%h%x09%cr%x09%s", w.Head)

	if err != nil {
		return d, err
	}
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) == 3 {
			d.Commits = append(d.Commits, Commit{Hash: parts[0], When: parts[1], Subject: oneLine(parts[2])})
		}
	}
	return d, nil
}
