package jira

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/google/go-github/v60/github"
)

func getDomainFromIssueUrl(raw string) string {
	if raw == "" {
		return ""
	}

	if _, err := canonicalSourceURL(raw); err != nil {
		return ""
	}
	parsed, _ := url.Parse(raw) // canonicalSourceURL has already validated this URL.
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	return strings.Join(parts[:2], "/")
}

func getIssueNumberFromIssueUrl(url string) string {
	if url == "" {
		return ""
	}

	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return ""
	}

	return parts[len(parts)-1]
}

func expandDescription(body, rawURL string) (string, error) {
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	var out []string

	matcher := regexp.MustCompile(`^- \[ \] #([0-9]+)(?: (.*))?$`)

	for _, line := range lines {
		if parts := matcher.FindStringSubmatch(line); parts != nil {
			issue, trailer := parts[1], parts[2]
			linkedURL, err := url.Parse(rawURL)
			if err != nil {
				return "", fmt.Errorf("invalid checklist source URL")
			}
			path := strings.TrimSuffix(linkedURL.Path, "/")
			linkedURL.Path = path[:strings.LastIndex(path, "/")+1] + issue
			linkedURL.RawPath = ""
			out = append(out,
				fmt.Sprintf(" * [%s|%s] %s",
					fmt.Sprintf("%s#%s", getDomainFromIssueUrl(rawURL), issue),
					linkedURL.String(),
					trailer))
		} else {
			out = append(out, line)
		}
	}

	return strings.Join(out, "\n"), nil
}

// CloneOption configures opt-in clone behavior without changing legacy callers.
type CloneOption func(*cloneOptions)

type cloneOptions struct {
	ifMissing bool
	writer    io.Writer
}

// WithIfMissing reuses a unique source relationship in the destination scope.
func WithIfMissing() CloneOption {
	return func(o *cloneOptions) { o.ifMissing = true }
}

// WithCloneWriter directs action output to a caller-provided writer.
func WithCloneWriter(w io.Writer) CloneOption {
	return func(o *cloneOptions) { o.writer = w }
}

// canonicalSourceURL identifies a repository issue, never a title or number alone.
func canonicalSourceURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u == nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil || u.Opaque != "" {
		return "", fmt.Errorf("source must have a valid GitHub issue URL")
	}
	parts := strings.Split(strings.TrimSuffix(u.Path, "/"), "/")
	if len(parts) != 5 || parts[1] == "" || parts[2] == "" || (parts[3] != "issues" && parts[3] != "pull") {
		return "", fmt.Errorf("source must have a repository issue URL")
	}
	number, err := strconv.Atoi(parts[4])
	if err != nil || number <= 0 {
		return "", fmt.Errorf("source URL must have a positive issue number")
	}
	for _, part := range parts[1:3] {
		if part == "." || part == ".." || strings.ContainsAny(part, "\\?#% \t\r\n") {
			return "", fmt.Errorf("source must have a valid repository issue URL")
		}
	}
	u.Scheme = "https"
	u.Host = strings.ToLower(u.Host)
	u.Path = "/" + strings.ToLower(parts[1]) + "/" + strings.ToLower(parts[2]) + "/" + parts[3] + "/" + strconv.Itoa(number)
	u.RawPath, u.RawQuery, u.Fragment = "", "", ""
	return u.String(), nil
}

func sourceMarker(canonicalURL string) string {
	return fmt.Sprintf("gh2jira-source-%x", sha256.Sum256([]byte(canonicalURL)))
}

func quoteJQL(value string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value) + `"`
}

// Only clone-equivalent relationships establish identity. An absent relationship
// retains compatibility with legacy links; other explicit roles stay separate.
func cloneRelationship(relationship string) bool {
	switch strings.ToLower(strings.TrimSpace(relationship)) {
	case "", "implements", "fixes":
		return true
	default:
		return false
	}
}

// findClone scans the entire destination scope so legacy links and intentional
// multiple relationships remain visible. Complete reads precede any mutation.
func (conn *Connection) findClone(ctx context.Context, sourceURL, marker, project, issueType string) (*Issue, bool, error) {
	jql := "project = " + quoteJQL(project) + " AND issuetype = " + quoteJQL(issueType)
	issues, err := conn.SearchIssues(ctx, jql)
	if err != nil {
		return nil, false, fmt.Errorf("searching clone destination: %w", err)
	}
	type match struct {
		issue  Issue
		linked bool
	}
	matches := make(map[string]match)
	for _, issue := range issues {
		if issue.Key == "" || issue.Fields == nil {
			return nil, false, fmt.Errorf("clone search returned an incomplete Jira issue; refusing to create")
		}
		links, err := conn.GetRemoteLinks(ctx, issue.Key)
		if err != nil {
			return nil, false, fmt.Errorf("reading remote links for %s: %w", issue.Key, err)
		}
		linked := false
		for _, link := range links {
			if link.Object == nil || !cloneRelationship(link.Relationship) {
				continue
			}
			canonical, err := canonicalSourceURL(link.Object.URL)
			if err == nil && canonical == sourceURL {
				linked = true
			}
		}
		marked := false
		for _, label := range issue.Fields.Labels {
			if label == marker {
				marked = true
			}
		}
		if linked || marked {
			matches[issue.Key] = match{issue: issue, linked: linked}
		}
	}
	if len(matches) > 1 {
		keys := make([]string, 0, len(matches))
		for key := range matches {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		return nil, false, fmt.Errorf("ambiguous source relationship: matching Jira issues %s; specify an existing issue using an existing-record link command or Jira's web-link UI, or omit --if-missing for an intentional version-specific clone", strings.Join(keys, ", "))
	}
	for _, found := range matches {
		return &found.issue, found.linked, nil
	}
	return nil, false, nil
}

func (conn *Connection) linkClone(ctx context.Context, issue *Issue, sourceURL string, number int) error {
	if issue == nil || (issue.ID == "" && issue.Key == "") {
		return fmt.Errorf("Jira create returned no issue ID or key; refusing to add a link")
	}
	id := issue.ID
	if id == "" {
		id = issue.Key
	}
	if err := conn.AddRemoteLink(ctx, id, &RemoteLink{Object: &RemoteLinkObject{
		URL: sourceURL, Title: fmt.Sprintf("%s#%d", getDomainFromIssueUrl(sourceURL), number),
	}}); err != nil {
		return fmt.Errorf("adding source link to Jira issue %s: %w", issue.Key, err)
	}
	return nil
}

// Clone creates a Jira issue by default. WithIfMissing opts into scoped reuse
// and recovery; it does not provide a concurrent or cross-machine transaction.
func (conn *Connection) Clone(ctx context.Context, fromIssue *github.Issue, project string, issueType string, dryRun bool, options ...CloneOption) (*Issue, error) {
	o := cloneOptions{writer: os.Stdout}
	for _, option := range options {
		option(&o)
	}
	if o.writer == nil {
		o.writer = io.Discard
	}
	if fromIssue == nil || fromIssue.GetNumber() <= 0 {
		return nil, fmt.Errorf("source must be a non-nil GitHub issue with a positive number")
	}
	sourceURL, err := canonicalSourceURL(fromIssue.GetHTMLURL())
	if err != nil {
		return nil, err
	}
	if getIssueNumberFromIssueUrl(sourceURL) != strconv.Itoa(fromIssue.GetNumber()) {
		return nil, fmt.Errorf("source issue number does not match its URL")
	}
	// Normalization is for identity only, never user-visible source presentation.
	displayURL := fromIssue.GetHTMLURL()
	if strings.TrimSpace(project) == "" || strings.TrimSpace(issueType) == "" {
		return nil, fmt.Errorf("clone requires a destination project and issue type")
	}
	if conn.jiraClient == nil {
		if err := conn.Connect(); err != nil {
			return nil, err
		}
	}

	marker := sourceMarker(sourceURL)
	if o.ifMissing {
		existing, linked, err := conn.findClone(ctx, sourceURL, marker, project, issueType)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			action := "reuse"
			if !linked {
				action = "reuse and add missing source link to"
			}
			if dryRun {
				_, err = fmt.Fprintf(o.writer, "DRY RUN: would %s Jira issue %s for source #%d\n", action, existing.Key, fromIssue.GetNumber())
				return existing, err
			}
			if !linked {
				if err := conn.linkClone(ctx, existing, displayURL, fromIssue.GetNumber()); err != nil {
					return existing, err
				}
			}
			_, err = fmt.Fprintf(o.writer, "Reused Jira issue %s for source #%d\n", existing.Key, fromIssue.GetNumber())
			return existing, err
		}
	}

	description, err := expandDescription(fromIssue.GetBody(), displayURL)
	if err != nil {
		return nil, err
	}

	ji := Issue{
		Fields: &IssueFields{
			Description: description,
			Type: IssueType{
				Name: issueType,
			},
			Project: Project{
				Key: project,
			},
			Summary: fmt.Sprintf("[UPSTREAM] %s #%d", fromIssue.GetTitle(), fromIssue.GetNumber()),
		},
	}

	if o.ifMissing {
		ji.Fields.Labels = []string{marker}
	}
	if dryRun {
		_, err = fmt.Fprintf(o.writer, "DRY RUN: would create Jira issue in project %s, type %s, and add source link for #%d\n", project, issueType, fromIssue.GetNumber())
		return nil, err
	}
	created, err := conn.CreateIssue(ctx, &ji)
	if err != nil {
		return created, err
	}
	if err := conn.linkClone(ctx, created, displayURL, fromIssue.GetNumber()); err != nil {
		return created, err
	}
	_, err = fmt.Fprintf(o.writer, "Created Jira issue %s for source #%d\n", created.Key, fromIssue.GetNumber())
	return created, err
}
