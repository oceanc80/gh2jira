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
	"bytes"
	"fmt"
	"os"

	"github.com/oceanc80/gh2jira/pkg/util"
)

const defaultJiraBaseURL string = "https://issues.redhat.com/"
const defaultGithubProject string = "operator-framework/operator-lifecycle-manager"
const defaultJiraProject string = "OPECO"

type Config struct {
	GithubProject string
	JiraProject   string
	JiraBaseUrl   string
	Tokens        *TokenPair

	Flags *util.FlagFeeder
}

func NewConfig(ff *util.FlagFeeder) *Config {
	return &Config{
		JiraBaseUrl:   defaultJiraBaseURL,
		GithubProject: defaultGithubProject,
		JiraProject:   defaultJiraProject,
		Tokens:        &TokenPair{},
		Flags:         ff,
	}
}

func (c *Config) Read() error {
	// order of precedence for determining the source of operation context:
	// 1. command line overrides via explicit flags (e.g. 'github-project' over profile[profile-name].github-project)
	// 2. requested profile
	// 3. default config file
	// 4. defaults

	tokenFile := ""

	if c.Flags.ProfilesFile != "" && c.Flags.ProfileName != "" {
		b, err := readProfiles(c.Flags.ProfilesFile)
		if err != nil {
			return err
		}
		reader := bytes.NewReader(b)

		profiles, err := ReadProfiles(reader)
		if err != nil {
			return err
		}
		if c.Flags.ProfileName != "" {
			profile := profiles.GetProfile(c.Flags.ProfileName)
			if profile == nil {
				return fmt.Errorf("profile %s not found", c.Flags.ProfileName)
			}
			c.GithubProject = profile.GithubConfig.Project
			c.JiraProject = profile.JiraConfig.Project

			tokenFile = profile.TokenStore
			if tokenFile != "" {
				c.Tokens, err = readTokens(tokenFile)
				if err != nil {
					return err
				}
			}
		}
	}

	if c.Flags.TokenFile != "" {
		tokens, err := readTokens(c.Flags.TokenFile)
		if err != nil {
			return err
		}
		c.Tokens = tokens
	}

	if c.Flags.GithubProject != "" {
		c.GithubProject = c.Flags.GithubProject
	}

	if c.Flags.JiraProject != "" {
		c.JiraProject = c.Flags.JiraProject
	}

	if c.Flags.JiraBaseURL != "" {
		c.JiraBaseUrl = c.Flags.JiraBaseURL
	}

	return nil
}

var readTokens = func(filename string) (*TokenPair, error) {
	rawTokens, err := ReadTokenStore(filename)
	if err != nil {
		return nil, err
	}
	return &TokenPair{
		GithubToken: rawTokens.Tokens.GithubToken,
		JiraToken:   rawTokens.Tokens.JiraToken,
	}, nil
}

// overrideable func for mocking os.ReadFile
var readProfiles = func(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}
