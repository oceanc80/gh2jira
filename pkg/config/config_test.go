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
package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/oceanc80/gh2jira/pkg/util"
)

var (
	profilesContent = `
profiles:
- description: Test Profile
  githubConfig:
    project: testdomain/testproject
    lifecycle: agile
  jiraConfig:
    project: TESTY
    lifecycle: agile
  lifecycleMapping: mapping1
  tokensStore: valid_token_file.yaml
`

	mockReadProfilesSuccess = func(filename string) ([]byte, error) {
		return []byte(profilesContent), nil
	}
	mockReadProfilesFailure = func(filename string) ([]byte, error) {
		return nil, errors.New("mock profiles read error")
	}

	// Mock readTokens function for testing
	mockReadTokensSuccess = func(filename string) (*TokenPair, error) {
		return &TokenPair{
			GithubToken: "mock_github_token",
			JiraToken:   "mock_jira_token",
		}, nil
	}

	// Mock readTokens function for testing
	mockReadTokensError = func(filename string) (*TokenPair, error) {
		return nil, errors.New("mock tokens read error")
	}
)

func TestConfig_Read(t *testing.T) {

	tests := []struct {
		name              string
		mockTokenReader   func(filename string) (*TokenPair, error)
		mockProfileReader func(filename string) ([]byte, error)
		flags             *util.FlagFeeder
		audit             func(t *testing.T, err error, c *Config)
	}{
		{
			name:              "successful defaults (no tokens)",
			mockTokenReader:   mockReadTokensSuccess,
			mockProfileReader: nil,
			flags: &util.FlagFeeder{
				ProfilesFile:  "profiles.yaml", // this is defaulted on in cmd/root.go
				ProfileName:   "",
				TokenFile:     "tokenstore.yaml", // this is defaulted on in cmd/root.go
				GithubProject: "",
				JiraProject:   "",
			},
			audit: func(t *testing.T, err error, c *Config) {
				require.NoError(t, err)
				require.Equal(t, "operator-framework/operator-lifecycle-manager", c.GithubProject)
				require.Equal(t, defaultJiraProject, c.JiraProject)
				require.Equal(t, "mock_github_token", c.Tokens.GithubToken)
				require.Equal(t, "mock_jira_token", c.Tokens.JiraToken)
			},
		},
		{
			name:              "successful with profile",
			mockTokenReader:   mockReadTokensSuccess,
			mockProfileReader: mockReadProfilesSuccess,
			flags: &util.FlagFeeder{
				ProfilesFile:  "profiles.yaml", // this is defaulted on in cmd/root.go
				ProfileName:   "test profile",
				TokenFile:     "tokenstore.yaml", // this is defaulted on in cmd/root.go
				GithubProject: "",
				JiraProject:   "",
			},
			audit: func(t *testing.T, err error, c *Config) {
				require.NoError(t, err)
				require.Equal(t, "testdomain/testproject", c.GithubProject)
				require.Equal(t, "TESTY", c.JiraProject)
				require.Equal(t, "mock_github_token", c.Tokens.GithubToken)
				require.Equal(t, "mock_jira_token", c.Tokens.JiraToken)
			},
		},
		{
			name:              "successful with profile and override projects",
			mockTokenReader:   mockReadTokensSuccess,
			mockProfileReader: mockReadProfilesSuccess,
			flags: &util.FlagFeeder{
				ProfilesFile:  "profiles.yaml", // this is defaulted on in cmd/root.go
				ProfileName:   "test profile",
				TokenFile:     "tokenstore.yaml", // this is defaulted on in cmd/root.go
				GithubProject: "overridedomain/overrideproject",
				JiraProject:   "OVER",
			},
			audit: func(t *testing.T, err error, c *Config) {
				require.NoError(t, err)
				require.Equal(t, "overridedomain/overrideproject", c.GithubProject)
				require.Equal(t, "OVER", c.JiraProject)
				require.Equal(t, "mock_github_token", c.Tokens.GithubToken)
				require.Equal(t, "mock_jira_token", c.Tokens.JiraToken)
			},
		},
		{
			name:              "error reading profiles",
			mockTokenReader:   nil,
			mockProfileReader: mockReadProfilesFailure,
			flags: &util.FlagFeeder{
				ProfilesFile:  "profiles.yaml", // this is defaulted on in cmd/root.go
				ProfileName:   "test profile",
				TokenFile:     "tokenstore.yaml", // this is defaulted on in cmd/root.go
				GithubProject: "",
				JiraProject:   "",
			},
			audit: func(t *testing.T, err error, c *Config) {
				require.Error(t, err)
				require.Equal(t, "mock profiles read error", err.Error())
			},
		},
		{
			name:              "error reading tokens",
			mockTokenReader:   mockReadTokensError,
			mockProfileReader: nil,
			flags: &util.FlagFeeder{
				ProfilesFile:  "profiles.yaml", // this is defaulted on in cmd/root.go
				ProfileName:   "",
				TokenFile:     "tokenstore.yaml", // this is defaulted on in cmd/root.go
				GithubProject: "",
				JiraProject:   "",
			},
			audit: func(t *testing.T, err error, c *Config) {
				require.Error(t, err)
				require.Equal(t, "mock tokens read error", err.Error())
			},
		},
	}

	for _, tt := range tests {
		config := NewConfig(tt.flags)
		readTokens = tt.mockTokenReader
		readProfiles = tt.mockProfileReader
		err := config.Read()
		tt.audit(t, err, config)
	}
}
