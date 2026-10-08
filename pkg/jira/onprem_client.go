package jira

import (
	"context"
	"fmt"

	onpremise "github.com/andygrunwald/go-jira/v2/onpremise"
)

type onpremClient struct {
	client *onpremise.Client
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
		opt := &onpremise.SearchOptions{
			MaxResults: 1000,
			StartAt:    lastIssue,
		}
		issues, resp, err := oc.client.Issue.Search(ctx, jql, opt)
		if err != nil {
			return nil, fmt.Errorf("searching issues: %w", err)
		}

		total := resp.Total
		for _, i := range issues {
			result = append(result, convertOnpremIssue(i))
		}

		lastIssue = resp.StartAt + len(issues)
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

	return &Issue{
		ID:  created.ID,
		Key: created.Key,
	}, nil
}

func (oc *onpremClient) AddRemoteLink(ctx context.Context, issueID string, link *RemoteLink) error {
	ol := &onpremise.RemoteLink{
		GlobalID:     link.GlobalID,
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

	var result []RemoteLink
	if rlinks != nil {
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
	}
	return result, nil
}
