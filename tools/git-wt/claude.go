package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

var pidAlive = func(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

var pathEncoder = strings.NewReplacer("/", "-", ".", "-")

type transcript struct {
	id            string
	custom, ai    string
	lastByCwd     map[string]time.Time
	latestCwd     string
	latestCwdTime time.Time
}

func (t *transcript) addCwd(cwd string, ts time.Time) {
	if prev, ok := t.lastByCwd[cwd]; !ok || ts.After(prev) {
		t.lastByCwd[cwd] = ts
	}
	if t.latestCwd == "" || !ts.Before(t.latestCwdTime) {
		t.latestCwd, t.latestCwdTime = cwd, ts
	}
}

func (t *transcript) merge(o *transcript) {
	for cwd, ts := range o.lastByCwd {
		t.addCwd(cwd, ts)
	}
	if o.custom != "" {
		t.custom = o.custom
	}
	if o.ai != "" {
		t.ai = o.ai
	}
}

func (t *transcript) title() string {
	custom, ai := oneLine(t.custom), oneLine(t.ai)
	switch {
	case custom != "":
		return custom
	case ai != "":
		return ai
	case len(t.id) > 8:
		return t.id[:8]
	}
	return t.id
}

var errSessionsUnreadable = errors.New("Claude session data unreadable")

// LoadSessions returns Claude Code sessions per worktree path, newest first.
func LoadSessions(claudeDir string, ws []Worktree) (map[string][]Session, error) {
	out := map[string][]Session{}
	attr := newAttributor(ws)
	files := transcriptFiles(filepath.Join(claudeDir, "projects"), ws)

	results := make(chan *transcript)
	jobs := make(chan string)
	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range jobs {
				if t := parseTranscript(f); t != nil {
					results <- t
				}
			}
		}()
	}
	go func() {
		for _, f := range files {
			jobs <- f
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	byID := map[string]*transcript{}
	readable := 0
	for t := range results {
		if len(t.lastByCwd) > 0 {
			readable++
		}
		if prev, ok := byID[t.id]; ok {
			prev.merge(t)
		} else {
			byID[t.id] = t
		}
	}

	live, liveOK := liveSessions(claudeDir)
	liveAt := map[string]string{}
	for id, startCwd := range live {
		cwd := startCwd
		if t, ok := byID[id]; ok && t.latestCwd != "" {
			cwd = t.latestCwd
		}
		if p, ok := attr.worktree(cwd); ok {
			liveAt[id] = p
		}
	}

	for id, t := range byID {
		last := map[string]time.Time{}
		for cwd, ts := range t.lastByCwd {
			if p, ok := attr.worktree(cwd); ok && (last[p].IsZero() || ts.After(last[p])) {
				last[p] = ts
			}
		}
		if p, ok := liveAt[id]; ok {
			if _, seen := last[p]; !seen {
				last[p] = time.Time{}
			}
		}
		for p, ts := range last {
			out[p] = append(out[p], Session{ID: id, Title: t.title(), Last: ts, Live: liveAt[id] == p})
		}
	}
	for id, p := range liveAt {
		if _, ok := byID[id]; !ok {
			out[p] = append(out[p], Session{ID: id, Title: shortID(id), Live: true})
		}
	}
	for _, s := range out {
		sort.SliceStable(s, func(i, j int) bool { return s[i].Last.After(s[j].Last) })
	}
	if (len(files) > 0 && readable == 0) || !liveOK {
		return out, errSessionsUnreadable
	}
	return out, nil
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

type attributor struct{ paths []string }

func newAttributor(ws []Worktree) attributor {
	var a attributor
	for _, w := range ws {
		a.paths = append(a.paths, strings.TrimRight(w.Path, "/"))
	}
	sort.Slice(a.paths, func(i, j int) bool { return len(a.paths[i]) > len(a.paths[j]) })
	return a
}

func (a attributor) worktree(cwd string) (string, bool) {
	for _, p := range a.paths {
		if cwd == p || strings.HasPrefix(cwd, p+"/") {
			return p, true
		}
	}
	return "", false
}

func transcriptFiles(projects string, ws []Worktree) []string {
	entries, err := os.ReadDir(projects)
	if err != nil {
		return nil
	}
	exact := map[string]bool{}
	mainPrefix := ""
	for _, w := range ws {
		exact[pathEncoder.Replace(w.Path)] = true
		if w.Main {
			mainPrefix = pathEncoder.Replace(w.Path)
		}
	}
	var files []string
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() || !(exact[name] || (mainPrefix != "" && strings.HasPrefix(name, mainPrefix))) {
			continue
		}
		matches, _ := filepath.Glob(filepath.Join(projects, name, "*.jsonl"))
		files = append(files, matches...)
	}
	return files
}

func parseTranscript(path string) *transcript {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	t := &transcript{
		id:        strings.TrimSuffix(filepath.Base(path), ".jsonl"),
		lastByCwd: map[string]time.Time{},
	}
	r := bufio.NewReaderSize(f, 1<<20)
	var line []byte
	for {
		line = line[:0]
		var readErr error
		for {
			frag, err := r.ReadSlice('\n')
			line = append(line, frag...)
			if err != bufio.ErrBufferFull {
				readErr = err
				break
			}
		}
		t.scanLine(line)
		if readErr != nil {
			if readErr != io.EOF {
				return nil
			}
			break
		}
	}
	return t
}

var (
	keyCwd       = []byte(`"cwd":"`)
	keyTimestamp = []byte(`"timestamp":"`)
	keyCustom    = []byte(`"customTitle":"`)
	keyAI        = []byte(`"aiTitle":"`)
	keySession   = []byte(`"sessionId":"`)
)

func (t *transcript) scanLine(line []byte) {
	if id, ok := stringField(line, keySession); ok && id != "" {
		t.id = id
	}
	if s, ok := stringField(line, keyCustom); ok && s != "" {
		t.custom = s
	}
	if s, ok := stringField(line, keyAI); ok && s != "" {
		t.ai = s
	}
	cwd, ok := stringField(line, keyCwd)
	if !ok || cwd == "" {
		return
	}
	var ts time.Time
	if s, ok := stringField(line, keyTimestamp); ok {
		ts, _ = time.Parse(time.RFC3339Nano, s)
	}
	t.addCwd(cwd, ts)
}

func stringField(line, key []byte) (string, bool) {
	i := bytes.Index(line, key)
	if i < 0 {
		return "", false
	}
	rest := line[i+len(key):]
	escaped := false
	for j := 0; j < len(rest); j++ {
		switch rest[j] {
		case '\\':
			escaped = true
			j++
		case '"':
			raw := rest[:j]
			if !escaped {
				return string(raw), true
			}
			var s string
			quoted := make([]byte, 0, j+2)
			quoted = append(append(append(quoted, '"'), raw...), '"')
			if err := json.Unmarshal(quoted, &s); err != nil {
				return "", false
			}
			return s, true
		}
	}
	return "", false
}

func liveSessions(claudeDir string) (map[string]string, bool) {
	live := map[string]string{}
	files, _ := filepath.Glob(filepath.Join(claudeDir, "sessions", "*.json"))
	parsed := 0
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var s struct {
			Pid       int    `json:"pid"`
			SessionID string `json:"sessionId"`
			Cwd       string `json:"cwd"`
		}
		if json.Unmarshal(data, &s) != nil || s.SessionID == "" || s.Pid <= 0 {
			continue
		}
		parsed++
		if pidAlive(s.Pid) {
			live[s.SessionID] = s.Cwd
		}
	}
	return live, len(files) == 0 || parsed > 0
}
