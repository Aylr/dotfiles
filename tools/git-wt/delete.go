package main

import (
	"fmt"
	"strings"
)

type Step struct {
	Label string
	run   func(r Repo) error
}

func (s Step) Run(r Repo) error { return s.run(r) }

func DeleteSteps(w Worktree) []Step {
	var steps []Step
	if !w.BranchOnly() {
		steps = append(steps, Step{"worktree", func(r Repo) error { return removeWorktree(r, w) }})
	}
	if w.Branch != "" {
		steps = append(steps, Step{"branch", func(r Repo) error {
			_, err := runGit(r.Root, "branch", "-D", w.Branch)
			return err
		}})
	}
	if remote := RemoteBranch(w); remote != "" {
		steps = append(steps, Step{remote, func(r Repo) error { return deleteRemoteBranch(r, remote) }})
	}
	return steps
}

func removeWorktree(r Repo, w Worktree) error {
	args := []string{"worktree", "remove"}
	if len(w.DirtyFiles)+len(w.Untracked) > 0 {
		args = append(args, "--force")
	}
	_, err := runGit(r.Root, append(args, w.Path)...)
	return err
}

func deleteRemoteBranch(r Repo, upstream string) error {
	if _, err := runGit(r.Root, "show-ref", "--verify", "--quiet", "refs/remotes/"+upstream); err != nil {
		return nil
	}
	remote, branch, _ := strings.Cut(upstream, "/")
	if branch == r.DefaultBranch {
		return fmt.Errorf("refusing to delete %s", upstream)
	}
	_, err := runGit(r.Root, "push", remote, "--delete", branch)
	return err
}
