package main

import (
	"strings"
	"testing"
)

const ghFixture = `{"data":{"repository":{
 "h0":{"nodes":[{"number":618,"title":"Add pfe contract","state":"OPEN","isDraft":false,"headRefName":"feat/pfe-contract","headRefOid":"abc123","url":"https://github.com/o/r/pull/618","reviewDecision":"APPROVED"}]},
 "h1":{"nodes":[{"number":744,"title":"WIP thing","state":"OPEN","isDraft":true,"headRefName":"wip/thing","headRefOid":"def456","url":"https://github.com/o/r/pull/744","reviewDecision":""}]},
 "h2":{"nodes":[]}
}}}`

func TestParsePRs(t *testing.T) {
	prs, err := parsePRs([]byte(ghFixture))
	if err != nil {
		t.Fatal(err)
	}
	want := PR{Number: 744, Title: "WIP thing", URL: "https://github.com/o/r/pull/744", State: "OPEN", Draft: true, HeadRef: "wip/thing", HeadOid: "def456"}
	if len(prs) != 2 || prs[1] != want {
		t.Fatalf("got %+v", prs)
	}
	if prs[0].HeadRef != "feat/pfe-contract" || prs[0].HeadOid != "abc123" || prs[0].ReviewDecision != "APPROVED" || prs[0].Draft {
		t.Fatalf("got %+v", prs[0])
	}
}

func TestParsePRsInvalid(t *testing.T) {
	if _, err := parsePRs([]byte("nope")); err == nil {
		t.Fatal("expected error")
	}
}

func TestPRQuery(t *testing.T) {
	q := prQuery([]string{"feat/a", `we"ird`})
	for _, want := range []string{`h0:pullRequests(headRefName:"feat/a"`, `h1:pullRequests(headRefName:"we\"ird"`, "headRefOid"} {
		if !strings.Contains(q, want) {
			t.Fatalf("query missing %q:\n%s", want, q)
		}
	}
}

func TestPRHeads(t *testing.T) {
	got := PRHeads([]Worktree{
		{Branch: "a", Upstream: "origin/a"},
		{Branch: "b", Upstream: "origin/feature/b"},
		{Detached: true},
		{Branch: "a"},
		{Branch: "main"},
		{Branch: "c", Upstream: "origin/main"},
	}, "main")
	want := []string{"a", "b", "feature/b", "c"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestMatchPR(t *testing.T) {
	prs := []PR{
		{Number: 1, HeadRef: "foo/bar", State: "MERGED"},
		{Number: 9, HeadRef: "foo/bar", State: "OPEN"},
		{Number: 5, HeadRef: "local-only"},
		{Number: 7, HeadRef: "upstream-name"},
	}
	tests := []struct {
		name string
		w    Worktree
		want int
	}{
		{"upstream with nested slash", Worktree{Branch: "other", Upstream: "origin/foo/bar"}, 9},
		{"upstream beats local branch", Worktree{Branch: "local-only", Upstream: "origin/upstream-name"}, 7},
		{"falls back to local branch", Worktree{Branch: "local-only"}, 5},
		{"upstream without PR falls back", Worktree{Branch: "local-only", Upstream: "origin/gone"}, 5},
		{"highest number wins", Worktree{Branch: "foo/bar"}, 9},
		{"detached", Worktree{Detached: true}, 0},
		{"no match", Worktree{Branch: "nothing", Upstream: "origin/nothing"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchPR(prs, tt.w)
			if tt.want == 0 {
				if got != nil {
					t.Fatalf("got %+v, want nil", got)
				}
				return
			}
			if got == nil || got.Number != tt.want {
				t.Fatalf("got %+v, want #%d", got, tt.want)
			}
		})
	}
}

func TestMatchPRReturnsCopy(t *testing.T) {
	prs := []PR{{Number: 1, HeadRef: "a", Title: "x"}}
	got := MatchPR(prs, Worktree{Branch: "a"})
	got.Title = "changed"
	if prs[0].Title != "x" {
		t.Fatal("MatchPR returned a pointer into the slice")
	}
}

func TestPRWord(t *testing.T) {
	var nilPR *PR
	tests := []struct {
		pr   *PR
		want string
	}{
		{nilPR, "—"},
		{&PR{State: "MERGED", Number: 1}, "merged"},
		{&PR{State: "CLOSED", Number: 2}, "closed"},
		{&PR{State: "OPEN", Number: 618}, "open #618"},
		{&PR{State: "OPEN", Number: 744, Draft: true}, "draft #744"},
	}
	for _, tt := range tests {
		if got := tt.pr.Word(); got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
}
