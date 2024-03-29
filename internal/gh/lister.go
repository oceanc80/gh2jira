// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gh

import (
	"context"
	"strings"

	"github.com/google/go-github/v47/github"
)

type ListSpec struct {
	project   string
	milestone string
	assignee  string
	labels    []string
}

type ListOption func(*ListSpec) error

func WithProject(project string) ListOption {
	return func(l *ListSpec) error {
		l.project = project
		return nil
	}
}

func WithMilestone(milestone string) ListOption {
	return func(l *ListSpec) error {
		l.milestone = milestone
		return nil
	}
}

func WithAssignee(assignee string) ListOption {
	return func(l *ListSpec) error {
		l.assignee = assignee
		return nil
	}
}

func WithLabels(labels ...string) ListOption {
	return func(l *ListSpec) error {
		l.labels = labels
		return nil
	}
}

func (a *ListSpec) GetGithubOrg() string {
	return strings.Split(a.project, "/")[0]
}

func (a *ListSpec) GetGithubRepo() string {
	s := strings.Split(a.project, "/")
	if len(s) == 1 {
		return s[0]
	}
	return s[1]
}

func (c *Connection) GetIssue(issueNum int, options ...ListOption) (*github.Issue, error) {
	action := &ListSpec{}
	for _, opt := range options {
		if err := opt(action); err != nil {
			return nil, err
		}
	}

	issue, _, err := c.client.Issues.Get(c.ctx, action.GetGithubOrg(), action.GetGithubRepo(), issueNum)

	if err != nil {
		return nil, err
	}
	return issue, nil
}

// returns a list of all matching issues until there are no more pages
func (c *Connection) ListIssues(options ...ListOption) ([]*github.Issue, error) {
	action := &ListSpec{}
	for _, opt := range options {
		if err := opt(action); err != nil {
			return nil, err
		}
	}

	opt := &github.IssueListByRepoOptions{
		ListOptions: github.ListOptions{PerPage: 50},
		State:       "open",
		Milestone:   action.milestone,
		Assignee:    action.assignee,
		Labels:      action.labels,
	}

	var allIssues []*github.Issue

	for {
		issues, resp, err := c.client.Issues.ListByRepo(
			context.Background(),
			action.GetGithubOrg(),
			action.GetGithubRepo(),
			opt,
		)

		if err != nil {
			return nil, err
		}

		allIssues = append(allIssues, issues...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allIssues, nil
}
