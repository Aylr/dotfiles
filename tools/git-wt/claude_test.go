package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var claudeWorktrees = []Worktree{
	{Path: "/r/app", Main: true},
	{Path: "/r/app/.worktrees/a"},
	{Path: "/r/app/.worktrees/b"},
	{Path: "/r/app/.claude/worktrees/c"},
	{Path: "/elsewhere/wt-d"},
	{Path: "/r/app-other"},
}

func stubPids(t *testing.T, alive ...int) {
	t.Helper()
	old := pidAlive
	t.Cleanup(func() { pidAlive = old })
	pidAlive = func(pid int) bool {
		for _, p := range alive {
			if p == pid {
				return true
			}
		}
		return false
	}
}

func byID(ss []Session) map[string]Session {
	m := map[string]Session{}
	for _, s := range ss {
		m[s.ID] = s
	}
	return m
}

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t.Fatal(err)
	}
	return ts
}

func TestLoadSessionsAttribution(t *testing.T) {
	stubPids(t)
	got := LoadSessions("testdata/claude", claudeWorktrees)

	multi := map[string]string{
		"/r/app":              "2026-03-01T10:00:00Z",
		"/r/app/.worktrees/a": "2026-03-01T11:30:00Z",
		"/r/app/.worktrees/b": "2026-03-01T12:00:00Z",
	}
	for path, want := range multi {
		s, ok := byID(got[path])["multi-0001"]
		if !ok {
			t.Fatalf("%s missing multi-0001: %+v", path, got[path])
		}
		if !s.Last.Equal(mustTime(t, want)) {
			t.Errorf("%s Last = %v, want %s", path, s.Last, want)
		}
		if s.Title != "Final \"quoted\" café \\ end" {
			t.Errorf("%s title = %q", path, s.Title)
		}
	}
	if _, ok := byID(got["/r/app/.claude/worktrees/c"])["multi-0001"]; ok {
		t.Error("subagent transcript attributed to c")
	}
}

func TestLoadSessionsTitles(t *testing.T) {
	stubPids(t)
	got := LoadSessions("testdata/claude", claudeWorktrees)

	c := byID(got["/r/app/.claude/worktrees/c"])
	if s := c["aionly-0001"]; s.Title != "Only AI" {
		t.Errorf("ai title = %q", s.Title)
	}
	a := byID(got["/r/app/.worktrees/a"])
	if s := a["abcdef1234567890"]; s.Title != "abcdef12" {
		t.Errorf("fallback title = %q", s.Title)
	} else if !s.Last.Equal(mustTime(t, "2026-03-03T09:00:00Z")) {
		t.Errorf("fallback Last = %v", s.Last)
	}
	if s := a["escaped-0001"]; s.Title != "tab here 🚀" {
		t.Errorf("escaped title = %q", s.Title)
	} else if !s.Last.Equal(mustTime(t, "2026-03-04T09:00:00Z")) {
		t.Errorf("escaped cwd Last = %v", s.Last)
	}
	if _, ok := byID(got["/r/app/.worktrees/b"])["escaped-0001"]; ok {
		t.Error("cwd inside string value was attributed")
	}
}

func TestLoadSessionsFileSelection(t *testing.T) {
	stubPids(t)
	got := LoadSessions("testdata/claude", claudeWorktrees)

	if _, ok := byID(got["/r/app/.worktrees/b"])["perwt-0001"]; !ok {
		t.Error("per-worktree project dir missed")
	}
	if _, ok := byID(got["/elsewhere/wt-d"])["outside-0001"]; !ok {
		t.Error("worktree outside main dir missed")
	}
	if _, ok := byID(got["/r/app-other"])["other-0001"]; !ok {
		t.Error("worktree /r/app-other should own its own session")
	}
	for path, ss := range got {
		if path == "/r/app-other" {
			continue
		}
		if _, ok := byID(ss)["other-0001"]; ok {
			t.Errorf("sibling repo session attributed to %s", path)
		}
	}
	rootOnly := LoadSessions("testdata/claude", claudeWorktrees[:5])
	for path, ss := range rootOnly {
		if _, ok := byID(ss)["other-0001"]; ok {
			t.Errorf("sibling repo session attributed to %s", path)
		}
	}
}

func TestLoadSessionsSortedNewestFirst(t *testing.T) {
	stubPids(t)
	got := LoadSessions("testdata/claude", claudeWorktrees)
	for path, ss := range got {
		for i := 1; i < len(ss); i++ {
			if ss[i].Last.After(ss[i-1].Last) {
				t.Errorf("%s not sorted: %+v", path, ss)
			}
		}
	}
	b := got["/r/app/.worktrees/b"]
	if len(b) < 2 || b[0].ID != "perwt-0001" {
		t.Errorf("b order = %+v", b)
	}
}

func TestLoadSessionsLive(t *testing.T) {
	stubPids(t, 111, 333)
	got := LoadSessions("testdata/claude", claudeWorktrees)

	if s := byID(got["/r/app/.worktrees/b"])["live-0001"]; !s.Live {
		t.Errorf("live-0001 not live in b: %+v", got["/r/app/.worktrees/b"])
	}
	if s, ok := byID(got["/r/app"])["live-0001"]; !ok || s.Live {
		t.Errorf("live-0001 in root should exist and be not live: %+v", s)
	}
	if s := byID(got["/r/app/.worktrees/a"])["ghost-session-id"]; !s.Live || s.Title != "ghost-se" {
		t.Errorf("ghost fallback = %+v", s)
	}
	if s := byID(got["/r/app/.worktrees/b"])["multi-0001"]; s.Live {
		t.Error("dead pid marked live")
	}
}

func TestLoadSessionsMissingDirs(t *testing.T) {
	stubPids(t)
	if got := LoadSessions(filepath.Join(t.TempDir(), "nope"), claudeWorktrees); len(got) != 0 {
		t.Errorf("got %v", got)
	}
}

func TestTranscriptHugeLine(t *testing.T) {
	stubPids(t)
	dir := t.TempDir()
	proj := filepath.Join(dir, "projects", "-r-app")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	blob := strings.Repeat(`x\"cwd\":\"y `, 5<<20/12)
	content := fmt.Sprintf(
		`{"sessionId":"big-0001","cwd":"/r/app/.worktrees/a","timestamp":"2026-04-01T00:00:00.000Z"}`+"\n"+
			`{"type":"tool","output":"%s","cwd":"/r/app/.worktrees/b","timestamp":"2026-04-02T00:00:00.000Z"}`+"\n"+
			`{"type":"custom-title","customTitle":"after blob","sessionId":"big-0001"}`+"\n"+
			`{"sessionId":"big-0001","cwd":"/r/app","timestamp":"2026-04-03T00:00:00.000Z"}`,
		blob)
	if len(content) < 5<<20 {
		t.Fatalf("blob too small: %d", len(content))
	}
	if err := os.WriteFile(filepath.Join(proj, "big-0001.jsonl"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got := LoadSessions(dir, claudeWorktrees)
	for path, want := range map[string]string{
		"/r/app/.worktrees/a": "2026-04-01T00:00:00Z",
		"/r/app/.worktrees/b": "2026-04-02T00:00:00Z",
		"/r/app":              "2026-04-03T00:00:00Z",
	} {
		s, ok := byID(got[path])["big-0001"]
		if !ok {
			t.Fatalf("%s missing big-0001", path)
		}
		if !s.Last.Equal(mustTime(t, want)) || s.Title != "after blob" {
			t.Errorf("%s = %+v", path, s)
		}
	}
}
