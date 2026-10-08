// Package report produces read-only correlations without imposing a workflow
// mapping between independently managed Jira and GitHub states.
package report

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/google/go-github/v60/github"
	"github.com/oceanc80/gh2jira/pkg/gh"
	"github.com/oceanc80/gh2jira/pkg/jira"
)

// JiraReader exposes only the reads needed to build a report.
type JiraReader interface {
	SearchIssues(context.Context, string) ([]jira.Issue, error)
	GetRemoteLinks(context.Context, string) ([]jira.RemoteLink, error)
}

// GitHubReader exposes the existing issue reader and the read-only PR reader.
type GitHubReader interface {
	GetIssue(int, ...gh.ListOption) (*github.Issue, error)
	GetPullRequest(context.Context, string, string, int) (*github.PullRequest, error)
}

// JiraItem contains only downstream identity and state.
type JiraItem struct {
	Key    string `json:"key"`
	Status string `json:"status"`
}

// Source contains upstream identity and state. Merged is present only for PRs.
type Source struct {
	Repository string `json:"repository"`
	Type       string `json:"type"`
	Number     int    `json:"number"`
	URL        string `json:"url"`
	State      string `json:"state"`
	Merged     *bool  `json:"merged,omitempty"`
}

// Row is a correlation or an explicit missing/malformed-link observation.
// It deliberately excludes Jira summaries, descriptions, people and link titles.
type Row struct {
	Jira       JiraItem `json:"jira"`
	GitHub     *Source  `json:"github,omitempty"`
	LinkStatus string   `json:"link_status"`
	Problem    string   `json:"problem,omitempty"`
}

type reference struct {
	owner, repo, kind string
	number            int
}

func (r reference) key() string {
	// Issues and PRs share a repository's number namespace.
	return r.owner + "/" + r.repo + "/" + strconv.Itoa(r.number)
}

func (r reference) source(state string, merged *bool) *Source {
	path := "issues"
	if r.kind == "pull_request" {
		path = "pull"
	}
	return &Source{
		Repository: r.owner + "/" + r.repo, Type: r.kind, Number: r.number,
		URL:   fmt.Sprintf("https://github.com/%s/%s/%s/%d", r.owner, r.repo, path, r.number),
		State: state, Merged: merged,
	}
}

var ownerName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*$`)
var repoName = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
var positiveID = regexp.MustCompile(`^[0-9]+$`)

// parseReference ignores unrelated links. GitHub-looking but unsupported URLs
// yield a problem without exposing potentially sensitive raw URL components.
func parseReference(raw string) (reference, bool, string) {
	var ref reference
	// Only the apparent authority identifies a GitHub reference. A GitHub URL
	// mentioned in an unrelated site's path, query or fragment is not a link.
	authority := raw
	if i := strings.Index(authority, "://"); i >= 0 && !strings.ContainsAny(authority[:i], "/?#") {
		authority = authority[i+3:]
	} else {
		authority = strings.TrimPrefix(authority, "//")
	}
	if i := strings.IndexAny(authority, "/?#"); i >= 0 {
		authority = authority[:i]
	}
	if !strings.Contains(strings.ToLower(authority), "github.com") {
		return ref, false, ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || !strings.EqualFold(u.Host, "github.com") || u.User != nil || u.Opaque != "" {
		return ref, true, "expected https://github.com without userinfo, port or alternate host"
	}
	path := strings.TrimSuffix(u.Path, "/")
	parts := strings.Split(path, "/")
	if strings.Contains(u.EscapedPath(), "%") || len(parts) != 5 || parts[0] != "" ||
		!ownerName.MatchString(parts[1]) || !repoName.MatchString(parts[2]) || parts[2] == "." || parts[2] == ".." ||
		(parts[3] != "issues" && parts[3] != "pull") || !positiveID.MatchString(parts[4]) {
		return ref, true, "expected /owner/repository/issues/positive-ID or /owner/repository/pull/positive-ID"
	}
	number, err := strconv.Atoi(parts[4])
	if err != nil || number <= 0 {
		return ref, true, "GitHub issue or pull request number must be a positive integer"
	}
	ref = reference{owner: strings.ToLower(parts[1]), repo: strings.ToLower(parts[2]), kind: "issue", number: number}
	if parts[3] == "pull" {
		ref.kind = "pull_request"
	}
	return ref, true, ""
}

// Generate returns deterministically ordered rows for a project scope, including
// closed Jira items. Any failed read fails the report instead of implying absence.
func Generate(ctx context.Context, project, query string, jc JiraReader, gc GitHubReader) ([]Row, error) {
	jql, err := Scope(project, query)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	issues, err := jc.SearchIssues(ctx, jql)
	if err != nil {
		return nil, fmt.Errorf("search Jira report scope: %w", err)
	}
	rows := make([]Row, 0)
	cache := make(map[string]*Source)
	for _, issue := range issues {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		item := JiraItem{Key: issue.Key}
		if issue.Fields != nil && issue.Fields.Status != nil {
			item.Status = issue.Fields.Status.Name
		}
		links, err := jc.GetRemoteLinks(ctx, issue.Key)
		if err != nil {
			return nil, fmt.Errorf("read remote links for %s: %w", issue.Key, err)
		}
		seen := make(map[string]bool)
		supported := false
		for _, link := range links {
			if link.Object == nil {
				continue
			}
			ref, candidate, problem := parseReference(link.Object.URL)
			if !candidate {
				continue
			}
			if problem != "" {
				key := "malformed:" + link.Object.URL
				if !seen[key] {
					rows = append(rows, Row{Jira: item, LinkStatus: "malformed_github_reference", Problem: problem})
					seen[key] = true
				}
				continue
			}
			supported = true
			key := ref.key()
			source, ok := cache[key]
			// A PR may be addressed through either URL form, but an ordinary
			// issue does not validate a pull URL. Check before per-item dedup.
			if !ok || (ref.kind == "pull_request" && source.Type != "pull_request") {
				source, err = lookup(ctx, ref, gc)
				if err != nil {
					return nil, fmt.Errorf("read GitHub %s for %s: %w", key, issue.Key, err)
				}
				cache[key] = source
			}
			if seen[key] {
				continue
			}
			seen[key] = true
			rows = append(rows, Row{Jira: item, GitHub: source, LinkStatus: "linked"})
		}
		if !supported {
			rows = append(rows, Row{Jira: item, LinkStatus: "no_supported_github_links"})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.Jira.Key != b.Jira.Key {
			return a.Jira.Key < b.Jira.Key
		}
		if a.Jira.Status != b.Jira.Status {
			return a.Jira.Status < b.Jira.Status
		}
		if a.LinkStatus != b.LinkStatus {
			return a.LinkStatus < b.LinkStatus
		}
		if a.GitHub != nil && b.GitHub != nil {
			if a.GitHub.Repository != b.GitHub.Repository {
				return a.GitHub.Repository < b.GitHub.Repository
			}
			return a.GitHub.Number < b.GitHub.Number
		}
		return a.Problem < b.Problem
	})
	return rows, nil
}

func lookup(ctx context.Context, ref reference, gc GitHubReader) (*Source, error) {
	if ref.kind == "issue" {
		issue, err := gc.GetIssue(ref.number, gh.WithProject(ref.owner+"/"+ref.repo))
		if err != nil {
			return nil, err
		}
		if issue == nil || issue.GetState() == "" {
			return nil, fmt.Errorf("GitHub returned an issue without state")
		}
		if !issue.IsPullRequest() {
			return ref.source(issue.GetState(), nil), nil
		}
		ref.kind = "pull_request"
	}
	pr, err := gc.GetPullRequest(ctx, ref.owner, ref.repo, ref.number)
	if err != nil {
		return nil, err
	}
	if pr == nil || pr.GetState() == "" || pr.Merged == nil {
		return nil, fmt.Errorf("GitHub returned a pull request without state or merged flag")
	}
	return ref.source(pr.GetState(), pr.Merged), nil
}
