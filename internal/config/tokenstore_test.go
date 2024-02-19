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

package config

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	expectedGhToken   string = "foo"
	expectedJiraToken string = "bar"
	mockReadFileGood         = func(file string) ([]byte, error) {
		data := fmt.Sprintf(`
schema: gh2jira.tokenstore
authTokens: 
 jira: %s
 github: %s
`, expectedJiraToken, expectedGhToken)
		return []byte(data), nil
	}
	mockReadFileBadFile = func(file string) ([]byte, error) {
		return nil, errors.New("oh no!")
	}
	mockReadFileBadYaml = func(file string) ([]byte, error) {
		data := `
schema: gh2jira.tokenstore
authTokens: 
jira= bar
github: foo
`
		return []byte(data), nil
	}
	mockReadFileMissingGhToken = func(file string) ([]byte, error) {
		data := `
schema: gh2jira.tokenstore
authTokens: 
jira: foo
`
		return []byte(data), nil
	}
	mockReadFileMissingJiraToken = func(file string) ([]byte, error) {
		data := `
schema: gh2jira.tokenstore
authTokens: 
github: bar
`
		return []byte(data), nil
	}
)

func TestReadFile(t *testing.T) {
	tests := []struct {
		name      string
		mock      func(file string) ([]byte, error)
		ghtoken   string
		jiratoken string
		wantErr   bool
	}{
		{
			name:      "ReadFileGood",
			mock:      mockReadFileGood,
			ghtoken:   expectedGhToken,
			jiratoken: expectedJiraToken,
			wantErr:   false,
		},
		{
			name:    "ReadFileBadFile",
			mock:    mockReadFileBadFile,
			wantErr: true,
		},
		{
			name:    "ReadFileBadYaml",
			mock:    mockReadFileBadYaml,
			wantErr: true,
		},
		{
			name:      "ReadFileMissingGhToken",
			mock:      mockReadFileMissingGhToken,
			jiratoken: expectedGhToken,
			wantErr:   true,
		},
		{
			name:    "ReadFileMissingJiraToken",
			mock:    mockReadFileMissingJiraToken,
			ghtoken: expectedGhToken,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			readFile = tt.mock
			token, err := ReadTokenStore("")
			if !tt.wantErr {
				require.NoError(t, err)
				require.Equal(t, expectedGhToken, token.Tokens.GithubToken)
				require.Equal(t, expectedJiraToken, token.Tokens.JiraToken)
			}
		})
	}
}
