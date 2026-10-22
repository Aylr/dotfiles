package main

import (
	"fmt"
	"strings"
)

func BlockedReasons(w Worktree, defaultBranch string) []string {
	var reasons []string
	if w.Main {
		reasons = append(reasons, "main worktree")
	}
	if w.Current {
		reasons = append(reasons, "you are in this worktree")
	}
	if w.Branch != "" && w.Branch == defaultBranch {
		reasons = append(reasons, "default branch")
	}
	if w.PR != nil && w.PR.State == "OPEN" {
		if w.PR.Draft {
			reasons = append(reasons, fmt.Sprintf("PR #%d is a draft", w.PR.Number))
		} else {
			reasons = append(reasons, fmt.Sprintf("PR #%d is open", w.PR.Number))
		}
	}
	for _, s := range w.Sessions {
		if s.Live {
			reasons = append(reasons, "live Claude session")
			break
		}
	}
	if len(w.Procs) > 0 {
		reasons = append(reasons, "in use by "+procNames(w.Procs))
	}
	if w.Locked {
		reasons = append(reasons, "locked")
	}
	return reasons
}

func PreMark(w Worktree, defaultBranch string) bool {
	if len(BlockedReasons(w, defaultBranch)) > 0 {
		return false
	}
	if len(w.DirtyFiles) > 0 || len(w.Untracked) > 0 || w.Lost > 0 {
		return false
	}
	if w.InBase || w.PR != nil && (w.PR.State == "MERGED" || w.PR.State == "CLOSED") {

		return true
	}
	return w.Detached && !w.Orphaned
}

func Losses(w Worktree) []string {
	var losses []string
	if n := len(w.DirtyFiles); n > 0 {
		losses = append(losses, count(n, "dirty file", "dirty files"))
	}
	if n := len(w.Untracked); n > 0 {
		losses = append(losses, count(n, "untracked file", "untracked files"))
	}
	if w.Lost > 0 {
		if w.Lost == 1 {
			losses = append(losses, "1 commit exists only here")
		} else {
			losses = append(losses, fmt.Sprintf("%d commits exist only here", w.Lost))
		}
	}
	if w.Orphaned {
		losses = append(losses, "orphaned detached HEAD")
	}
	if remote := RemoteBranch(w); remote != "" {
		losses = append(losses, "remote "+remote)
	}
	return losses
}

func RemoteBranch(w Worktree) string {
	if _, name, ok := strings.Cut(w.Upstream, "/"); ok && name == w.Branch {
		return w.Upstream
	}
	return ""
}

func count(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}
