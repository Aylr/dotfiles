package main

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func loadedModel(t *testing.T, width, height int) model {
	t.Helper()
	m, _ := newTestModel(t)
	m = loadAllAsync(m)
	m = send(m,
		sizeMsg{gen: m.gen, path: rowNamed(m, "merged-feature").Path, kb: 412 * 1024},
		sizeMsg{gen: m.gen, path: rowNamed(m, "validate").Path, kb: 2200 * 1024},
		sessionsLoadedMsg{gen: m.gen, sessions: map[string][]Session{
			testRoot + "/.worktrees/open-feature":   {{ID: "0123456789ab", Title: "Patient flow entities and a very long session title", Live: true}},
			testRoot + "/.worktrees/merged-feature": {{ID: "ba9876543210", Title: "Entity programs"}},
		}},
		detailMsg{key: m.cursorKey, detail: Detail{
			MergeBase: "base",
			Stat:      " app/models.py | 12 +++++++---\n 1 file changed, 7 insertions(+), 3 deletions(-)",
			Commits:   []Commit{{Hash: "abc1234", When: "3 days ago", Subject: "Add entity programs"}},
		}},
		tea.WindowSizeMsg{Width: width, Height: height},
	)
	return m
}

func stages(t *testing.T, width, height int) map[string]model {
	t.Helper()
	m := loadedModel(t, width, height)
	confirming := send(m, press("x")...)
	deleting := send(confirming, press("y")...)
	isStep := func(msg tea.Msg) bool { _, ok := msg.(stepMsg); return ok }
	done := drive(confirming, press("y")[0], isStep)
	filtering := send(m, press("/", "f", "e")...)
	return map[string]model{
		"browsing":   m,
		"filtering":  filtering,
		"help":       send(m, press("?")...),
		"confirming": confirming,
		"deleting":   deleting,
		"results":    done,
	}
}

func TestViewFitsWidth(t *testing.T) {
	for _, width := range []int{60, 80, 120, 200} {
		for name, m := range stages(t, width, 40) {
			out := m.View().Content
			lines := strings.Split(out, "\n")
			if len(lines) > 40 {
				t.Errorf("%s at %d: %d lines, height 40", name, width, len(lines))
			}
			for i, line := range lines {
				if w := lipgloss.Width(line); w > width {
					t.Errorf("%s at %d: line %d is %d wide: %q", name, width, i, w, ansi.Strip(line))
				}
			}
			if os.Getenv("WT_PREVIEW") != "" && width == previewWidth() {
				t.Logf("%s\n%s", name, out)
			}
		}
	}
}

func TestViewStages(t *testing.T) {
	s := stages(t, 120, 40)
	checks := map[string]string{
		"confirming": "Remote branch deletes are permanent.",
		"deleting":   "Deleting",
		"results":    "any other key to continue",
		"browsing":   "to reclaim",
	}
	for stage, want := range checks {
		if out := ansi.Strip(s[stage].View().Content); !strings.Contains(out, want) {
			t.Errorf("%s view missing %q", stage, want)
		}
	}
}

func TestViewNamesBothKinds(t *testing.T) {
	s := stages(t, 120, 40)
	checks := map[string][]string{
		"browsing":   {"KIND", "worktree", "branch", "4 worktrees · 1 branch"},
		"confirming": {"Delete 2 worktrees and 1 branch?", "Frees "},
		"results":    {"Deleted 2 worktrees and 1 branch"},
	}
	for stage, wants := range checks {
		out := ansi.Strip(s[stage].View().Content)
		for _, want := range wants {
			if !strings.Contains(out, want) {
				t.Errorf("%s view missing %q:\n%s", stage, want, out)
			}
		}
	}
	out := ansi.Strip(s["browsing"].View().Content)
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "old-branch") && strings.Contains(line, "?") {
			t.Errorf("branch row should have no size cell: %q", line)
		}
	}
}

func TestBranchDetailOmitsSessions(t *testing.T) {
	m := loadedModel(t, 120, 40)
	if cursorName(m) != "old-branch" {
		t.Fatalf("cursor on %q", cursorName(m))
	}
	out := ansi.Strip(m.View().Content)
	if !strings.Contains(out, "no worktree") || strings.Contains(out, "no sessions") {
		t.Errorf("branch detail wrong:\n%s", out)
	}
}

func TestBranchOnlyConfirmSkipsFrees(t *testing.T) {
	m := loadedModel(t, 120, 40)
	for i := range m.all {
		if !m.all[i].BranchOnly() {
			m.all[i].Marked = false
		}
	}
	m = send(m, press("x")...)
	out := ansi.Strip(m.View().Content)
	if !strings.Contains(out, "Delete 1 branch?") || strings.Contains(out, "Frees") {
		t.Errorf("confirm for a lone branch:\n%s", out)
	}
}

func TestTinyTerminal(t *testing.T) {
	for _, size := range [][2]int{{40, 10}, {59, 40}, {120, 8}} {
		m := loadedModel(t, size[0], size[1])
		out := m.View().Content
		if !strings.Contains(out, "too small") {
			t.Errorf("%dx%d: expected too-small message", size[0], size[1])
		}
		for _, line := range strings.Split(out, "\n") {
			if lipgloss.Width(line) > size[0] {
				t.Errorf("%dx%d: line too wide", size[0], size[1])
			}
		}
	}
}

func TestPRCellIsHyperlink(t *testing.T) {
	m := loadedModel(t, 120, 40)
	out := m.View().Content
	for _, n := range []string{"601", "618"} {
		if !strings.Contains(out, "\x1b]8;;https://github.com/acme/backend/pull/"+n+"\a") {
			t.Fatalf("PR #%s cell has no OSC 8 link", n)
		}
	}
}

func previewWidth() int {
	if os.Getenv("WT_PREVIEW") == "80" {
		return 80
	}
	if os.Getenv("WT_PREVIEW") == "60" {
		return 60
	}
	return 120
}

func TestHeaderDegradesBeforeTruncating(t *testing.T) {
	for _, width := range []int{60, 80, 120} {
		m := loadedModel(t, width, 40)
		m.pending = map[string]bool{"fetch": true, "prs": true, "sizes": true}
		m.sizesDone, m.sizesTotal = 3, 24
		header := ansi.Strip(m.headerView())
		if strings.Contains(header, "…") || !strings.Contains(header, "marked") {
			t.Errorf("width %d: header truncated or lost the mark count: %q", width, header)
		}
		if !strings.ContainsAny(header, "⣾⣽⣻⢿⡿⣟⣯⣷") {
			t.Errorf("width %d: header dropped the spinner: %q", width, header)
		}
		if w := lipgloss.Width(header); w != width {
			t.Errorf("width %d: header is %d wide", width, w)
		}
	}
	if header := ansi.Strip(loadedModel(t, 120, 40).headerView()); !strings.Contains(header, "to reclaim") {
		t.Errorf("wide header should keep the full reclaim text: %q", header)
	}
}

func TestColumnsStayPutWhenPRsLand(t *testing.T) {
	m, _ := newTestModel(t)
	before := m.tableLayout(118).cols
	after := loadAllAsync(m).tableLayout(118).cols
	if len(before) != len(after) {
		t.Fatalf("columns changed: %d → %d", len(before), len(after))
	}
	for i := range before {
		if before[i].width != after[i].width {
			t.Errorf("%s width %d → %d", before[i].title, before[i].width, after[i].width)
		}
	}
}

func TestOnlyMainShowsEmptyState(t *testing.T) {
	m := newModel(testRepo(), testWorktrees()[:1], stubSources(&stepLog{deleted: map[string]bool{}}))
	m = send(m, tea.WindowSizeMsg{Width: 100, Height: 30})
	out := ansi.Strip(m.View().Content)
	for _, want := range []string{"No worktrees or branches besides main", "no other worktrees or branches"} {

		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
}

func TestWideTitlesKeepRowsAligned(t *testing.T) {
	m, _ := newTestModel(t)
	m = loadAllAsync(m)
	m = send(m, sessionsLoadedMsg{gen: m.gen, sessions: map[string][]Session{
		testRoot + "/.worktrees/merged-feature": {{ID: "a1", Title: "日本語のセッションタイトルです長い長い"}},
		testRoot + "/.worktrees/validate":       {{ID: "b2", Title: "🚀 launch prep ✨ with a long tail of words"}},
		testRoot + "/.worktrees/scratch":        {{ID: "c3", Title: "Ελληνικά καὶ عربى mixed"}},
	}})
	for _, width := range []int{80, 120, 200} {
		m = send(m, tea.WindowSizeMsg{Width: width, Height: 40})
		lines := strings.Split(m.View().Content, "\n")
		for i, line := range lines[:len(lines)-1] {
			if w := lipgloss.Width(line); w != width {
				t.Errorf("width %d: line %d is %d wide: %q", width, i, w, ansi.Strip(line))
			}
		}
	}
}
