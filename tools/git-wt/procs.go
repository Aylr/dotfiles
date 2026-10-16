package main

import (
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var procSource = func() string {
	out, _ := run("", "lsof", "-w", "-d", "cwd", "-FpRcn")
	return out
}

// LoadProcs returns the processes, other than git wt's own process tree, whose working directory is inside each worktree.
func LoadProcs(ws []Worktree) map[string][]Proc {
	return parseProcs(procSource(), newAttributor(ws), os.Getpid())
}

type procEntry struct {
	Proc
	ppid int
	cwd  string
}

func parseProcs(out string, attr attributor, self int) map[string][]Proc {
	var entries []procEntry
	parent := map[int]int{}
	var cur procEntry
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		switch line[0] {
		case 'p':
			cur = procEntry{}
			cur.PID, _ = strconv.Atoi(line[1:])
		case 'R':
			cur.ppid, _ = strconv.Atoi(line[1:])
			parent[cur.PID] = cur.ppid
		case 'c':
			cur.Command = line[1:]
		case 'n':
			cur.cwd = line[1:]
			entries = append(entries, cur)
		}
	}
	procs := map[string][]Proc{}
	for _, e := range entries {
		if e.PID == 0 || descendsFrom(e.PID, self, parent) || descendsFrom(self, e.PID, parent) {
			continue
		}
		e.Command = displayName(e.Command)
		if p, ok := attr.worktree(e.cwd); ok {
			procs[p] = append(procs[p], e.Proc)
		}
	}
	return procs
}

var versionName = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// Claude Code's executable is named after its version, e.g. 2.1.301.
func displayName(command string) string {
	if versionName.MatchString(command) {
		return "claude"
	}
	return command
}

func descendsFrom(pid, ancestor int, parent map[int]int) bool {
	for range 64 {
		if pid == ancestor {
			return true
		}
		next, ok := parent[pid]
		if !ok || next == pid || next <= 1 {
			return false
		}
		pid = next
	}
	return false
}

func procNames(ps []Proc) string {
	var names []string
	for _, p := range ps {
		if !slices.Contains(names, p.Command) {
			names = append(names, p.Command)
		}
	}
	return strings.Join(names, ", ")
}
