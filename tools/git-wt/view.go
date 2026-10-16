package main

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	minWidth      = 60
	minName       = 20
	labelWidth    = 10
	maxClaude     = 36
	balancedWidth = 24
	maxUntracked  = 5
	prReserve     = len("draft #999")
)

type palette struct {
	text, muted, faint, accent, red, yellow, green, cursorBG, border color.Color
}

type styles struct {
	c                                       palette
	text, muted, faint, accent, red, yellow lipgloss.Style
	green, bold, badge, title, header       lipgloss.Style
}

func newStyles(isDark bool) styles {
	ld := lipgloss.LightDark(isDark)
	hex := func(light, dark string) color.Color { return ld(lipgloss.Color(light), lipgloss.Color(dark)) }
	c := palette{
		text:     hex("#1F2328", "#E4E4EA"),
		muted:    hex("#57606A", "#A0A3AE"),
		faint:    hex("#6E7781", "#80838F"),
		accent:   hex("#7C3AED", "#B4A2FF"),
		red:      hex("#CF222E", "#FF7B72"),
		yellow:   hex("#9A6700", "#E3B341"),
		green:    hex("#1A7F37", "#56D364"),
		cursorBG: hex("#EEF0F8", "#262839"),
		border:   hex("#D0D7DE", "#3A3C4B"),
	}
	fg := func(col color.Color) lipgloss.Style { return lipgloss.NewStyle().Foreground(col) }
	return styles{
		c:      c,
		text:   fg(c.text),
		muted:  fg(c.muted),
		faint:  fg(c.faint),
		accent: fg(c.accent),
		red:    fg(c.red),
		yellow: fg(c.yellow),
		green:  fg(c.green),
		bold:   fg(c.text).Bold(true),
		badge:  lipgloss.NewStyle().Bold(true).Foreground(hex("#FFFFFF", "#16161E")).Background(c.accent),
		title:  fg(c.accent).Bold(true),
		header: fg(c.muted).Bold(true),
	}
}

type layout struct {
	tableRows, tableH, detailH int
	ok                         bool
}

func (m model) layout() layout {
	avail := m.height - 1 - lipgloss.Height(m.footerView())
	detailH := max(8, m.height/3)
	tableH := avail - detailH
	if want := max(len(m.all), 2) + 3; tableH > want {
		tableH = want
		detailH = avail - tableH
	}
	l := layout{tableRows: tableH - 3, tableH: tableH, detailH: detailH}
	l.ok = m.width >= minWidth && l.tableRows >= 1
	return l
}

func (m *model) sizeDetail() {
	l := m.layout()
	m.detail.SetWidth(max(m.width-4, 1))
	m.detail.SetHeight(max(l.detailH-2, 1))
	m.help.SetWidth(max(m.width-2, 1))
	m.filter.SetWidth(max(m.width/3, 10))
}

func (m model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.WindowTitle = "git wt · " + m.repo.Name
	if m.stage == deleting {
		v.ProgressBar = tea.NewProgressBar(tea.ProgressBarDefault, m.progress())
	}
	return v
}

func (m model) render() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	l := m.layout()
	if !l.ok {
		return m.tooSmall()
	}
	screen := m.browseView(l)
	if m.stage == browsing {
		return screen
	}
	return m.overlay(screen, m.modalView())
}

func (m model) tooSmall() string {
	msg := fmt.Sprintf("terminal too small · %d×%d, need %d×14", m.width, m.height, minWidth)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.st.muted.Render(truncate(msg, m.width)))
}

func (m model) browseView(l layout) string {
	return strings.Join([]string{
		m.headerView(),
		m.tableView(l),
		m.detailView(l),
		m.footerView(),
	}, "\n")
}

type headerLevel int

const (
	headerFull headerLevel = iota
	headerShortActivity
	headerShortReclaim
	headerNoCount
	headerSpinnerOnly
)

func (m model) headerView() string {
	var left, right string
	for level := headerFull; level <= headerSpinnerOnly; level++ {
		left, right = m.headerLeft(level), m.activityView(level)
		if lipgloss.Width(left)+lipgloss.Width(right)+1 <= m.width {
			break
		}
	}
	return fitLine(left, right, m.width)
}

func (m model) headerLeft(level headerLevel) string {
	parts := []string{m.st.bold.Render(m.repo.Name)}
	if level < headerNoCount {
		parts = append(parts, m.worktreeCount())
	}
	if n, kb, partial := m.markedTotals(); n > 0 {
		reclaim := humanLong(kb)
		if partial {
			reclaim += "+"
		}
		if level < headerShortReclaim {
			reclaim += " to reclaim"
		}
		parts = append(parts, m.st.red.Render(fmt.Sprintf("%d marked", n)), reclaim)
	}
	if q := m.filter.Value(); q != "" {
		parts = append(parts, m.st.accent.Render(fmt.Sprintf("/%s %d/%d", q, len(m.visible), len(m.all))))
	}
	return " " + m.st.badge.Render(" git wt ") + "  " + strings.Join(parts, m.st.muted.Render(" · "))
}

func (m model) worktreeCount() string {
	n := 0
	for _, w := range m.all {
		if !w.Main {
			n++
		}
	}
	if n == 0 {
		return "no other worktrees"
	}
	return count(n, "worktree", "worktrees")
}

func (m model) markedTotals() (n int, kb int64, partial bool) {
	for _, w := range m.all {
		if !w.Marked {
			continue
		}
		n++
		if w.SizeKB > 0 {
			kb += w.SizeKB
		} else {
			partial = true
		}
	}
	return n, kb, partial
}

func (m model) activityView(level headerLevel) string {
	var steps []string
	for _, p := range []struct{ key, label string }{
		{"list", "reloading"},
		{"fetch", "fetching"},
		{"prs", "PRs"},
		{"sessions", "sessions"},
		{"sizes", fmt.Sprintf("sizes %d/%d", m.sizesDone, m.sizesTotal)},
	} {
		if m.pending[p.key] {
			steps = append(steps, p.label)
		}
	}
	switch {
	case len(steps) > 0 && level >= headerSpinnerOnly:
		return m.spinner.View() + " "
	case len(steps) > 0 && level >= headerShortActivity:
		return m.spinner.View() + " " + m.st.muted.Render(steps[0]) + " "
	case len(steps) > 0:
		return m.spinner.View() + " " + m.st.muted.Render(strings.Join(steps, " · ")) + " "
	case m.loadedIn > 0:
		return m.st.faint.Render("✓ "+m.loadedIn.Round(100*time.Millisecond).String()) + " "
	}
	return ""
}

func fitLine(left, right string, width int) string {
	if lipgloss.Width(left)+lipgloss.Width(right)+1 > width {
		if width-lipgloss.Width(right)-1 < 30 {
			right = ""
		}
		left = truncate(left, width-lipgloss.Width(right)-1)
	}
	gap := max(width-lipgloss.Width(left)-lipgloss.Width(right), 0)
	return left + strings.Repeat(" ", gap) + right
}

type colID int

const (
	colMark colID = iota
	colName
	colAge
	colState
	colMain
	colPR
	colSize
	colClaude
)

type column struct {
	id    colID
	title string
	width int
	right bool
}

type tableLayout struct {
	cols   []column
	aheadW int
}

func (m model) tableLayout(inner int) tableLayout {
	tl := tableLayout{aheadW: 2}
	for _, w := range m.all {
		tl.aheadW = max(tl.aheadW, len(fmt.Sprint(w.Ahead))+1)
	}
	measure := func(id colID, title string) int {
		width := lipgloss.Width(title)
		for _, w := range m.all {
			width = max(width, lipgloss.Width(m.cellText(w, id, tl.aheadW)))
		}
		return width
	}
	nameMax := measure(colName, "NAME")
	nameMin := min(minName, nameMax)
	claudeMax := min(measure(colClaude, "CLAUDE"), maxClaude)
	optional := []column{
		{id: colAge, title: "AGE", right: true},
		{id: colState, title: "STATE"},
		{id: colPR, title: "PR"},
		{id: colSize, title: "SIZE", right: true},
		{id: colMain, title: "±MAIN"},
		{id: colClaude, title: "CLAUDE"},
	}
	used := 1 + 3 + 1 + nameMin + 1
	included := map[colID]column{}
	for _, col := range optional {
		col.width = measure(col.id, col.title)
		switch {
		case col.id == colClaude:
			col.width = min(claudeMax, 10)
		case col.id == colPR && !m.ghFailed:
			col.width = max(col.width, prReserve)
		}
		if used+2+col.width <= inner {
			used += 2 + col.width
			included[col.id] = col
		}
	}
	extra := inner - used
	nameW := nameMin
	claude, hasClaude := included[colClaude]
	grow := func(width *int, limit int) {
		n := max(min(extra, limit-*width), 0)
		*width += n
		extra -= n
	}
	grow(&nameW, min(nameMax, balancedWidth))
	if hasClaude {
		grow(&claude.width, min(claudeMax, balancedWidth))
	}
	grow(&nameW, nameMax)
	if hasClaude {
		grow(&claude.width, claudeMax)
		included[colClaude] = claude
	}
	tl.cols = []column{{id: colMark, width: 3}, {id: colName, title: "NAME", width: nameW}}
	for id := colAge; id <= colClaude; id++ {
		if c, ok := included[id]; ok {
			tl.cols = append(tl.cols, c)
		}
	}
	return tl
}

func (m model) cellText(w Worktree, id colID, aheadW int) string {
	switch id {
	case colMark:
		if w.Marked {
			return "[x]"
		}
		return "[ ]"
	case colName:
		return w.Name
	case colAge:
		return age(w.LastActivity)
	case colState:
		return stateText(w)
	case colMain:
		return fmt.Sprintf("%-*s −%d", aheadW, fmt.Sprintf("+%d", w.Ahead), w.Behind)
	case colPR:
		if m.ghFailed {
			return "?"
		}
		return w.PR.Word()
	case colSize:
		if w.Main {
			return ""
		}
		return sizeText(w.SizeKB)
	case colClaude:
		return claudeText(w)
	}
	return ""
}

func stateText(w Worktree) string {
	switch {
	case len(w.DirtyFiles) > 0:
		return fmt.Sprintf("%d dirty", len(w.DirtyFiles))
	case len(w.Untracked) > 0:
		return fmt.Sprintf("%d untracked", len(w.Untracked))
	case w.Lost > 0:
		return fmt.Sprintf("%d unpushed", w.Lost)
	case w.Orphaned:
		return "⚠ orphaned"
	case w.Detached:
		return "detached"
	}
	return "clean"
}

func isLive(w Worktree) bool {
	for _, s := range w.Sessions {
		if s.Live {
			return true
		}
	}
	return false
}

func claudeText(w Worktree) string {
	switch {
	case isLive(w):
		return "● live"
	case len(w.Sessions) > 0:
		return w.Sessions[0].Title
	}
	return ""
}

func (m model) tableView(l layout) string {
	inner := m.width - 2
	tl := m.tableLayout(inner)
	lines := []string{m.headerRow(tl, inner)}
	end := min(m.offset+l.tableRows, len(m.visible))
	for i := m.offset; i < end; i++ {
		lines = append(lines, m.renderRow(m.all[m.visible[i]], tl, i == m.cursor, inner))
	}
	switch {
	case len(m.visible) == 0:
		lines = append(lines, m.st.muted.Render(fmt.Sprintf("  no worktrees match %q · esc clears", m.filter.Value())))
	case len(m.all) == 1 && m.filter.Value() == "":
		lines = append(lines, m.st.muted.Render("  No worktrees besides main · nothing to clean up"))
	}
	label := ""
	if len(m.visible) > l.tableRows {
		label = m.st.muted.Render(fmt.Sprintf("%d/%d", m.cursor+1, len(m.visible)))
	}
	return m.box(lines, m.width, l.tableH, "", label, m.st.c.border)
}

func (m model) headerRow(tl tableLayout, inner int) string {
	var b strings.Builder
	b.WriteString(" ")
	for _, col := range tl.cols {
		b.WriteString(strings.Repeat(" ", colGap(col.id)))
		st := m.st.header.Width(col.width)
		if col.right {
			st = st.Align(lipgloss.Right)
		}
		b.WriteString(st.Render(col.title))
	}
	return b.String()
}

func colGap(id colID) int {
	switch id {
	case colMark:
		return 0
	case colName:
		return 1
	}
	return 2
}

func (m model) renderRow(w Worktree, tl tableLayout, cursor bool, inner int) string {
	blocked := len(BlockedReasons(w, m.repo.DefaultBranch)) > 0
	base := m.st.text
	switch {
	case w.Marked:
		base = m.st.red
	case blocked || w.Main:
		base = m.st.faint
	}
	if cursor {
		base = base.Background(m.st.c.cursorBG)
	}
	var b strings.Builder
	if cursor {
		b.WriteString(base.Foreground(m.st.c.accent).Render("▌"))
	} else {
		b.WriteString(base.Render(" "))
	}
	for _, col := range tl.cols {
		b.WriteString(base.Render(strings.Repeat(" ", colGap(col.id))))
		b.WriteString(m.renderCell(w, col, base, tl.aheadW, cursor, blocked))
	}
	if pad := inner - lipgloss.Width(b.String()); pad > 0 {
		b.WriteString(base.Render(strings.Repeat(" ", pad)))
	}
	return b.String()
}

func (m model) renderCell(w Worktree, col column, base lipgloss.Style, aheadW int, cursor, blocked bool) string {
	text := truncate(m.cellText(w, col.id, aheadW), col.width)
	st := base.Width(col.width)
	if col.right {
		st = st.Align(lipgloss.Right)
	}
	plain := !w.Marked && !blocked && !w.Main
	switch col.id {
	case colName:
		st = st.Bold(cursor)
	case colMark:
		st = st.Bold(w.Marked)
	case colState:
		if strings.HasPrefix(text, "⚠") {
			st = st.Foreground(m.st.c.yellow)
		}
	case colClaude:
		if isLive(w) {
			st = st.Foreground(m.st.c.green).Bold(true)
		} else if plain {
			st = st.Foreground(m.st.c.muted)
		}
	case colPR:
		if w.PR != nil && !m.ghFailed && w.PR.URL != "" {
			return m.prLink(w.PR, text, base, col.width)
		}
	case colAge, colSize, colMain:
		if plain {
			st = st.Foreground(m.st.c.muted)
		}
	}
	return st.Render(text)
}

func (m model) prLink(pr *PR, text string, base lipgloss.Style, width int) string {
	link := base.Foreground(m.prColor(pr)).Underline(true).Hyperlink(pr.URL).Render(text)
	return link + base.Render(strings.Repeat(" ", max(width-lipgloss.Width(text), 0)))
}

func (m model) prColor(pr *PR) color.Color {
	switch {
	case pr.State == "MERGED":
		return m.st.c.accent
	case pr.State == "CLOSED":
		return m.st.c.red
	case pr.Draft:
		return m.st.c.muted
	}
	return m.st.c.green
}

func (m model) box(lines []string, width, height int, title, label string, border color.Color) string {
	inner := width - 2
	bc := lipgloss.NewStyle().Foreground(border)
	top := bc.Render("╭" + strings.Repeat("─", inner) + "╮")
	if title != "" {
		t := truncate(title, inner-4)
		top = bc.Render("╭─ ") + m.st.title.Render(t) + bc.Render(" "+strings.Repeat("─", max(inner-3-lipgloss.Width(t), 0))+"╮")
	}
	bottom := bc.Render("╰" + strings.Repeat("─", inner) + "╯")
	if label != "" && lipgloss.Width(label)+4 <= inner {
		bottom = bc.Render("╰"+strings.Repeat("─", inner-lipgloss.Width(label)-3)+" ") + label + bc.Render(" ─╯")
	}
	out := []string{top}
	for i := 0; i < height-2; i++ {
		line := ""
		if i < len(lines) {
			line = truncate(lines[i], inner)
		}
		out = append(out, bc.Render("│")+line+strings.Repeat(" ", max(inner-lipgloss.Width(line), 0))+bc.Render("│"))
	}
	return strings.Join(append(out, bottom), "\n")
}

func (m model) detailView(l layout) string {
	title := ""
	if w := m.current(); w != nil {
		title = w.Name
	}
	var lines []string
	for _, line := range strings.Split(m.detail.View(), "\n") {
		lines = append(lines, " "+line)
	}
	label := ""
	if !m.detail.AtTop() || !m.detail.AtBottom() {
		label = m.st.muted.Render(fmt.Sprintf("%d%% · pgup/pgdn", int(m.detail.ScrollPercent()*100)))
	}
	return m.box(lines, m.width, l.detailH, title, label, m.st.c.border)
}

func (m *model) setDetailContent() {
	w := m.current()
	if w == nil {
		m.detail.SetContent(m.st.muted.Render("no worktree selected"))
		return
	}
	lines := m.detailLines(*w)
	width := m.detail.Width()
	for i := range lines {
		lines[i] = truncate(lines[i], width)
	}
	m.detail.SetContent(strings.Join(lines, "\n"))
}

func (m model) detailLines(w Worktree) []string {
	var lines []string
	add := func(label string, values ...string) {
		for i, v := range values {
			l := ""
			if i == 0 {
				l = label
			}
			lines = append(lines, m.st.muted.Width(labelWidth).Render(l)+v)
		}
	}
	add("branch", m.branchLine(w))
	add("PR", m.prLine(w))
	if reasons := BlockedReasons(w, m.repo.DefaultBranch); len(reasons) > 0 {
		var vs []string
		for _, r := range reasons {
			vs = append(vs, m.st.red.Render("✕ "+r))
		}
		add("blocked", vs...)
	}
	add("vs main", m.diffLines(w)...)
	if n := len(w.Untracked); n > 0 {
		vs := append([]string(nil), w.Untracked[:min(n, maxUntracked)]...)
		if n > maxUntracked {
			vs = append(vs, m.st.muted.Render(fmt.Sprintf("+%d more", n-maxUntracked)))
		}
		add("untracked", vs...)
	}
	if res, ok := m.details[w.Path]; ok && len(res.detail.Commits) > 0 {
		var vs []string
		for _, c := range res.detail.Commits {
			vs = append(vs, m.st.yellow.Render(c.Hash)+"  "+c.Subject+m.st.muted.Render("  · "+c.When))
		}
		add("commits", vs...)
	}
	add("claude", m.sessionLines(w)...)
	return lines
}

func (m model) branchLine(w Worktree) string {
	if w.Detached {
		head := w.Head
		if len(head) > 7 {
			head = head[:7]
		}
		s := "detached at " + m.st.yellow.Render(head)
		if w.Orphaned {
			s += m.st.yellow.Render(" · ⚠ orphaned, unreachable from any branch")
		}
		return s
	}
	s := m.st.bold.Render(w.Branch)
	if w.Upstream != "" {
		s += m.st.muted.Render(" → ") + w.Upstream
	} else {
		s += m.st.muted.Render(" · no upstream")
	}
	if w.Lost > 0 {
		s += m.st.yellow.Render(" · " + count(w.Lost, "commit only here", "commits only here"))
	}
	return s
}

func (m model) prLine(w Worktree) string {
	switch {
	case m.ghFailed:
		return m.st.muted.Render("? " + m.ghErr)
	case w.PR == nil:
		return m.st.muted.Render("—")
	}
	pr := w.PR
	title := m.st.text.Underline(true).Hyperlink(pr.URL).Render(pr.Title)
	state := map[string]string{"MERGED": "merged", "CLOSED": "closed", "OPEN": "open"}[pr.State]
	stateStyle := map[string]lipgloss.Style{"MERGED": m.st.accent, "CLOSED": m.st.red, "OPEN": m.st.green}[pr.State]
	if pr.Draft && pr.State == "OPEN" {
		state, stateStyle = "draft", m.st.muted
	}
	s := m.st.muted.Render(fmt.Sprintf("#%d ", pr.Number)) + title + "  " + stateStyle.Render(state)
	if review := reviewText(pr.ReviewDecision); review != "" {
		s += m.st.muted.Render(" · " + review)
	}
	return s
}

func reviewText(decision string) string {
	return strings.ToLower(strings.ReplaceAll(decision, "_", " "))
}

func (m model) diffLines(w Worktree) []string {
	head := m.st.green.Render(fmt.Sprintf("+%d", w.Ahead)) + m.st.muted.Render(" ahead  ") +
		m.st.red.Render(fmt.Sprintf("−%d", w.Behind)) + m.st.muted.Render(" behind "+m.repo.Base)
	res, ok := m.details[w.Path]
	switch {
	case !ok && m.detailLoading[w.Path]:
		return []string{head, m.spinner.View() + m.st.muted.Render(" loading diff")}
	case !ok:
		return []string{head, m.st.faint.Render("…")}
	case res.err != nil:
		return []string{head, m.st.red.Render("✕ " + res.err.Error())}
	case strings.TrimSpace(res.detail.Stat) == "":
		return []string{head, m.st.muted.Render("no changes")}
	}
	lines := []string{head}
	for _, l := range strings.Split(res.detail.Stat, "\n") {
		lines = append(lines, m.statLine(l))
	}
	return lines
}

func (m model) statLine(line string) string {
	file, graph, ok := strings.Cut(line, "|")
	if !ok {
		return m.st.muted.Render(strings.TrimSpace(line))
	}
	fields := strings.Fields(graph)
	if len(fields) != 2 || strings.Trim(fields[1], "+-") != "" {
		return strings.TrimLeft(file, " ") + m.st.muted.Render("|") + graph
	}
	bar := fields[1]
	plus := strings.Count(bar, "+")
	counts := graph[:strings.Index(graph, bar)]
	return strings.TrimLeft(file, " ") + m.st.muted.Render("|") + counts +
		m.st.green.Render(bar[:plus]) + m.st.red.Render(bar[plus:])
}

func (m model) sessionLines(w Worktree) []string {
	if len(w.Sessions) == 0 {
		return []string{m.st.muted.Render("no sessions")}
	}
	var lines []string
	for _, s := range w.Sessions {
		mark := "  "
		if s.Live {
			mark = m.st.green.Bold(true).Render("● ")
		}
		meta := "  " + relTime(s.Last) + "  " + shortID(s.ID)
		if s.Live {
			meta += "  " + m.st.green.Render("live")
		}
		lines = append(lines, mark+s.Title+m.st.muted.Render(meta))
	}
	return lines
}

func (m model) footerView() string {
	var line string
	switch {
	case m.filtering:
		line = m.filter.View() + "   " + m.help.ShortHelpView(m.keys.filterHelp())
	case m.status != "" && strings.HasPrefix(m.status, "✕"):
		line = m.st.red.Render(m.status)
	case m.status != "":
		line = m.st.yellow.Render(m.status)
	default:
		line = m.help.View(m.keys)
	}
	lines := strings.Split(line, "\n")
	for i := range lines {
		lines[i] = truncate(" "+lines[i], m.width)
	}
	return strings.Join(lines, "\n")
}

func (m model) overlay(base, modal string) string {
	lines := strings.Split(ansi.Strip(base), "\n")
	for i := range lines {
		lines[i] = m.st.faint.Render(lines[i])
	}
	x := max((m.width-lipgloss.Width(modal))/2, 0)
	y := max((m.height-lipgloss.Height(modal))/2, 0)
	return lipgloss.NewCompositor(
		lipgloss.NewLayer(strings.Join(lines, "\n")),
		lipgloss.NewLayer(modal).X(x).Y(y).Z(1),
	).Render()
}

func (m model) modalView() string {
	var head, body, foot []string
	border := m.st.c.accent
	if m.stage == confirming {
		head, body, foot = m.confirmLines()
		border = m.st.c.red
	} else {
		head, body, foot = m.progressLines()
	}
	if room := m.height - 4 - len(head) - len(foot); len(body) > room {
		hidden := len(body) - max(room-1, 0)
		body = append(body[:max(room-1, 0)], m.st.muted.Render(fmt.Sprintf("… %d more lines", hidden)))
	}
	lines := append(append(head, body...), foot...)
	widest := 0
	for _, l := range lines {
		widest = max(widest, lipgloss.Width(l))
	}
	width := min(max(widest+6, 44), m.width-4)
	for i := range lines {
		lines[i] = "  " + truncate(lines[i], width-6) + "  "
	}
	return m.box(append([]string{""}, append(lines, "")...), width, len(lines)+4, "", "", border)
}

func (m model) itemNameWidth() int {
	w := 0
	for _, it := range m.items {
		w = max(w, lipgloss.Width(it.wt.Name))
	}
	return min(w, 40)
}

func (m model) confirmLines() (head, body, foot []string) {
	head = []string{m.st.bold.Render(fmt.Sprintf("Delete %s?", count(len(m.items), "worktree", "worktrees"))), ""}
	nameW := m.itemNameWidth()
	var kb int64
	partial := false
	for _, it := range m.items {
		var labels []string
		for _, s := range it.steps {
			labels = append(labels, s.Label)
		}
		body = append(body, m.st.bold.Width(nameW).Render(truncate(it.wt.Name, nameW))+"   "+m.st.muted.Render(strings.Join(labels, " · ")))
		for _, loss := range Losses(it.wt) {
			body = append(body, "    "+m.st.yellow.Render(loss))
		}
		if it.wt.SizeKB > 0 {
			kb += it.wt.SizeKB
		} else {
			partial = true
		}
	}
	frees := humanLong(kb)
	if partial {
		frees += "+"
	}
	foot = []string{
		"",
		"Frees " + m.st.bold.Render(frees) + ". " + m.st.red.Render("Remote branch deletes are permanent."),
		"",
		m.st.red.Bold(true).Render("y") + " delete   " + m.st.bold.Render("n / esc") + m.st.muted.Render(" back"),
	}
	return head, body, foot
}

func (m model) progressLines() (head, body, foot []string) {
	nameW := m.itemNameWidth()
	done, failed, cancelled := 0, 0, 0
	var kb int64
	for _, it := range m.items {
		line := m.itemIcon(it) + " " + m.st.bold.Width(nameW).Render(truncate(it.wt.Name, nameW)) + "  "
		for j, s := range it.steps {
			line += " " + m.stepGlyph(it.states[j]) + " " + m.stepLabel(it.states[j], s.Label)
		}
		body = append(body, line)
		switch {
		case it.cancelled:
			cancelled++
		case it.err != nil:
			failed++
			body = append(body, "    "+m.st.red.Render("✕ "+firstLine(it.err.Error())))
		case it.finished():
			done++
		}
		if it.removed() && it.wt.SizeKB > 0 {
			kb += it.wt.SizeKB
		}
	}
	total := len(m.items)
	if m.stage == deleting {
		title := m.st.bold.Render(fmt.Sprintf("Deleting %d of %d", done+failed, total))
		if m.stopping {
			title = m.st.bold.Render("Stopping after the running worktrees") + m.st.muted.Render(" · ctrl+c again to quit now")
		}
		return []string{m.spinner.View() + " " + title, ""}, body, nil
	}
	summary := m.st.green.Render("✓ ") + m.st.bold.Render("Deleted "+count(done, "worktree", "worktrees"))
	if failed+cancelled > 0 {
		icon := m.st.red.Render("✕ ")
		if failed == 0 {
			icon = m.st.yellow.Render("– ")
		}
		summary = icon + m.st.bold.Render(fmt.Sprintf("Deleted %d of %d worktrees", done, total))
		if failed > 0 {
			summary += m.st.red.Render(fmt.Sprintf(" · %d failed", failed))
		}
		if cancelled > 0 {
			summary += m.st.yellow.Render(fmt.Sprintf(" · %d not started", cancelled))
		}
	}
	head = []string{summary + m.st.muted.Render(" · freed "+humanLong(kb)), ""}
	foot = []string{"", m.st.muted.Render("q quit · any other key to continue")}
	return head, body, foot
}

func (m model) itemIcon(it deleteItem) string {
	switch {
	case it.cancelled:
		return m.st.faint.Render("–")
	case it.err != nil:
		return m.st.red.Render("✕")
	case it.finished():
		return m.st.green.Render("✓")
	case it.states[0] != stepQueued:
		return m.spinner.View()
	}
	return m.st.muted.Render("·")
}

func (m model) stepGlyph(s stepState) string {
	switch s {
	case stepDone:
		return m.st.green.Render("✓")
	case stepRunning:
		return m.spinner.View()
	case stepFailed:
		return m.st.red.Render("✕")
	case stepSkipped:
		return m.st.faint.Render("–")
	}
	return m.st.muted.Render("·")
}

func (m model) stepLabel(s stepState, label string) string {
	switch s {
	case stepQueued:
		return m.st.muted.Render(label)
	case stepSkipped:
		return m.st.faint.Render(label)
	}
	return label
}

func (m model) progress() int {
	total, done := 0, 0
	for _, it := range m.items {
		for _, s := range it.states {
			total++
			if s != stepQueued && s != stepRunning {
				done++
			}
		}
	}
	if total == 0 {
		return 0
	}
	return done * 100 / total
}

func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	return ansi.Truncate(s, width, "…")
}

func age(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	d := time.Since(t)
	switch {
	case d >= 24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	case d >= time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d >= time.Minute:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return "now"
}

func relTime(t time.Time) string {
	if t.IsZero() {
		return "running"
	}
	if a := age(t); a != "now" {
		return a + " ago"
	}
	return "just now"
}

func sizeText(kb int64) string {
	switch {
	case kb < 0:
		return "…"
	case kb == 0:
		return "?"
	}
	return humanShort(kb)
}

func humanShort(kb int64) string {
	v := float64(kb)
	for _, unit := range []string{"K", "M", "G"} {
		if v < 1000 {
			if v < 10 && unit != "K" {
				return fmt.Sprintf("%.1f%s", v, unit)
			}
			return fmt.Sprintf("%.0f%s", v, unit)
		}
		v /= 1024
	}
	return fmt.Sprintf("%.1fT", v)
}

func humanLong(kb int64) string {
	v := float64(kb)
	for _, unit := range []string{"KB", "MB", "GB"} {
		if v < 1000 {
			if v < 10 && unit != "KB" {
				return fmt.Sprintf("%.1f %s", v, unit)
			}
			return fmt.Sprintf("%.0f %s", v, unit)
		}
		v /= 1024
	}
	return fmt.Sprintf("%.1f TB", v)
}
