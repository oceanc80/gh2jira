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
	"net/http"
	"reflect"
	"testing"

	"github.com/google/go-github/v60/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	"github.com/stretchr/testify/require"
)

func TestLister_GetOrg(t *testing.T) {
	type scenario struct {
		name    string
		project string
		org     string
	}
	scenarios := []scenario{
		{
			name:    "valid project yields valid org",
			project: "operator-framework/operator-lifecycle-manager",
			org:     "operator-framework",
		},
		{
			name:    "empty project yields empty org",
			project: "",
			org:     "",
		},
		{
			name:    "project with no / yields entire string",
			project: "operator-framework",
			org:     "operator-framework",
		},
		{
			name:    "project with leading / yields empty string",
			project: "/operator-framework",
			org:     "",
		},
	}
	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			spec := ListSpec{project: s.project}
			require.Equal(t, spec.GetGithubOrg(), s.org)
		})
	}
}

func TestLister_GetRepo(t *testing.T) {
	type scenario struct {
		name    string
		project string
		repo    string
	}
	scenarios := []scenario{
		{
			name:    "valid project yields valid repo",
			project: "operator-framework/operator-lifecycle-manager",
			repo:    "operator-lifecycle-manager",
		},
		{
			name:    "empty project yields empty repo",
			project: "",
			repo:    "",
		},
		{
			name:    "project with no / yields entire string",
			project: "operator-framework",
			repo:    "operator-framework",
		},
		{
			name:    "project with leading / yields second string",
			project: "/operator-framework",
			repo:    "operator-framework",
		},
	}
	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			spec := ListSpec{project: s.project}
			require.Equal(t, spec.GetGithubRepo(), s.repo)
		})
	}

}

func TestLister_Options(t *testing.T) {
	type scenario struct {
		name    string
		options []ListOption
		want    *ListSpec
		wantErr bool
	}
	scenarios := []scenario{
		{
			name: "WithProject sets project",
			options: []ListOption{
				WithProject("project"),
			},
			want: &ListSpec{
				project: "project",
			},
			wantErr: false,
		},
		{
			name: "WithMilestone sets milestone",
			options: []ListOption{
				WithMilestone("milestone"),
			},
			want: &ListSpec{
				milestone: "milestone",
			},
			wantErr: false,
		},
		{
			name: "WithAssignee sets assignee",
			options: []ListOption{
				WithAssignee("assignee"),
			},
			want: &ListSpec{
				assignee: "assignee",
			},
			wantErr: false,
		},
		{
			name: "WithLabels sets labels",
			options: []ListOption{
				WithLabels("label1", "label2"),
			},
			want: &ListSpec{
				labels: []string{"label1", "label2"},
			},
			wantErr: false,
		},
	}
	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			spec := &ListSpec{}
			for _, opt := range s.options {
				if err := opt(spec); (err != nil) != s.wantErr {
					t.Errorf("ListOption() error = %v, wantErr %v", err, s.wantErr)
					return
				}
			}
			if !reflect.DeepEqual(spec, s.want) {
				t.Errorf("ListOption() = %v, want %v", spec, s.want)
			}
		})
	}
}

func TestLister_ListIssues(t *testing.T) {
	type scenario struct {
		name              string
		options           []ListOption
		connectionOptions []ConnectionOption
		want              []*github.Issue
		wantErr           bool
		errMatch          string
	}
	scenarios := []scenario{
		{
			name: "finds open issues",
			options: []ListOption{
				WithProject("fakeorg/fakeproject"),
			},
			connectionOptions: []ConnectionOption{
				WithToken("token"),
				WithContext(context.Background()),
				WithTransport(mock.NewMockedHTTPClient(
					mock.WithRequestMatch(mock.GetReposIssuesByOwnerByRepo,
						[]github.Issue{
							{
								ID:    github.Int64(123),
								Title: github.String("Issue 1"),
								State: github.String("open"),
							},
							{
								ID:    github.Int64(456),
								Title: github.String("Issue 2"),
								State: github.String("open"),
							},
						},
					),
				)),
			},
			want: []*github.Issue{
				{
					ID:    github.Int64(123),
					Title: github.String("Issue 1"),
					State: github.String("open"),
				},
				{
					ID:    github.Int64(456),
					Title: github.String("Issue 2"),
					State: github.String("open"),
				},
			},
			wantErr:  false,
			errMatch: "",
		},
		{
			name: "return error if list fails",
			options: []ListOption{
				WithProject("fakeorg/fakeproject"),
			},
			connectionOptions: []ConnectionOption{
				WithToken("token"),
				WithContext(context.Background()),
				WithTransport(mock.NewMockedHTTPClient(
					mock.WithRequestMatchHandler(
						mock.GetReposIssuesByOwnerByRepo,
						http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							mock.WriteError(
								w,
								http.StatusInternalServerError,
								"github went belly up or something",
							)
						}),
					),
				),
				)},
			want:     []*github.Issue{},
			wantErr:  true,
			errMatch: "github went belly up or something",
		},
	}
	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			c, err := NewConnection(s.connectionOptions...)
			require.NoError(t, err)

			err = c.Connect()
			require.NoError(t, err)

			iss, err := c.ListIssues(s.options...)

			if err == nil {
				require.ElementsMatch(t, iss, s.want)
			} else {
				require.Equal(t, s.wantErr, true)
				require.Nil(t, iss)

				// because the mock library doesn't return a completely valid github.ErrorResponse (missing response)
				// we have to wrap and compare the field(s) we care about
				gherr, ok := err.(*github.ErrorResponse)
				require.True(t, ok)
				require.Contains(t, gherr.Message, s.errMatch)
			}
		})
	}
}

func TestLister_GetIssue(t *testing.T) {
	type scenario struct {
		name              string
		options           []ListOption
		connectionOptions []ConnectionOption
		id                int
		want              *github.Issue
		wantErr           bool
		errMatch          string
	}
	scenarios := []scenario{
		{
			name: "success",
			options: []ListOption{
				WithProject("fakeorg/fakeproject"),
			},
			connectionOptions: []ConnectionOption{
				WithToken("token"),
				WithContext(context.Background()),
				WithTransport(mock.NewMockedHTTPClient(
					mock.WithRequestMatch(mock.GetReposIssuesByOwnerByRepoByIssueNumber,
						github.Issue{
							ID:    github.Int64(456),
							Title: github.String("Issue 2"),
							State: github.String("open"),
						},
					),
				)),
			},
			want: &github.Issue{
				ID:    github.Int64(456),
				Title: github.String("Issue 2"),
				State: github.String("open"),
			},
			id:       456,
			wantErr:  false,
			errMatch: "",
		},
	}
	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			c, err := NewConnection(s.connectionOptions...)
			require.NoError(t, err)

			err = c.Connect()
			require.NoError(t, err)

			iss, err := c.GetIssue(s.id, s.options...)

			if err == nil {
				require.Equal(t, iss, s.want)
			} else {
				require.Equal(t, s.wantErr, true)
				require.Nil(t, iss)

				// because the mock library doesn't return a completely valid github.ErrorResponse (missing response)
				// we have to wrap and compare the field(s) we care about
				gherr, ok := err.(*github.ErrorResponse)
				require.True(t, ok)
				require.Contains(t, gherr.Message, s.errMatch)
			}
		})
	}
}
