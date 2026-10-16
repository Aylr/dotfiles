package main

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

const testRoot = "/repo"

func testRepo() Repo {
	return Repo{Root: testRoot, Name: "acme/backend", Toplevel: testRoot, DefaultBranch: "main", Base: "origin/main"}
}

func testWorktrees() []Worktree {
	now := time.Now()
	return []Worktree{
		{Path: testRoot, Name: "backend", Branch: "main", Main: true, Current: true, Upstream: "origin/main", LastActivity: now, SizeKB: -1},
		{Path: testRoot + "/.worktrees/merged-feature", Name: "merged-feature", Branch: "merged-feature", Upstream: "origin/merged-feature", LastActivity: now.Add(-61 * 24 * time.Hour), SizeKB: -1},
		{Path: testRoot + "/.worktrees/open-feature", Name: "open-feature", Branch: "open-feature", Upstream: "origin/open-feature", Lost: 1, LastActivity: now.Add(-21 * 24 * time.Hour), SizeKB: -1},
		{Path: testRoot + "/.worktrees/scratch", Name: "scratch", Branch: "scratch", DirtyFiles: []string{"a.go"}, LastActivity: now.Add(-5 * time.Hour), SizeKB: -1},
		{Path: testRoot + "/.worktrees/validate", Name: "validate", Head: "abcdef1234567", Detached: true, LastActivity: now.Add(-3 * 24 * time.Hour), SizeKB: -1},
	}
}

func testPRs() []PR {
	return []PR{
		{Number: 601, Title: "Merged feature", URL: "https://github.com/acme/backend/pull/601", State: "MERGED", HeadRef: "merged-feature"},
		{Number: 618, Title: "Open feature", URL: "https://github.com/acme/backend/pull/618", State: "OPEN", HeadRef: "open-feature"},
	}
}

type stepLog struct {
	ran     []string
	deleted map[string]bool
	procs   map[string][]Proc
}

func stubSources(log *stepLog) sources {
	return sources{
		list: func(Repo) ([]Worktree, error) {
			var ws []Worktree
			for _, w := range testWorktrees() {
				if !log.deleted[w.Name] {
					ws = append(ws, w)
				}
			}
			return ws, nil
		},
		fetch:     func(Repo) error { return nil },
		collect:   func(Repo, []Worktree) {},
		prs:       func(string, []string) ([]PR, error) { return testPRs(), nil },
		sessions:  func([]Worktree) (map[string][]Session, error) { return nil, nil },
		size:      func(string) (int64, error) { return 1024, nil },
		detail:    func(Repo, Worktree) (Detail, error) { return Detail{MergeBase: "base"}, nil },
		mergeBase: func(Repo, Worktree) (string, error) { return "base", nil },
		procs:     func(ws []Worktree) map[string][]Proc { return log.procs },
		steps: func(w Worktree) []Step {
			var steps []Step
			for _, label := range []string{"worktree", "branch"} {
				steps = append(steps, Step{Label: label, run: func(Repo) error {
					log.ran = append(log.ran, w.Name+":"+label)
					if w.Name == "scratch" && label == "worktree" {
						return errors.New("worktree is locked")
					}
					log.deleted[w.Name] = true
					return nil
				}})
			}
			return steps
		},
	}
}

func newTestModel(t *testing.T) (model, *stepLog) {
	t.Helper()
	log := &stepLog{deleted: map[string]bool{}}
	m := newModel(testRepo(), testWorktrees(), stubSources(log))
	return send(m, tea.WindowSizeMsg{Width: 120, Height: 40}), log
}

func send(m model, msgs ...tea.Msg) model {
	for _, msg := range msgs {
		next, _ := m.Update(msg)
		m = next.(model)
	}
	return m
}

func press(keys ...string) []tea.Msg {
	var msgs []tea.Msg
	for _, k := range keys {
		switch k {
		case "space":
			msgs = append(msgs, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
		case "esc":
			msgs = append(msgs, tea.KeyPressMsg{Code: tea.KeyEscape})
		case "enter":
			msgs = append(msgs, tea.KeyPressMsg{Code: tea.KeyEnter})
		case "up":
			msgs = append(msgs, tea.KeyPressMsg{Code: tea.KeyUp})
		case "down":
			msgs = append(msgs, tea.KeyPressMsg{Code: tea.KeyDown})
		default:
			r := []rune(k)[0]
			msgs = append(msgs, tea.KeyPressMsg{Code: r, Text: k})
		}
	}
	return msgs
}

func runCmd(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	select {
	case msg := <-ch:
		if batch, ok := msg.(tea.BatchMsg); ok {
			var out []tea.Msg
			for _, c := range batch {
				out = append(out, runCmd(c)...)
			}
			return out
		}
		return []tea.Msg{msg}
	case <-time.After(50 * time.Millisecond):
		return nil
	}
}

func drive(m model, msg tea.Msg, keep func(tea.Msg) bool) model {
	queue := []tea.Msg{msg}
	for len(queue) > 0 {
		next, cmd := m.Update(queue[0])
		m = next.(model)
		queue = queue[1:]
		for _, out := range runCmd(cmd) {
			if keep(out) {
				queue = append(queue, out)
			}
		}
	}
	return m
}

func rowNamed(m model, name string) Worktree {
	for _, w := range m.all {
		if w.Name == name {
			return w
		}
	}
	panic("no worktree " + name)
}

func cursorName(m model) string {
	if w := m.current(); w != nil {
		return w.Name
	}
	return ""
}

func loadAllAsync(m model) model {
	m = send(m, fetchDoneMsg{gen: m.gen})
	m = send(m, collectedMsg{gen: m.gen, ws: m.all, afterFetch: true})
	m = send(m, prsLoadedMsg{gen: m.gen, prs: testPRs()})
	m = send(m, collectedMsg{gen: m.gen, ws: m.all, afterFetch: true, withPRs: true})
	return send(m, sessionsLoadedMsg{gen: m.gen})
}

func TestSortStalestFirstMainLast(t *testing.T) {
	m, _ := newTestModel(t)
	var names []string
	for _, i := range m.visible {
		names = append(names, m.all[i].Name)
	}
	want := []string{"merged-feature", "open-feature", "validate", "scratch", "backend"}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("order = %v, want %v", names, want)
		}
	}
}

func TestCursorClampsAtBothEnds(t *testing.T) {
	m, _ := newTestModel(t)
	m = send(m, press("k", "k")...)
	if m.cursor != 0 {
		t.Fatalf("cursor = %d after moving up from top", m.cursor)
	}
	m = send(m, press("j", "j", "j", "j", "j", "j", "j")...)
	if m.cursor != len(m.visible)-1 {
		t.Fatalf("cursor = %d, want %d", m.cursor, len(m.visible)-1)
	}
	m = send(m, press("g")...)
	if m.cursor != 0 {
		t.Fatalf("g: cursor = %d", m.cursor)
	}
	m = send(m, press("G")...)
	if cursorName(m) != "backend" {
		t.Fatalf("G: cursor on %q", cursorName(m))
	}
}

func TestFilterNarrowsAndEscClears(t *testing.T) {
	m, _ := newTestModel(t)
	m = send(m, press("/", "f", "e", "a", "t")...)
	if !m.filtering || len(m.visible) != 2 {
		t.Fatalf("filtering=%v visible=%d, want 2", m.filtering, len(m.visible))
	}
	m = send(m, press("enter")...)
	if m.filtering || len(m.visible) != 2 {
		t.Fatalf("enter should keep the filter: filtering=%v visible=%d", m.filtering, len(m.visible))
	}
	m = send(m, press("j")...)
	if cursorName(m) != "open-feature" {
		t.Fatalf("cursor on %q", cursorName(m))
	}
	m = send(m, press("esc")...)
	if m.filter.Value() != "" || len(m.visible) != len(m.all) {
		t.Fatalf("esc should clear the filter: %q %d", m.filter.Value(), len(m.visible))
	}
	if cursorName(m) != "open-feature" {
		t.Fatalf("cursor should stay on open-feature, on %q", cursorName(m))
	}
}

func TestFilterMatchesPRAndSessionTitles(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadAllAsync(m)
	m = send(m, sessionsLoadedMsg{gen: m.gen, sessions: map[string][]Session{
		testRoot + "/.worktrees/validate": {{ID: "abcdef123456", Title: "Refactor Billing"}},
	}})
	m = send(m, press("/", "b", "i", "l", "l")...)
	if len(m.visible) != 1 || cursorName(m) != "validate" {
		t.Fatalf("session title filter: visible=%d cursor=%q", len(m.visible), cursorName(m))
	}
	m = send(m, press("esc", "/", "O", "P", "E", "N", " ", "F")...)
	if len(m.visible) != 1 || cursorName(m) != "open-feature" {
		t.Fatalf("PR title filter: visible=%d cursor=%q", len(m.visible), cursorName(m))
	}
}

func TestSpaceOnBlockedRowSetsStatus(t *testing.T) {
	m, _ := newTestModel(t)
	m = send(m, press("G", "space")...)
	if rowNamed(m, "backend").Marked {
		t.Fatal("main worktree was marked")
	}
	if m.status != "✕ main worktree" {
		t.Fatalf("status = %q", m.status)
	}
	m = send(m, press("k")...)
	if m.status != "" {
		t.Fatalf("status should clear on next key, got %q", m.status)
	}
}

func TestAsyncLoadsFillFieldsAndPremarkOnce(t *testing.T) {
	m, _ := newTestModel(t)
	m = send(m, press("j", "j", "space")...)
	if !rowNamed(m, "validate").Marked {
		t.Fatal("toggle before pre-mark did not mark validate")
	}
	m = send(m, press("space")...)

	m = send(m, sizeMsg{gen: m.gen, path: rowNamed(m, "scratch").Path, kb: 2048})
	if rowNamed(m, "scratch").SizeKB != 2048 {
		t.Fatal("size not applied")
	}
	m = send(m, prsLoadedMsg{gen: m.gen, prs: testPRs()})
	if rowNamed(m, "merged-feature").PR == nil || rowNamed(m, "merged-feature").PR.Number != 601 {
		t.Fatal("PR not matched")
	}
	if m.premarked {
		t.Fatal("pre-marked before post-PR collect landed")
	}
	m = send(m, sessionsLoadedMsg{gen: m.gen})
	if m.premarked {
		t.Fatal("pre-marked before post-PR collect landed")
	}
	collected := append([]Worktree(nil), m.all...)
	for i := range collected {
		collected[i].Marked = true
		collected[i].SizeKB = 7
		if collected[i].Name == "open-feature" {
			collected[i].Lost = 0
		}
	}
	m = send(m, collectedMsg{gen: m.gen, ws: collected, withPRs: true})
	if !m.premarked {
		t.Fatal("pre-mark did not run")
	}
	if rowNamed(m, "scratch").SizeKB != 2048 {
		t.Fatal("collect merge clobbered SizeKB")
	}
	if rowNamed(m, "open-feature").Lost != 0 {
		t.Fatal("collect merge did not apply git facts")
	}
	if !rowNamed(m, "merged-feature").Marked {
		t.Fatal("merged clean worktree not pre-marked")
	}
	if rowNamed(m, "validate").Marked {
		t.Fatal("pre-mark overrode the user's toggle")
	}
	if rowNamed(m, "scratch").Marked || rowNamed(m, "backend").Marked {
		t.Fatal("dirty or main worktree pre-marked")
	}

	m = send(m, press("g", "space")...)
	m = send(m, collectedMsg{gen: m.gen, ws: m.all, withPRs: true})
	m = send(m, sessionsLoadedMsg{gen: m.gen})
	if rowNamed(m, "merged-feature").Marked {
		t.Fatal("pre-mark ran a second time")
	}
}

func TestGhFailureShowsQuestionMarks(t *testing.T) {
	m, _ := newTestModel(t)
	m = send(m, prsLoadedMsg{gen: m.gen, err: errors.New("gh: not logged in")})
	if !m.ghFailed || m.status == "" {
		t.Fatalf("ghFailed=%v status=%q", m.ghFailed, m.status)
	}
	for _, w := range m.all {
		if got := m.cellText(w, colPR, 2); got != "?" {
			t.Fatalf("PR cell for %s = %q, want ?", w.Name, got)
		}
	}
	m = send(m, sessionsLoadedMsg{gen: m.gen})
	if !m.premarked || !rowNamed(m, "validate").Marked {
		t.Fatal("pre-mark should still run without PR data")
	}
}

func TestStaleGenerationIgnored(t *testing.T) {
	m, _ := newTestModel(t)
	m = send(m, prsLoadedMsg{gen: m.gen - 1, prs: testPRs()})
	if rowNamed(m, "merged-feature").PR != nil {
		t.Fatal("stale PR message applied")
	}
}

func TestDeleteWithNothingMarked(t *testing.T) {
	m, _ := newTestModel(t)
	m = send(m, press("x")...)
	if m.stage != browsing || m.status == "" {
		t.Fatalf("stage=%v status=%q", m.stage, m.status)
	}
}

func TestDeleteExcludesBlockedRows(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadAllAsync(m)
	for i := range m.all {
		if m.all[i].Name == "open-feature" {
			m.all[i].Marked = true
		}
	}
	m = send(m, press("x")...)
	if rowNamed(m, "open-feature").Marked {
		t.Fatal("blocked row stayed marked")
	}
	for _, it := range m.items {
		if it.wt.Name == "open-feature" {
			t.Fatal("blocked row in delete plan")
		}
	}
}

func TestDeleteFlowAndReload(t *testing.T) {
	m, log := newTestModel(t)
	m = loadAllAsync(m)
	m = send(m, press("G", "k", "space")...)
	if !rowNamed(m, "scratch").Marked || !rowNamed(m, "merged-feature").Marked || !rowNamed(m, "validate").Marked {
		t.Fatal("expected merged-feature, validate and scratch marked")
	}
	m = send(m, press("x")...)
	if m.stage != confirming || len(m.items) != 3 {
		t.Fatalf("stage=%v items=%d", m.stage, len(m.items))
	}
	m = send(m, press("n")...)
	if m.stage != browsing {
		t.Fatal("n should go back")
	}
	m = send(m, press("x")...)

	isStep := func(msg tea.Msg) bool { _, ok := msg.(stepMsg); return ok }
	m = drive(m, press("y")[0], isStep)
	if m.stage != results {
		t.Fatalf("stage = %v, want results", m.stage)
	}
	want := []string{"merged-feature:branch", "merged-feature:worktree", "scratch:worktree", "validate:branch", "validate:worktree"}
	if got := slices.Sorted(slices.Values(log.ran)); !slices.Equal(got, want) {
		t.Fatalf("ran %v, want %v", log.ran, want)
	}
	for _, name := range []string{"merged-feature", "validate"} {
		if slices.Index(log.ran, name+":worktree") > slices.Index(log.ran, name+":branch") {
			t.Fatalf("%s ran its steps out of order: %v", name, log.ran)
		}
	}
	scratch := m.items[2]
	if scratch.err == nil || scratch.states[0] != stepFailed || scratch.states[1] != stepSkipped {
		t.Fatalf("scratch item = %+v", scratch.states)
	}

	isListed := func(msg tea.Msg) bool { _, ok := msg.(listedMsg); return ok }
	m = drive(m, press("j")[0], isListed)
	if m.stage != browsing {
		t.Fatalf("stage = %v after results", m.stage)
	}
	if !rowNamed(m, "scratch").Marked {
		t.Fatal("failed item lost its mark after reload")
	}
	if len(m.all) != 3 {
		t.Fatalf("%d rows after reload, want 3", len(m.all))
	}
	if len(m.details) != 0 {
		t.Fatal("detail cache not invalidated")
	}
	m = loadAllAsync(m)
	if !rowNamed(m, "scratch").Marked {
		t.Fatal("pre-mark after reload dropped the user's mark")
	}
}

func TestDetailLoadsLazilyAndCaches(t *testing.T) {
	m, _ := newTestModel(t)
	path := m.cursorPath
	m = send(m, detailDueMsg{path: path})
	if !m.detailLoading[path] {
		t.Fatal("detail load not started")
	}
	m = send(m, detailMsg{path: path, detail: Detail{MergeBase: "base", Stat: " a.go | 2 +-"}})
	if m.details[path].detail.MergeBase != "base" || m.detailLoading[path] {
		t.Fatal("detail not cached")
	}
	m = send(m, detailDueMsg{path: path})
	if m.detailLoading[path] {
		t.Fatal("cached detail reloaded")
	}
}

func ctrlC() tea.Msg { return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl} }

func quits(cmd tea.Cmd) bool {
	for _, msg := range runCmd(cmd) {
		if _, ok := msg.(tea.QuitMsg); ok {
			return true
		}
	}
	return false
}

func confirmed(t *testing.T) model {
	t.Helper()
	m, _ := newTestModel(t)
	m = loadAllAsync(m)
	m = send(m, press("G", "k", "space", "x")...)
	if m.stage != confirming || len(m.items) != 3 {
		t.Fatalf("stage=%v items=%d", m.stage, len(m.items))
	}
	return m
}

func TestResultsQQuitsOtherKeysReload(t *testing.T) {
	m := confirmed(t)
	isStep := func(msg tea.Msg) bool { _, ok := msg.(stepMsg); return ok }
	m = drive(m, press("y")[0], isStep)
	if m.stage != results {
		t.Fatalf("stage = %v", m.stage)
	}
	if _, cmd := m.Update(press("q")[0]); !quits(cmd) {
		t.Fatal("q on results should quit")
	}
	next, cmd := m.Update(press("j")[0])
	if quits(cmd) || next.(model).stage != browsing {
		t.Fatal("other keys on results should reload")
	}
}

func withDeleteWorkers(t *testing.T, n int) {
	old := deleteWorkers
	deleteWorkers = n
	t.Cleanup(func() { deleteWorkers = old })
}

func TestDeletesRunInParallelUpToWorkerLimit(t *testing.T) {
	withDeleteWorkers(t, 2)
	m := confirmed(t)
	next, cmd := m.Update(press("y")[0])
	m = next.(model)
	var running []string
	for _, it := range m.items {
		if it.states[0] == stepRunning {
			running = append(running, it.wt.Name)
		}
	}
	if len(running) != 2 || m.items[2].states[0] != stepQueued {
		t.Fatalf("running %v, third item %v", running, m.items[2].states)
	}
	isStep := func(msg tea.Msg) bool { _, ok := msg.(stepMsg); return ok }
	for _, msg := range runCmd(cmd) {
		if isStep(msg) {
			m = drive(m, msg, isStep)
		}
	}
	if m.stage != results || m.inFlight != 0 {
		t.Fatalf("stage = %v, inFlight = %d", m.stage, m.inFlight)
	}
	for _, it := range m.items {
		if it.states[0] == stepQueued || it.states[0] == stepRunning {
			t.Fatalf("%s never finished: %v", it.wt.Name, it.states)
		}
	}
}

func TestCtrlCDuringDeleteStopsAfterCurrentItem(t *testing.T) {
	withDeleteWorkers(t, 1)
	m := confirmed(t)
	next, cmd := m.Update(press("y")[0])
	m = next.(model)
	next, quit := m.Update(ctrlC())
	m = next.(model)
	if quits(quit) || !m.stopping || m.stage != deleting {
		t.Fatalf("first ctrl+c should request a stop: stopping=%v stage=%v", m.stopping, m.stage)
	}
	isStep := func(msg tea.Msg) bool { _, ok := msg.(stepMsg); return ok }
	for _, msg := range runCmd(cmd) {
		if isStep(msg) {
			m = drive(m, msg, isStep)
		}
	}
	if m.stage != results {
		t.Fatalf("stage = %v, want results", m.stage)
	}
	first := m.items[0]
	if !first.finished() || first.cancelled {
		t.Fatalf("running item should finish all its steps: %+v", first.states)
	}
	for _, it := range m.items[1:] {
		if !it.cancelled || it.removed() {
			t.Fatalf("%s should not have started: %+v", it.wt.Name, it.states)
		}
	}
	if out := ansi.Strip(m.View().Content); !strings.Contains(out, "2 not started") {
		t.Fatalf("results should count skipped items:\n%s", out)
	}
}

func TestSecondCtrlCDuringDeleteQuits(t *testing.T) {
	m := confirmed(t)
	m = send(m, press("y")[0], ctrlC())
	if _, cmd := m.Update(ctrlC()); !quits(cmd) {
		t.Fatal("second ctrl+c should quit")
	}
}

func TestDeleteWaitsForPRsAndSessions(t *testing.T) {
	m, _ := newTestModel(t)
	m = send(m, press("space", "x")...)
	if m.stage != browsing || !strings.Contains(m.status, "PRs") || !strings.Contains(m.status, "sessions") {
		t.Fatalf("stage=%v status=%q", m.stage, m.status)
	}
}

func TestRowBecomingBlockedLosesMark(t *testing.T) {
	m, _ := newTestModel(t)
	m = send(m, press("j", "space")...)
	if !rowNamed(m, "open-feature").Marked {
		t.Fatal("open-feature should be markable before PRs load")
	}
	m = loadAllAsync(m)
	if rowNamed(m, "open-feature").Marked {
		t.Fatal("row with an open PR kept its mark")
	}
}

func TestGhFailureReasonShownOnce(t *testing.T) {
	m, _ := newTestModel(t)
	m.src.prs = func(string, []string) ([]PR, error) {
		return nil, errors.New(`error parsing "owner" value: none of the git remotes configured for this repository point to a known GitHub host`)
	}
	m = send(m, prsLoadedMsg{gen: m.gen, err: errors.New("none of the git remotes configured")})
	if m.status != "✕ PRs unavailable · no GitHub remote" {
		t.Fatalf("status = %q", m.status)
	}
	m = send(m, press("j")...)
	m.stage = results
	isLoad := func(msg tea.Msg) bool {
		switch msg.(type) {
		case listedMsg, prsLoadedMsg:
			return true
		}
		return false
	}
	m = drive(m, press("j")[0], isLoad)
	if !m.ghFailed || m.status != "" {
		t.Fatalf("reload repeated the gh warning: ghFailed=%v status=%q", m.ghFailed, m.status)
	}
}

func TestFetchInvalidatesCachedDetail(t *testing.T) {
	m, _ := newTestModel(t)
	path := m.cursorPath
	m = send(m, detailDueMsg{path: path})
	stale := detailMsg{gen: m.detailGen, path: path, detail: Detail{MergeBase: "old"}}
	m = send(m, stale)
	if !m.detailFresh(path) {
		t.Fatal("detail not cached")
	}
	m = send(m, fetchDoneMsg{gen: m.gen})
	if m.detailFresh(path) || !m.detailLoading[path] {
		t.Fatal("fetch should mark the cached detail stale and reload it")
	}
	if m.details[path].detail.MergeBase != "old" {
		t.Fatal("stale detail should stay on screen until the fresh one lands")
	}
	m = send(m, stale)
	if m.detailFresh(path) {
		t.Fatal("pre-fetch detail result accepted after fetch")
	}
	msgs := runCmd(m.diff())
	if len(msgs) != 1 || msgs[0].(diffReadyMsg).base == "old" {
		t.Fatalf("diff used the pre-fetch merge-base: %+v", msgs)
	}
}

func TestDiffOnVanishedWorktree(t *testing.T) {
	m, _ := newTestModel(t)
	msgs := runCmd(m.diff())
	if len(msgs) != 1 {
		t.Fatalf("msgs = %v", msgs)
	}
	m = send(m, msgs[0])
	if m.status != "✕ diff: worktree directory is gone" {
		t.Fatalf("status = %q", m.status)
	}
}

func TestUnreadableSessionsWarnOnceAndStayInHeader(t *testing.T) {
	m, _ := newTestModel(t)
	m = send(m, sessionsLoadedMsg{gen: m.gen, err: errSessionsUnreadable})
	if !strings.Contains(m.status, "won't block deletion") {
		t.Fatalf("status = %q", m.status)
	}
	m = send(m, press("j")...)
	m = send(m, sessionsLoadedMsg{gen: m.gen, err: errSessionsUnreadable})
	if m.status != "" {
		t.Fatalf("warning repeated: %q", m.status)
	}
	if out := ansi.Strip(m.View().Content); !strings.Contains(out, "sessions unreadable") || !strings.Contains(out, "can't read Claude session data") {
		t.Fatalf("header/detail missing warning:\n%s", out)
	}
	m = send(m, sessionsLoadedMsg{gen: m.gen})
	if m.sessionsBlind {
		t.Fatal("warning should clear once sessions read")
	}
}

func TestSquashMergedShowsInMain(t *testing.T) {
	m, _ := newTestModel(t)
	for i := range m.all {
		if m.all[i].Name == "scratch" {
			m.all[i].SquashMerged = true
		}
	}
	m = send(m, prsLoadedMsg{gen: m.gen, err: errors.New("offline")})
	if out := ansi.Strip(m.View().Content); !strings.Contains(out, "in main") {
		t.Fatalf("squash-merged row should say in main even when gh fails:\n%s", out)
	}
}

func TestDeleteRescansProcessesAtConfirm(t *testing.T) {
	m, log := newTestModel(t)
	m = loadAllAsync(m)
	m = send(m, press("G", "k", "space")...)
	merged := rowNamed(m, "merged-feature")
	if !merged.Marked {
		t.Fatal("merged-feature should be pre-marked")
	}
	log.procs = map[string][]Proc{merged.Path: {{PID: 9, Command: "zsh"}}}
	m = send(m, press("x")...)
	if rowNamed(m, "merged-feature").Marked {
		t.Fatal("row with a shell open in it stayed marked")
	}
	for _, it := range m.items {
		if it.wt.Name == "merged-feature" {
			t.Fatal("row with a shell open in it reached the confirm list")
		}
	}
	if !strings.Contains(m.status, "merged-feature in use by zsh") {
		t.Fatalf("status = %q", m.status)
	}
}
