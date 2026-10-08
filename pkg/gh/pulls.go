package gh

import (
	"context"
	"fmt"

	"github.com/google/go-github/v60/github"
)

// GetPullRequest reads a pull request, including its merged flag, through the
// existing official GitHub API client. It never follows remote-link URLs.
func (c *Connection) GetPullRequest(ctx context.Context, owner, repo string, number int) (*github.PullRequest, error) {
	if c.client == nil {
		return nil, fmt.Errorf("github client is not connected")
	}
	pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, err
	}
	return pr, nil
}
