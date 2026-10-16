package main

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"
	"unicode"
)

const cmdTimeout = 60 * time.Second

func run(dir, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if line := firstLine(stderr.String()); line != "" {
			return stdout.String(), errors.New(line)
		}
		return stdout.String(), err
	}
	return stdout.String(), nil
}

func runGit(dir string, args ...string) (string, error) {
	out, err := run(dir, "git", args...)
	return strings.TrimRight(out, "\n"), err
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimRight(strings.TrimPrefix(s, "fatal: "), ";:")
}

func oneLine(s string) string {
	return strings.Join(strings.FieldsFunc(s, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }), " ")
}
