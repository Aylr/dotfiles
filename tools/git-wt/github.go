package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

type ghPR struct {
	Number         int    `json:"number"`
	Title          string `json:"title"`
	State          string `json:"state"`
	IsDraft        bool   `json:"isDraft"`
	HeadRefName    string `json:"headRefName"`
	HeadRefOid     string `json:"headRefOid"`
	URL            string `json:"url"`
	ReviewDecision string `json:"reviewDecision"`
}

const prFields = "number title state isDraft headRefName headRefOid url reviewDecision"

var prSource = func(root string, heads []string) ([]PR, error) {
	if len(heads) == 0 {
		return nil, nil
	}
	if _, err := exec.LookPath("gh"); err != nil {
		return nil, errors.New("gh not installed")
	}
	out, err := run(root, "gh", "api", "graphql", "-F", "owner={owner}", "-F", "repo={repo}", "-f", "query="+prQuery(heads))
	if err != nil {
		return nil, err
	}
	return parsePRs([]byte(out))
}

func prQuery(heads []string) string {
	var b strings.Builder
	b.WriteString("query($owner:String!,$repo:String!){repository(owner:$owner,name:$repo){")
	for i, h := range heads {
		name, _ := json.Marshal(h)
		fmt.Fprintf(&b, "h%d:pullRequests(headRefName:%s,first:5,orderBy:{field:CREATED_AT,direction:DESC}){nodes{%s}}", i, name, prFields)
	}
	b.WriteString("}}")
	return b.String()
}

func PRHeads(ws []Worktree, defaultBranch string) []string {
	seen := map[string]bool{defaultBranch: true}
	var heads []string
	add := func(h string) {
		if h != "" && !seen[h] {
			seen[h] = true
			heads = append(heads, h)
		}
	}
	for _, w := range ws {
		add(w.Branch)
		if _, name, ok := strings.Cut(w.Upstream, "/"); ok {
			add(name)
		}
	}
	return heads
}

func parsePRs(data []byte) ([]PR, error) {
	var resp struct {
		Data struct {
			Repository map[string]struct {
				Nodes []ghPR `json:"nodes"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unexpected gh output: %w", err)
	}
	var prs []PR
	for _, conn := range resp.Data.Repository {
		for _, r := range conn.Nodes {
			prs = append(prs, PR{
				Number:         r.Number,
				Title:          oneLine(r.Title),
				URL:            r.URL,
				State:          r.State,
				Draft:          r.IsDraft,
				HeadRef:        r.HeadRefName,
				HeadOid:        r.HeadRefOid,
				ReviewDecision: r.ReviewDecision,
			})
		}
	}
	sort.Slice(prs, func(i, j int) bool { return prs[i].Number < prs[j].Number })
	return prs, nil
}

func MatchPR(prs []PR, w Worktree) *PR {
	if w.Upstream != "" {
		if _, name, ok := strings.Cut(w.Upstream, "/"); ok {
			if pr := latestPR(prs, name); pr != nil {
				return pr
			}
		}
	}
	if w.Branch == "" {
		return nil
	}
	return latestPR(prs, w.Branch)
}

func latestPR(prs []PR, headRef string) *PR {
	var best *PR
	for i := range prs {
		if prs[i].HeadRef == headRef && (best == nil || prs[i].Number > best.Number) {
			best = &prs[i]
		}
	}
	if best == nil {
		return nil
	}
	cp := *best
	return &cp
}

func (p *PR) Word() string {
	switch {
	case p == nil:
		return "—"
	case p.State == "MERGED":
		return "merged"
	case p.State == "CLOSED":
		return "closed"
	case p.Draft:
		return fmt.Sprintf("draft #%d", p.Number)
	default:
		return fmt.Sprintf("open #%d", p.Number)
	}
}
