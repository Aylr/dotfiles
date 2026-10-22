package main

import "time"

type Repo struct {
	Root          string // main worktree path
	Name          string // Root relative to ~/repos, else its basename
	Toplevel      string // worktree containing the process cwd
	DefaultBranch string
	Base          string // origin/<default> when present, else <default>
}

type PR struct {
	Number         int
	Title, URL     string
	State          string // OPEN | CLOSED | MERGED
	Draft          bool
	HeadRef        string
	HeadOid        string
	ReviewDecision string
}

type Session struct {
	ID, Title string
	Last      time.Time
	Live      bool
}

type Worktree struct {
	Path, Name, Branch, Head        string
	Main, Current, Detached, Locked bool
	Upstream                        string // "origin/foo"; empty when none or gone
	DirtyFiles, Untracked           []string
	Lost                            int // commits that exist only here
	Ahead, Behind                   int
	Orphaned                        bool // detached HEAD unreachable from any branch
	InBase                          bool // no PR, but every change here already exists on base
	LastActivity                    time.Time
	PR                              *PR
	Sessions                        []Session
	Procs                           []Proc
	SizeKB                          int64 // -1 while pending
	Marked                          bool
}

func (w Worktree) BranchOnly() bool { return w.Path == "" }

func (w Worktree) Key() string {
	if w.BranchOnly() {
		return "branch:" + w.Branch
	}
	return w.Path
}

func (w Worktree) Dir(root string) string {
	if w.BranchOnly() {
		return root
	}
	return w.Path
}

type Proc struct {
	PID     int
	Command string
}

type Commit struct {
	Hash, When, Subject string
}

type Detail struct {
	MergeBase string
	Stat      string
	Commits   []Commit
}
