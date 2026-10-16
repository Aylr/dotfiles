package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/tabwriter"
	"time"
)

func main() {
	dump := flag.Bool("dump", false, "print one tab-separated line per worktree and exit")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: git wt [--dump]\n\nReview and delete the worktrees of the current repo.\nKeys: space mark · d diff · / filter · x delete marked · ? help · q quit")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() > 0 {
		flag.Usage()
		os.Exit(2)
	}

	cwd, _ := os.Getwd()
	r, err := DetectRepo(cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "git wt:", err)
		os.Exit(1)
	}
	if *dump {
		err = runDump(r)
	} else {
		err = runTUI(r)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "git wt:", err)
		os.Exit(1)
	}
}

func claudeDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}

func loadAll(r Repo) ([]Worktree, error) {
	ws, err := ListWorktrees(r)
	if err != nil {
		return nil, err
	}
	_ = Fetch(r)
	prs, prErr := prSource(r.Root, PRHeads(ws, r.DefaultBranch))
	for i := range ws {
		if prErr == nil {
			ws[i].PR = MatchPR(prs, ws[i])
		}
	}
	Collect(r, ws)
	sessions := LoadSessions(claudeDir(), ws)
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	for i := range ws {
		ws[i].Sessions = sessions[ws[i].Path]
		wg.Add(1)
		go func(w *Worktree) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if kb, err := DirSizeKB(w.Path); err == nil {
				w.SizeKB = kb
			}
		}(&ws[i])
	}
	wg.Wait()
	if prErr != nil {
		fmt.Fprintln(os.Stderr, "gh:", prErr)
	}
	return ws, nil
}

func runDump(r Repo) error {
	ws, err := loadAll(r)
	if err != nil {
		return err
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tAGE\tDIRTY/UNTR/LOST\tAHEAD/BEHIND\tPR\tSIZE\tCLAUDE\tLIVE\tVERDICT")
	for _, w := range ws {
		pr := w.PR.Word()
		if w.PR != nil && w.PR.State != "OPEN" {
			pr = fmt.Sprintf("%s #%d", pr, w.PR.Number)
		}
		title, live := "", false
		for _, s := range w.Sessions {
			live = live || s.Live
		}
		if len(w.Sessions) > 0 {
			title = w.Sessions[0].Title
		}
		verdict := "keep"
		if reasons := BlockedReasons(w, r.DefaultBranch); len(reasons) > 0 {
			verdict = "blocked: " + strings.Join(reasons, ", ")
		} else if PreMark(w, r.DefaultBranch) {
			verdict = "pre-marked"
		}
		fmt.Fprintf(tw, "%s\t%s\t%d/%d/%d\t+%d/-%d\t%s\t%dM\t%s\t%v\t%s\n",
			w.Name, dumpAge(w.LastActivity), len(w.DirtyFiles), len(w.Untracked), w.Lost,
			w.Ahead, w.Behind, pr, w.SizeKB/1024, title, live, verdict)
	}
	return tw.Flush()
}

func dumpAge(t time.Time) string {
	d := time.Since(t)
	if d >= 24*time.Hour {
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}
