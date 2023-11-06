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
	"fmt"

	"github.com/spf13/cobra"
)

const defaultJiraBaseURL string = "https://issues.redhat.com/"
const defaultGithubProject string = "operator-framework/operator-sdk"
const defaultJiraProject string = "OSDK"

type Config struct {
	GithubProject string
	JiraProject   string
	JiraBaseUrl   string
	Tokens        *TokenPair

	// cobraCmd is the cobra command that is being executed
	cobraCmd *cobra.Command
}

func NewConfig(cmd *cobra.Command) *Config {
	return &Config{
		cobraCmd:      cmd,
		JiraBaseUrl:   defaultJiraBaseURL,
		GithubProject: defaultGithubProject,
		JiraProject:   defaultJiraProject,
		Tokens:        &TokenPair{},
	}
}

func (c *Config) Read() error {
	// order of precedence for determining the source of operation context:
	// 1. command line overrides via explicit flags (e.g. 'github-project' over profile[profile-name].github-project)
	// 2. requested profile
	// 3. default config file
	// 4. defaults

	profileFile, err := c.cobraCmd.Flags().GetString("profiles-file")
	if err != nil {
		return err
	}
	tokenFile := ""

	fmt.Printf(">>> reading profiles from %q\n", profileFile)
	profiles, err := ReadProfiles(profileFile)
	if err != nil {
		return err
	}
	profileName, err := c.cobraCmd.Flags().GetString("profile-name")
	if err != nil {
		return err
	}
	if profileName != "" {
		fmt.Printf(">>> using profile %q\n", profileName)
		profile := profiles.GetProfile(profileName)
		if profile == nil {
			return fmt.Errorf("profile %s not found", profileName)
		}
		c.GithubProject = profile.GithubConfig.Project
		c.JiraProject = profile.JiraConfig.Project
		fmt.Printf(">>> config after reading profiles: %#v\n", c)

		tokenFile = profile.TokenStore
		if tokenFile != "" {
			fmt.Printf(">>> token file from profile: %q\n", tokenFile)
			c.Tokens, err = readTokens(tokenFile)
			if err != nil {
				return err
			}
		}
	}

	tokenFile, err = c.cobraCmd.Flags().GetString("token-file")
	if err != nil {
		return err
	}
	if tokenFile != "" {
		c.Tokens, err = readTokens(tokenFile)
		if err != nil {
			return err
		}
		fmt.Printf(">>> token file from command line: %q\n", tokenFile)
	}

	githubProject, err := c.cobraCmd.Flags().GetString("github-project")
	if err != nil {
		return err
	}
	if githubProject != "" {
		fmt.Printf(">>> github project from command line: %q\n", githubProject)
		c.GithubProject = githubProject
	}

	jiraProject, err := c.cobraCmd.Flags().GetString("jira-project")
	if err != nil {
		return err
	}
	if jiraProject != "" {
		fmt.Printf(">>> jira project from command line: %q\n", jiraProject)
		c.JiraProject = jiraProject
	}
	fmt.Printf(">>> config after processing flags: %#v\n", c)

	return nil
}

func readTokens(filename string) (*TokenPair, error) {
	rawTokens, err := ReadTokenStore(filename)
	if err != nil {
		return nil, err
	}
	return &TokenPair{
		GithubToken: rawTokens.Tokens.GithubToken,
		JiraToken:   rawTokens.Tokens.JiraToken,
	}, nil
}
