package jira

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	cloud "github.com/andygrunwald/go-jira/v2/cloud"
)

type cloudClient struct {
	client *cloud.Client
}

func newCloudClient(baseURL, email, apiToken string) (*cloudClient, error) {
	tp := &cloud.BasicAuthTransport{
		Username: email,
		APIToken: apiToken,
	}
	c, err := cloud.NewClient(baseURL, tp.Client())
	if err != nil {
		return nil, err
	}
	return &cloudClient{client: c}, nil
}

type v3SearchResult struct {
	Issues        []v3Issue `json:"issues"`
	NextPageToken string    `json:"nextPageToken,omitempty"`
	IsLast        *bool     `json:"isLast"`
}

type v3Issue struct {
	ID     string         `json:"id"`
	Key    string         `json:"key"`
	Fields *v3IssueFields `json:"fields"`
}

type v3IssueFields struct {
	Summary     string     `json:"summary"`
	Description any        `json:"description"`
	Labels      []string   `json:"labels"`
	Type        v3Named    `json:"issuetype"`
	Project     v3Keyed    `json:"project"`
	Priority    *v3Named   `json:"priority"`
	Status      *v3Named   `json:"status"`
	Assignee    *v3UserRef `json:"assignee"`
	Reporter    *v3UserRef `json:"reporter"`
}

type v3Named struct {
	Name string `json:"name"`
}

type v3Keyed struct {
	Key string `json:"key"`
}

type v3UserRef struct {
	DisplayName string `json:"displayName"`
}

func (cc *cloudClient) SearchIssues(ctx context.Context, jql string) ([]Issue, error) {
	var result []Issue
	var nextPageToken string
	seenTokens := make(map[string]bool)

	for {
		u := url.URL{Path: "rest/api/3/search/jql"}
		params := url.Values{}
		params.Set("jql", jql)
		params.Set("maxResults", strconv.Itoa(1000))
		params.Set("fields", "*navigable,labels")
		if nextPageToken != "" {
			params.Set("nextPageToken", nextPageToken)
		}
		u.RawQuery = params.Encode()

		req, err := cc.client.NewRequest(ctx, "GET", u.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("creating search request: %w", err)
		}

		var sr v3SearchResult
		_, err = cc.client.Do(req, &sr)
		if err != nil {
			return nil, fmt.Errorf("executing search: %w", err)
		}

		if sr.Issues == nil || sr.IsLast == nil {
			return nil, fmt.Errorf("incomplete Jira search pagination; refusing partial results")
		}
		for _, i := range sr.Issues {
			if i.Fields == nil || i.Key == "" {
				return nil, fmt.Errorf("incomplete Jira search issue; refusing partial results")
			}
			result = append(result, convertV3Issue(i))
		}

		if *sr.IsLast {
			break
		}
		if sr.NextPageToken == "" || seenTokens[sr.NextPageToken] {
			return nil, fmt.Errorf("incomplete Jira search pagination; refusing partial results")
		}
		seenTokens[sr.NextPageToken] = true
		nextPageToken = sr.NextPageToken
	}

	return result, nil
}

func convertV3Issue(i v3Issue) Issue {
	if i.Fields == nil {
		return Issue{ID: i.ID, Key: i.Key}
	}
	issue := Issue{
		ID:  i.ID,
		Key: i.Key,
		Fields: &IssueFields{
			Summary: i.Fields.Summary,
			Labels:  i.Fields.Labels,
			Type:    IssueType{Name: i.Fields.Type.Name},
			Project: Project{Key: i.Fields.Project.Key},
		},
	}
	if desc, ok := i.Fields.Description.(string); ok {
		issue.Fields.Description = desc
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
	return issue
}

func (cc *cloudClient) CreateIssue(ctx context.Context, issue *Issue) (*Issue, error) {
	ci := &cloud.Issue{
		Fields: &cloud.IssueFields{
			Summary:     issue.Fields.Summary,
			Description: issue.Fields.Description,
			Labels:      issue.Fields.Labels,
			Type: cloud.IssueType{
				Name: issue.Fields.Type.Name,
			},
			Project: cloud.Project{
				Key: issue.Fields.Project.Key,
			},
		},
	}

	created, resp, err := cc.client.Issue.Create(ctx, ci)
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

func (cc *cloudClient) AddRemoteLink(ctx context.Context, issueID string, link *RemoteLink) error {
	cl := &cloud.RemoteLink{
		GlobalID:     link.GlobalID,
		Relationship: link.Relationship,
		Object: &cloud.RemoteLinkObject{
			URL:   link.Object.URL,
			Title: link.Object.Title,
		},
	}

	_, _, err := cc.client.Issue.AddRemoteLink(ctx, issueID, cl)
	if err != nil {
		return fmt.Errorf("adding remote link: %w", err)
	}
	return nil
}

func (cc *cloudClient) GetRemoteLinks(ctx context.Context, issueKey string) ([]RemoteLink, error) {
	rlinks, resp, err := cc.client.Issue.GetRemoteLinks(ctx, issueKey)
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
		result = append(result, RemoteLink{GlobalID: rl.GlobalID, Relationship: rl.Relationship, Object: obj})
	}
	return result, nil
}
