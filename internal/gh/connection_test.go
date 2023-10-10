// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package gh

import (
	"testing"

	"github.com/google/go-github/v47/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	"github.com/stretchr/testify/require"
)

func TestConnection_Connect(t *testing.T) {
	type scenario struct {
		name     string
		options  []ConnectionOption
		wantErr  bool
		errMatch string
	}
	scenarios := []scenario{
		{
			name:     "failure with no token",
			options:  []ConnectionOption{},
			wantErr:  true,
			errMatch: "cannot create github client without a token",
		},
		{
			name: "creates client",
			options: []ConnectionOption{
				WithToken("token"),
				WithTransport(
					mock.NewMockedHTTPClient(
						mock.WithRequestMatch(
							mock.GetReposIssuesByOwnerByRepo,
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
							})))},
			wantErr:  false,
			errMatch: "",
		},
	}
	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			c, err := NewConnection(s.options...)
			require.NoError(t, err)
			err = c.Connect()
			if (err != nil) != s.wantErr {
				t.Errorf("Connect() error = %v, wantErr %v", err, s.wantErr)
				return
			}
			if err != nil {
				require.ErrorContains(t, err, s.errMatch)
			}
		})
	}
}
