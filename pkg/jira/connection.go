package jira

import (
	"context"
	"errors"

	"github.com/oceanc80/gh2jira/pkg/config"
)

type ConnectionOption func(*Connection) error

type Connection struct {
	jiraClient JiraClient
	auth       config.JiraAuth
	baseUri    string
}

// WithJiraClient supplies an adapter, including synthetic clients for tests.
func WithJiraClient(client JiraClient) ConnectionOption {
	return func(c *Connection) error {
		c.jiraClient = client
		return nil
	}
}

func WithBaseURI(u string) ConnectionOption {
	return func(c *Connection) error {
		c.baseUri = u
		return nil
	}
}

func WithAuth(a config.JiraAuth) ConnectionOption {
	return func(c *Connection) error {
		c.auth = a
		return nil
	}
}

func (c *Connection) BaseUri() string { return c.baseUri }

func NewConnection(options ...ConnectionOption) (*Connection, error) {
	c := &Connection{}
	for _, o := range options {
		if err := o(c); err != nil {
			return nil, err
		}
	}
	if c.baseUri == "" {
		return nil, errors.New("no base URI for jira")
	}

	hasToken := c.auth.Token != ""
	hasCloud := c.auth.Email != "" && c.auth.APIToken != ""
	if !hasToken && !hasCloud {
		return nil, errors.New("cannot access jira without credentials")
	}

	return c, nil
}

func (c *Connection) Connect() error {
	if c.jiraClient != nil {
		return nil
	}

	if c.auth.IsCloud() {
		client, err := newCloudClient(c.baseUri, c.auth.Email, c.auth.APIToken)
		if err != nil {
			return err
		}
		c.jiraClient = client
	} else {
		client, err := newOnpremClient(c.baseUri, c.auth.Token)
		if err != nil {
			return err
		}
		c.jiraClient = client
	}
	return nil
}

func (c *Connection) SearchIssues(ctx context.Context, jql string) ([]Issue, error) {
	return c.jiraClient.SearchIssues(ctx, jql)
}

func (c *Connection) CreateIssue(ctx context.Context, issue *Issue) (*Issue, error) {
	return c.jiraClient.CreateIssue(ctx, issue)
}

func (c *Connection) AddRemoteLink(ctx context.Context, issueID string, link *RemoteLink) error {
	return c.jiraClient.AddRemoteLink(ctx, issueID, link)
}

func (c *Connection) GetRemoteLinks(ctx context.Context, issueKey string) ([]RemoteLink, error) {
	return c.jiraClient.GetRemoteLinks(ctx, issueKey)
}
