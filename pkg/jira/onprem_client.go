package jira

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	onpremise "github.com/andygrunwald/go-jira/v2/onpremise"
)

type onpremClient struct {
	client *onpremise.Client
}

// Pointer metadata distinguishes an empty, verified page from a malformed body.
type dcSearchResult struct {
	Issues  []onpremise.Issue `json:"issues"`
	StartAt *int              `json:"startAt"`
	Total   *int              `json:"total"`
}

func newOnpremClient(baseURL, token string) (*onpremClient, error) {
	tp := &onpremise.BearerAuthTransport{Token: token}
	c, err := onpremise.NewClient(baseURL, tp.Client())
	if err != nil {
		return nil, err
	}
	return &onpremClient{client: c}, nil
}

func (oc *onpremClient) SearchIssues(ctx context.Context, jql string) ([]Issue, error) {
	var result []Issue
	lastIssue := 0

	for {
		u := url.URL{Path: "rest/api/2/search"}
		params := url.Values{}
		params.Set("jql", jql)
		params.Set("maxResults", "1000")
		params.Set("startAt", strconv.Itoa(lastIssue))
		params.Set("fields", "*navigable,labels")
		u.RawQuery = params.Encode()
		req, err := oc.client.NewRequest(ctx, "GET", u.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("creating search request: %w", err)
		}
		var page dcSearchResult
		_, err = oc.client.Do(req, &page)
		if err != nil {
			return nil, fmt.Errorf("searching issues: %w", err)
		}
		if page.StartAt == nil || page.Total == nil || page.Issues == nil || *page.StartAt != lastIssue || *page.Total < lastIssue {
			return nil, fmt.Errorf("incomplete Jira search pagination; refusing partial results")
		}

		total := *page.Total
		if len(page.Issues) > total-lastIssue || (len(page.Issues) == 0 && lastIssue < total) {
			return nil, fmt.Errorf("incomplete Jira search pagination; refusing partial results")
		}
		for _, i := range page.Issues {
			if i.Fields == nil || i.Key == "" {
				return nil, fmt.Errorf("incomplete Jira search issue; refusing partial results")
			}
			result = append(result, convertOnpremIssue(i))
		}

		lastIssue += len(page.Issues)
		if lastIssue >= total {
			break
		}
	}

	return result, nil
}

func convertOnpremIssue(i onpremise.Issue) Issue {
	issue := Issue{
		ID:  i.ID,
		Key: i.Key,
	}
	if i.Fields != nil {
		issue.Fields = &IssueFields{
			Summary:     i.Fields.Summary,
			Description: i.Fields.Description,
			Labels:      i.Fields.Labels,
			Type:        IssueType{Name: i.Fields.Type.Name},
			Project:     Project{Key: i.Fields.Project.Key},
		}
		if i.Fields.Priority != nil {
			issue.Fields.Priority = &Priority{Name: i.Fields.Priority.Name}
		}
		if i.Fields.Status != nil {
			issue.Fields.Status = &Status{Name: i.Fields.Status.Name}
		}
		if i.Fields.Assignee != nil {
			issue.Fields.Assignee = &User{DisplayName: i.Fields.Assignee.DisplayName}
		}
		if i.Fields.Reporter != nil {
			issue.Fields.Reporter = &User{DisplayName: i.Fields.Reporter.DisplayName}
		}
	}
	return issue
}

func (oc *onpremClient) CreateIssue(ctx context.Context, issue *Issue) (*Issue, error) {
	oi := &onpremise.Issue{
		Fields: &onpremise.IssueFields{
			Summary:     issue.Fields.Summary,
			Description: issue.Fields.Description,
			Labels:      issue.Fields.Labels,
			Type: onpremise.IssueType{
				Name: issue.Fields.Type.Name,
			},
			Project: onpremise.Project{
				Key: issue.Fields.Project.Key,
			},
		},
	}

	created, resp, err := oc.client.Issue.Create(ctx, oi)
	if err != nil {
		return nil, fmt.Errorf("creating issue: %w (response: %v)", err, resp)
	}
	if created == nil || (created.ID == "" && created.Key == "") {
		return nil, fmt.Errorf("creating issue: Jira returned no issue ID or key")
	}

	return &Issue{
		ID:  created.ID,
		Key: created.Key,
	}, nil
}

func (oc *onpremClient) AddRemoteLink(ctx context.Context, issueID string, link *RemoteLink) error {
	ol := &onpremise.RemoteLink{
		Relationship: link.Relationship,
		Object: &onpremise.RemoteLinkObject{
			URL:   link.Object.URL,
			Title: link.Object.Title,
		},
	}

	_, _, err := oc.client.Issue.AddRemoteLink(ctx, issueID, ol)
	if err != nil {
		return fmt.Errorf("adding remote link: %w", err)
	}
	return nil
}

func (oc *onpremClient) GetRemoteLinks(ctx context.Context, issueKey string) ([]RemoteLink, error) {
	rlinks, resp, err := oc.client.Issue.GetRemoteLinks(ctx, issueKey)
	if err != nil {
		return nil, fmt.Errorf("getting remote links: %w", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	if rlinks == nil || *rlinks == nil {
		return nil, fmt.Errorf("getting remote links: null payload; refusing unverified absence")
	}
	result := make([]RemoteLink, 0, len(*rlinks))
	for _, rl := range *rlinks {
		var obj *RemoteLinkObject
		if rl.Object != nil {
			obj = &RemoteLinkObject{
				URL:   rl.Object.URL,
				Title: rl.Object.Title,
			}
		}
		result = append(result, RemoteLink{Object: obj, Relationship: rl.Relationship})
	}
	return result, nil
}
