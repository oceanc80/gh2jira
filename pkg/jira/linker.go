package jira

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// DefaultRelationship is the neutral relationship for new remote links.
const DefaultRelationship = "relates to"

var (
	jiraKeyPattern      = regexp.MustCompile(`^[A-Z][A-Z0-9_]*-[1-9][0-9]*$`)
	githubOwnerPattern  = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?$`)
	githubRepoPattern   = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
	githubNumberPattern = regexp.MustCompile(`^[0-9]+$`)
)

// LinkResult reports the outcome for one normalized GitHub URL.
type LinkResult struct {
	URL          string
	Relationship string
	Status       string
}

// ValidateLinkArguments validates the entire batch without configuration or I/O.
// It returns one remote link per distinct, normalized GitHub URL.
func ValidateLinkArguments(issueKey string, urls []string, relationship string) ([]RemoteLink, error) {
	if !jiraKeyPattern.MatchString(issueKey) {
		return nil, fmt.Errorf("invalid Jira key %q: expected PROJECT-123", issueKey)
	}
	switch relationship {
	case DefaultRelationship, "implements", "fixes", "depends on":
	default:
		return nil, fmt.Errorf("invalid relationship %q: use relates to, implements, fixes, or depends on", relationship)
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("at least one GitHub issue or pull request URL is required")
	}
	var links []RemoteLink
	seen := make(map[string]bool)
	for i, raw := range urls {
		normalized, title, err := normalizeGitHubURL(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid GitHub URL at argument %d: %w", i+2, err)
		}
		if seen[normalized] {
			continue
		}
		seen[normalized] = true
		digest := sha256.Sum256([]byte(normalized + "\n" + relationship))
		links = append(links, RemoteLink{
			GlobalID:     fmt.Sprintf("gh2jira:github:%x", digest),
			Relationship: relationship,
			Object:       &RemoteLinkObject{URL: normalized, Title: title},
		})
	}
	return links, nil
}

func normalizeGitHubURL(raw string) (string, string, error) {
	u, err := url.Parse(raw)
	// Parser errors can contain the entire input, including credentials. Only
	// static reasons may leave this function; the caller adds argument position.
	switch {
	case err != nil:
		return "", "", errors.New("malformed URL")
	case u.User != nil:
		return "", "", errors.New("userinfo is not supported")
	case u.Scheme != "https" || u.Host != "github.com":
		return "", "", errors.New("expected HTTPS with exactly the github.com host and no port")
	case u.RawQuery != "" || u.ForceQuery:
		return "", "", errors.New("queries are not supported")
	case strings.ContainsAny(raw, "%#"):
		return "", "", errors.New("fragments and percent encoding are not supported")
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) != 4 || !githubOwnerPattern.MatchString(parts[0]) ||
		!githubRepoPattern.MatchString(parts[1]) || parts[1] == "." || parts[1] == ".." ||
		(parts[2] != "issues" && parts[2] != "pull") || !githubNumberPattern.MatchString(parts[3]) {
		return "", "", errors.New("expected owner/repo/issues/N or owner/repo/pull/N")
	}
	number := strings.TrimLeft(parts[3], "0")
	if number == "" {
		return "", "", errors.New("issue or pull request number must be positive")
	}
	repo := strings.ToLower(parts[0] + "/" + parts[1])
	kind := "issue"
	if parts[2] == "pull" {
		kind = "pull request"
	}
	return "https://github.com/" + repo + "/" + parts[2] + "/" + number,
		fmt.Sprintf("%s %s #%s", repo, kind, number), nil
}

func neutralRelationship(relationship string) string {
	if relationship == "" {
		return DefaultRelationship
	}
	return relationship
}

// Link associates existing GitHub URLs with an existing Jira issue. It only
// reads and adds remote links; dry runs read existing links but never write.
func (conn *Connection) Link(ctx context.Context, issueKey string, urls []string, relationship string, dryRun bool) ([]LinkResult, error) {
	links, err := ValidateLinkArguments(issueKey, urls, relationship)
	if err != nil {
		return nil, err
	}
	if err := conn.Connect(); err != nil {
		return nil, err
	}
	existing, err := conn.GetRemoteLinks(ctx, issueKey)
	if err != nil {
		return nil, fmt.Errorf("reading links for %s: %w", issueKey, err)
	}

	// Plan the complete batch first, including checks against conflicting IDs.
	matched := make([]bool, len(links))
	for i, link := range links {
		for _, old := range existing {
			var oldURL string
			if old.Object != nil {
				oldURL, _, _ = normalizeGitHubURL(old.Object.URL)
			}
			same := oldURL == link.Object.URL && neutralRelationship(old.Relationship) == link.Relationship
			if old.GlobalID == link.GlobalID && !same {
				return nil, fmt.Errorf("remote link ID conflict for %s on %s", link.Object.URL, issueKey)
			}
			matched[i] = matched[i] || same
		}
	}

	var results []LinkResult
	for i, link := range links {
		status := "already linked"
		if !matched[i] {
			status = "would link"
			if !dryRun {
				if err := conn.AddRemoteLink(ctx, issueKey, &link); err != nil {
					return results, fmt.Errorf("linking %s to %s: %w", link.Object.URL, issueKey, err)
				}
				status = "linked"
			}
		}
		results = append(results, LinkResult{URL: link.Object.URL, Relationship: link.Relationship, Status: status})
	}
	return results, nil
}
