package main

import (
	"errors"
	"os"
	"os/exec"
	"slices"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

type stage int

const (
	browsing stage = iota
	confirming
	deleting
	results
)

const (
	sizeWorkers = 4
	detailDelay = 120 * time.Millisecond
	doneFade    = 2 * time.Second
)

type sources struct {
	list      func(Repo) ([]Worktree, error)
	fetch     func(Repo) error
	collect   func(Repo, []Worktree)
	prs       func(root string, heads []string) ([]PR, error)
	sessions  func([]Worktree) map[string][]Session
	size      func(path string) (int64, error)
	detail    func(Repo, Worktree) (Detail, error)
	mergeBase func(Repo, Worktree) (string, error)
	steps     func(Worktree) []Step
}

func realSources() sources {
	return sources{
		list:      ListWorktrees,
		fetch:     Fetch,
		collect:   Collect,
		prs:       func(root string, heads []string) ([]PR, error) { return prSource(root, heads) },
		sessions:  func(ws []Worktree) map[string][]Session { return LoadSessions(claudeDir(), ws) },
		size:      DirSizeKB,
		detail:    LoadDetail,
		mergeBase: MergeBase,
		steps:     DeleteSteps,
	}
}

type (
	fetchDoneMsg struct {
		gen int
		err error
	}
	collectedMsg struct {
		gen                 int
		ws                  []Worktree
		afterFetch, withPRs bool
	}
	prsLoadedMsg struct {
		gen int
		prs []PR
		err error
	}
	sessionsLoadedMsg struct {
		gen      int
		sessions map[string][]Session
	}
	sizeMsg struct {
		gen  int
		path string
		kb   int64
		err  error
	}
	listedMsg struct {
		ws  []Worktree
		err error
	}
	detailDueMsg struct{ path string }
	detailMsg    struct {
		gen    int
		path   string
		detail Detail
		err    error
	}
	diffReadyMsg struct {
		path, base, pager string
		err               error
	}
	diffDoneMsg struct{ err error }
	stepMsg     struct {
		item, step int
		err        error
	}
	fadeMsg struct{ gen int }
)

type detailResult struct {
	gen    int
	detail Detail
	err    error
}

type stepState int

const (
	stepQueued stepState = iota
	stepRunning
	stepDone
	stepFailed
	stepSkipped
)

type deleteItem struct {
	wt        Worktree
	steps     []Step
	states    []stepState
	err       error
	cancelled bool
}

func (d deleteItem) removed() bool { return len(d.states) > 0 && d.states[0] == stepDone }

func (d deleteItem) finished() bool {
	return !d.cancelled && (d.err != nil || len(d.states) == 0 || d.states[len(d.states)-1] == stepDone)
}

type model struct {
	repo Repo
	src  sources

	all            []Worktree
	visible        []int
	cursor, offset int
	cursorPath     string
	toggled        map[string]bool

	filter    textinput.Model
	filtering bool

	detail        viewport.Model
	detailPath    string
	details       map[string]detailResult
	detailLoading map[string]bool
	detailGen     int

	gen                 int
	pending             map[string]bool
	fetched, prsApplied bool
	prsCollected        bool
	sessionsLoaded      bool
	premarked           bool
	collecting          bool
	recollect           bool
	ghFailed            bool
	ghErr               string
	ghWarned            bool
	sizeQueue           []string
	sizesInFlight       int
	sizesDone           int
	sizesTotal          int
	loadStart           time.Time
	loadedIn            time.Duration
	fadeGen             int

	items              []deleteItem
	nextItem, inFlight int
	stopping           bool
	boot               tea.Cmd

	spinner       spinner.Model
	help          help.Model
	keys          keyMap
	status        string
	width, height int
	stage         stage
	isDark        bool
	st            styles
}

func runTUI(r Repo) error {
	ws, err := ListWorktrees(r)
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(newModel(r, ws, realSources())).Run()
	return err
}

func newModel(r Repo, ws []Worktree, src sources) model {
	m := model{
		repo:          r,
		src:           src,
		all:           ws,
		toggled:       map[string]bool{},
		details:       map[string]detailResult{},
		detailLoading: map[string]bool{},
		pending:       map[string]bool{},
		spinner:       spinner.New(spinner.WithSpinner(spinner.Spinner{Frames: strings.Split("⣾⣽⣻⢿⡿⣟⣯⣷", ""), FPS: time.Second / 12})),
		help:          help.New(),
		keys:          newKeyMap(),
		filter:        textinput.New(),
		detail:        viewport.New(),
		isDark:        true,
	}
	m.filter.Prompt = "/ "
	m.filter.Placeholder = "name, branch, PR or session"
	m.help.ShortSeparator = " · "
	m.applyTheme()
	m.rebuild()
	m.boot = m.startLoads()
	return m
}

func (m *model) applyTheme() {
	m.st = newStyles(m.isDark)
	m.help.Styles = help.Styles{
		ShortKey:       m.st.bold,
		ShortDesc:      m.st.muted,
		ShortSeparator: m.st.faint,
		Ellipsis:       m.st.faint,
		FullKey:        m.st.bold,
		FullDesc:       m.st.muted,
		FullSeparator:  m.st.faint,
	}
	fs := textinput.DefaultStyles(m.isDark)
	fs.Focused.Prompt = m.st.accent
	fs.Focused.Placeholder = m.st.muted
	fs.Blurred.Placeholder = m.st.muted
	m.filter.SetStyles(fs)
	m.spinner.Style = m.st.accent
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tea.RequestBackgroundColor, m.boot)
}

func (m *model) startLoads() tea.Cmd {
	m.gen++
	m.loadStart = time.Now()
	m.loadedIn = 0
	m.fetched, m.prsApplied, m.prsCollected, m.sessionsLoaded = false, false, false, false
	m.collecting, m.recollect, m.ghFailed = false, false, false
	m.pending = map[string]bool{"fetch": true, "prs": true, "sessions": true}
	m.sizeQueue, m.sizesInFlight, m.sizesDone = nil, 0, 0
	for _, w := range m.all {
		if w.SizeKB < 0 && !w.Main {
			m.sizeQueue = append(m.sizeQueue, w.Path)
		}
	}
	m.sizesTotal = len(m.sizeQueue)
	if m.sizesTotal > 0 {
		m.pending["sizes"] = true
	}

	r, src, gen := m.repo, m.src, m.gen
	ws := slices.Clone(m.all)
	heads := PRHeads(ws, r.DefaultBranch)
	return tea.Batch(
		func() tea.Msg { return fetchDoneMsg{gen, src.fetch(r)} },
		func() tea.Msg {
			prs, err := src.prs(r.Root, heads)
			return prsLoadedMsg{gen, prs, err}
		},
		func() tea.Msg { return sessionsLoadedMsg{gen, src.sessions(ws)} },
		m.nextSizeCmds(),
		m.spinner.Tick,
	)
}

func (m *model) nextSizeCmds() tea.Cmd {
	var cmds []tea.Cmd
	for m.sizesInFlight < sizeWorkers && len(m.sizeQueue) > 0 {
		path := m.sizeQueue[0]
		m.sizeQueue = m.sizeQueue[1:]
		m.sizesInFlight++
		size, gen := m.src.size, m.gen
		cmds = append(cmds, func() tea.Msg {
			kb, err := size(path)
			return sizeMsg{gen, path, kb, err}
		})
	}
	return tea.Batch(cmds...)
}

func (m *model) collectCmd() tea.Cmd {
	if m.collecting {
		m.recollect = true
		return nil
	}
	m.collecting = true
	ws := slices.Clone(m.all)
	r, collect, gen := m.repo, m.src.collect, m.gen
	afterFetch, withPRs := m.fetched, m.prsApplied
	return func() tea.Msg {
		collect(r, ws)
		return collectedMsg{gen, ws, afterFetch, withPRs}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmd := m.update(msg)
	m.rebuild()
	return m, tea.Batch(cmd, m.syncDetail())
}

func (m *model) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.BackgroundColorMsg:
		m.isDark = msg.IsDark()
		m.applyTheme()
	case spinner.TickMsg:
		if !m.busy() {
			return nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return cmd
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	case fadeMsg:
		if msg.gen == m.fadeGen {
			m.loadedIn = 0
		}
	case fetchDoneMsg:
		if msg.gen != m.gen {
			return nil
		}
		m.fetched = true
		if msg.err != nil {
			if m.status == "" {
				m.status = "✕ fetch: " + msg.err.Error()
			}
			return m.collectCmd()
		}
		m.detailGen++
		m.detailLoading = map[string]bool{}
		return tea.Batch(m.collectCmd(), m.loadDetail(m.cursorPath))
	case prsLoadedMsg:
		if msg.gen != m.gen {
			return nil
		}
		return m.applyPRs(msg.prs, msg.err)
	case collectedMsg:
		if msg.gen != m.gen {
			return nil
		}
		return m.mergeCollected(msg)
	case sessionsLoadedMsg:
		if msg.gen != m.gen {
			return nil
		}
		for i := range m.all {
			m.all[i].Sessions = msg.sessions[m.all[i].Path]
		}
		m.sessionsLoaded = true
		delete(m.pending, "sessions")
		m.maybePremark()
		return m.checkLoaded()
	case sizeMsg:
		return m.applySize(msg)
	case listedMsg:
		return m.applyListed(msg)
	case detailDueMsg:
		return m.loadDetail(msg.path)
	case detailMsg:
		if msg.gen != m.detailGen {
			return nil
		}
		delete(m.detailLoading, msg.path)
		m.details[msg.path] = detailResult{msg.gen, msg.detail, msg.err}
	case diffReadyMsg:
		if msg.err != nil {
			m.status = "✕ diff: " + msg.err.Error()
			return nil
		}
		return execDiff(msg)
	case diffDoneMsg:
		if msg.err != nil {
			m.status = "✕ diff: " + msg.err.Error()
		}
	case stepMsg:
		return m.applyStep(msg)
	}
	return nil
}

func (m *model) applyPRs(prs []PR, err error) tea.Cmd {
	m.prsApplied = true
	if err != nil {
		m.ghFailed = true
		m.ghErr = ghReason(err)
		m.prsCollected = true
		delete(m.pending, "prs")
		if !m.ghWarned {
			m.ghWarned = true
			m.status = "✕ PRs unavailable · " + m.ghErr
		}
		m.maybePremark()
		return m.checkLoaded()
	}
	for i := range m.all {
		m.all[i].PR = MatchPR(prs, m.all[i])
	}
	return m.collectCmd()
}

func (m *model) mergeCollected(msg collectedMsg) tea.Cmd {
	byPath := make(map[string]Worktree, len(msg.ws))
	for _, w := range msg.ws {
		byPath[w.Path] = w
	}
	for i := range m.all {
		if w, ok := byPath[m.all[i].Path]; ok {
			mergeGitFacts(&m.all[i], w)
		}
	}
	m.collecting = false
	if msg.afterFetch {
		delete(m.pending, "fetch")
	}
	if msg.withPRs {
		m.prsCollected = true
		delete(m.pending, "prs")
	}
	m.maybePremark()
	var cmd tea.Cmd
	if m.recollect {
		m.recollect = false
		cmd = m.collectCmd()
	}
	return tea.Batch(cmd, m.checkLoaded())
}

func mergeGitFacts(dst *Worktree, src Worktree) {
	dst.Upstream = src.Upstream
	dst.DirtyFiles, dst.Untracked = src.DirtyFiles, src.Untracked
	dst.Lost, dst.Ahead, dst.Behind = src.Lost, src.Ahead, src.Behind
	dst.Orphaned = src.Orphaned
	dst.LastActivity = src.LastActivity
}

func (m *model) applySize(msg sizeMsg) tea.Cmd {
	if i := m.indexOf(msg.path); i >= 0 {
		m.all[i].SizeKB = msg.kb
		if msg.err != nil {
			m.all[i].SizeKB = 0
		}
	}
	if msg.gen != m.gen {
		return nil
	}
	m.sizesInFlight--
	m.sizesDone++
	if m.sizesDone >= m.sizesTotal {
		delete(m.pending, "sizes")
	}
	return tea.Batch(m.nextSizeCmds(), m.checkLoaded())
}

func (m *model) maybePremark() {
	if m.premarked || !m.prsCollected || !m.sessionsLoaded {
		return
	}
	m.premarked = true
	for i := range m.all {
		if w := &m.all[i]; !m.toggled[w.Path] {
			w.Marked = PreMark(*w, m.repo.DefaultBranch)
		}
	}
}

func (m *model) checkLoaded() tea.Cmd {
	if len(m.pending) > 0 || m.loadStart.IsZero() {
		return nil
	}
	m.loadedIn = max(time.Since(m.loadStart), time.Millisecond)
	m.loadStart = time.Time{}
	m.fadeGen++
	gen := m.fadeGen
	return tea.Tick(doneFade, func(time.Time) tea.Msg { return fadeMsg{gen} })
}

func (m model) busy() bool {
	return len(m.pending) > 0 || m.stage == deleting || m.detailLoading[m.cursorPath]
}

func (m *model) handleKey(msg tea.KeyPressMsg) tea.Cmd {
	if m.stage == deleting {
		if key.Matches(msg, m.keys.ForceQuit) && m.stopping {
			return tea.Quit
		}
		if key.Matches(msg, m.keys.Quit) {
			m.stopping = true
		}
		return nil
	}
	if key.Matches(msg, m.keys.ForceQuit) {
		return tea.Quit
	}
	m.status = ""
	switch m.stage {
	case confirming:
		return m.confirmKey(msg)
	case results:
		if key.Matches(msg, m.keys.Quit) {
			return tea.Quit
		}
		return m.reload()
	}
	if m.filtering {
		return m.filterKey(msg)
	}
	switch {
	case key.Matches(msg, m.keys.Quit):
		return tea.Quit
	case key.Matches(msg, m.keys.Up):
		m.moveCursor(-1)
	case key.Matches(msg, m.keys.Down):
		m.moveCursor(1)
	case key.Matches(msg, m.keys.Top):
		m.moveCursor(-len(m.all))
	case key.Matches(msg, m.keys.Bottom):
		m.moveCursor(len(m.all))
	case key.Matches(msg, m.keys.PageUp):
		m.detail.PageUp()
	case key.Matches(msg, m.keys.PageDown):
		m.detail.PageDown()
	case key.Matches(msg, m.keys.Mark):
		m.toggleMark()
	case key.Matches(msg, m.keys.Diff):
		return m.diff()
	case key.Matches(msg, m.keys.Filter):
		m.filtering = true
		return m.filter.Focus()
	case key.Matches(msg, m.keys.Clear):
		m.filter.SetValue("")
	case key.Matches(msg, m.keys.Delete):
		m.startConfirm()
	case key.Matches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
	}
	return nil
}

func (m *model) filterKey(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Clear):
		m.filter.SetValue("")
		m.filtering = false
		m.filter.Blur()
		return nil
	case key.Matches(msg, m.keys.Accept):
		m.filtering = false
		m.filter.Blur()
		return nil
	case msg.String() == "up":
		m.moveCursor(-1)
		return nil
	case msg.String() == "down":
		m.moveCursor(1)
		return nil
	}
	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	return cmd
}

func (m *model) confirmKey(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Confirm):
		m.stage = deleting
		m.nextItem, m.inFlight = 0, 0
		return tea.Batch(m.startDeletes(), m.spinner.Tick)
	case key.Matches(msg, m.keys.Back):
		m.stage = browsing
		m.items = nil
	}
	return nil
}

func (m *model) moveCursor(delta int) {
	m.cursor = min(max(m.cursor+delta, 0), max(len(m.visible)-1, 0))
	if w := m.current(); w != nil {
		m.cursorPath = w.Path
	}
}

func (m *model) current() *Worktree {
	if m.cursor < 0 || m.cursor >= len(m.visible) {
		return nil
	}
	return &m.all[m.visible[m.cursor]]
}

func (m *model) indexOf(path string) int {
	return slices.IndexFunc(m.all, func(w Worktree) bool { return w.Path == path })
}

func (m *model) toggleMark() {
	w := m.current()
	if w == nil {
		return
	}
	if reasons := BlockedReasons(*w, m.repo.DefaultBranch); len(reasons) > 0 {
		m.status = "✕ " + reasons[0]
		return
	}
	w.Marked = !w.Marked
	m.toggled[w.Path] = true
}

func (m *model) startConfirm() {
	if waiting := m.loadingLabels("list", "fetch", "prs", "sessions"); len(waiting) > 0 {
		m.status = "Still checking " + strings.Join(waiting, " · ") + " · press x again in a moment"
		return
	}
	var items []deleteItem
	for i := range m.all {
		w := &m.all[i]
		if !w.Marked {
			continue
		}
		if len(BlockedReasons(*w, m.repo.DefaultBranch)) > 0 {
			w.Marked = false
			continue
		}
		steps := m.src.steps(*w)
		items = append(items, deleteItem{wt: *w, steps: steps, states: make([]stepState, len(steps))})
	}
	if len(items) == 0 {
		m.status = "Nothing marked · space marks a row"
		return
	}
	m.items = items
	m.stage = confirming
}

var deleteWorkers = 6

func (m *model) startDeletes() tea.Cmd {
	var cmds []tea.Cmd
	for m.nextItem < len(m.items) && m.inFlight < deleteWorkers && !m.stopping {
		i := m.nextItem
		m.nextItem++
		if len(m.items[i].steps) == 0 {
			continue
		}
		m.inFlight++
		cmds = append(cmds, m.runStep(i, 0))
	}
	if m.stopping {
		for ; m.nextItem < len(m.items); m.nextItem++ {
			it := &m.items[m.nextItem]
			it.cancelled = true
			for k := range it.states {
				it.states[k] = stepSkipped
			}
		}
	}
	if m.inFlight == 0 && m.nextItem >= len(m.items) {
		m.stage = results
		m.stopping = false
		return nil
	}
	return tea.Batch(cmds...)
}

func (m *model) runStep(item, step int) tea.Cmd {
	m.items[item].states[step] = stepRunning
	s, r := m.items[item].steps[step], m.repo
	return func() tea.Msg { return stepMsg{item, step, s.Run(r)} }
}

func (m *model) applyStep(msg stepMsg) tea.Cmd {
	it := &m.items[msg.item]
	if msg.err == nil {
		it.states[msg.step] = stepDone
		if msg.step+1 < len(it.steps) {
			return m.runStep(msg.item, msg.step+1)
		}
	} else {
		it.states[msg.step] = stepFailed
		it.err = msg.err
		for j := msg.step + 1; j < len(it.states); j++ {
			it.states[j] = stepSkipped
		}
	}
	m.inFlight--
	return m.startDeletes()
}

func (m *model) reload() tea.Cmd {
	removed := map[string]bool{}
	for _, it := range m.items {
		if it.removed() {
			removed[it.wt.Path] = true
		}
	}
	m.all = slices.DeleteFunc(m.all, func(w Worktree) bool { return removed[w.Path] })
	m.items = nil
	m.stage = browsing
	m.pending["list"] = true
	r, list := m.repo, m.src.list
	return tea.Batch(func() tea.Msg {
		ws, err := list(r)
		return listedMsg{ws, err}
	}, m.spinner.Tick)
}

func (m *model) applyListed(msg listedMsg) tea.Cmd {
	delete(m.pending, "list")
	if msg.err != nil {
		m.status = "✕ reload: " + msg.err.Error()
		return nil
	}
	old := make(map[string]Worktree, len(m.all))
	for _, w := range m.all {
		old[w.Path] = w
	}
	decided := map[string]bool{}
	for i := range msg.ws {
		w := &msg.ws[i]
		o, ok := old[w.Path]
		if !ok {
			continue
		}
		w.Marked, w.PR, w.Sessions = o.Marked, o.PR, o.Sessions
		if w.SizeKB < 0 {
			w.SizeKB = o.SizeKB
		}
		if m.premarked || m.toggled[w.Path] {
			decided[w.Path] = true
		}
	}
	m.all = msg.ws
	m.toggled = decided
	m.premarked = false
	m.details = map[string]detailResult{}
	m.detailLoading = map[string]bool{}
	m.detailGen++
	m.detailPath = ""
	return m.startLoads()
}

func (m *model) rebuild() {
	sort.SliceStable(m.all, func(i, j int) bool {
		a, b := m.all[i], m.all[j]
		if a.Main != b.Main {
			return b.Main
		}
		return a.LastActivity.Before(b.LastActivity)
	})
	q := strings.ToLower(strings.TrimSpace(m.filter.Value()))
	m.visible = make([]int, 0, len(m.all))
	for i, w := range m.all {
		if w.Marked && len(BlockedReasons(w, m.repo.DefaultBranch)) > 0 {
			m.all[i].Marked = false
		}
		if q == "" || matches(w, q) {
			m.visible = append(m.visible, i)
		}
	}
	m.cursor = slices.IndexFunc(m.visible, func(i int) bool { return m.all[i].Path == m.cursorPath })
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.moveCursor(0)
	rows := m.layout().tableRows
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if rows > 0 && m.cursor >= m.offset+rows {
		m.offset = m.cursor - rows + 1
	}
	m.offset = min(m.offset, max(len(m.visible)-rows, 0))
	m.offset = max(m.offset, 0)
	m.sizeDetail()
}

func matches(w Worktree, q string) bool {
	fields := []string{w.Name, w.Branch}
	if w.PR != nil {
		fields = append(fields, w.PR.Title)
	}
	for _, s := range w.Sessions {
		fields = append(fields, s.Title)
	}
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), q) {
			return true
		}
	}
	return false
}

func (m *model) syncDetail() tea.Cmd {
	if m.cursorPath == m.detailPath {
		m.setDetailContent()
		return nil
	}
	m.detailPath = m.cursorPath
	m.setDetailContent()
	m.detail.GotoTop()
	if m.detailFresh(m.cursorPath) || m.cursorPath == "" {
		return nil
	}
	path := m.cursorPath
	return tea.Tick(detailDelay, func(time.Time) tea.Msg { return detailDueMsg{path} })
}

func (m *model) detailFresh(path string) bool {
	res, ok := m.details[path]
	return ok && res.gen == m.detailGen
}

func (m *model) loadDetail(path string) tea.Cmd {
	i := m.indexOf(path)
	if path == "" || path != m.cursorPath || m.detailFresh(path) || m.detailLoading[path] || i < 0 {
		return nil
	}
	m.detailLoading[path] = true
	r, detail, w, gen := m.repo, m.src.detail, m.all[i], m.detailGen
	return tea.Batch(func() tea.Msg {
		d, err := detail(r, w)
		return detailMsg{gen, path, d, err}
	}, m.spinner.Tick)
}

func (m *model) diff() tea.Cmd {
	w := m.current()
	if w == nil {
		return nil
	}
	r, wt, mergeBase := m.repo, *w, m.src.mergeBase
	known := ""
	if m.detailFresh(w.Path) {
		known = m.details[w.Path].detail.MergeBase
	}
	return func() tea.Msg {
		msg := diffReadyMsg{path: wt.Path, base: known}
		if _, err := os.Stat(wt.Path); err != nil {
			msg.err = errors.New("worktree directory is gone")
			return msg
		}
		if msg.base == "" {
			msg.base, msg.err = mergeBase(r, wt)
		}
		msg.pager, _ = runGit(r.Root, "config", "wt.pager")
		return msg
	}
}

func execDiff(msg diffReadyMsg) tea.Cmd {
	c := exec.Command("git", "-C", msg.path, "--paginate", "diff", msg.base)
	if msg.pager != "" {
		c.Env = append(os.Environ(), "GIT_PAGER="+msg.pager)
	}
	return tea.ExecProcess(c, func(err error) tea.Msg { return diffDoneMsg{err} })
}

func (m model) loadingLabels(keys ...string) []string {
	var labels []string
	for _, k := range keys {
		if m.pending[k] {
			labels = append(labels, pendingLabel[k])
		}
	}
	return labels
}

var pendingLabel = map[string]string{"list": "worktrees", "fetch": "origin", "prs": "PRs", "sessions": "sessions"}

func ghReason(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "none of the git remotes"):
		return "no GitHub remote"
	case strings.Contains(msg, "gh auth login"):
		return "gh is not logged in · run gh auth login"
	}
	return msg
}
