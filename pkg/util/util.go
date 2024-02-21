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

package util

import (
	"github.com/spf13/cobra"
)

type FlagFeeder struct {
	ProfilesFile  string
	ProfileName   string
	TokenFile     string
	GithubProject string
	JiraProject   string
	JiraBaseURL   string
}

func NewFlagFeeder(c *cobra.Command) (*FlagFeeder, error) {
	profilesFile, err := c.Flags().GetString("profiles-file")
	if err != nil {
		return nil, err
	}
	profileName, err := c.Flags().GetString("profile-name")
	if err != nil {
		return nil, err
	}
	tokensFile, err := c.Flags().GetString("token-file")
	if err != nil {
		return nil, err
	}
	githubProject, err := c.Flags().GetString("github-project")
	if err != nil {
		return nil, err
	}
	jiraProject, err := c.Flags().GetString("jira-project")
	if err != nil {
		return nil, err
	}
	jiraBaseURL, err := c.Flags().GetString("jira-base-url")
	if err != nil {
		return nil, err
	}

	return &FlagFeeder{
		ProfilesFile:  profilesFile,
		ProfileName:   profileName,
		TokenFile:     tokensFile,
		GithubProject: githubProject,
		JiraProject:   jiraProject,
		JiraBaseURL:   jiraBaseURL,
	}, nil
}
