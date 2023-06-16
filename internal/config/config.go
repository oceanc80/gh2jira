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
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	JiraBaseURL string `yaml:"jiraBaseURL"`
	AuthTokens  struct {
		JiraToken   string `yaml:"jira"`
		GithubToken string `yaml:"github"`
	} `yaml:"authTokens"`
}

func ReadConfigYaml(file string) (*Config, error) {
	data, err := readFile(file)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal([]byte(data), &config)
	if err != nil {
		return nil, err
	} else if config.JiraBaseURL == "" {
		return nil, errors.New("missing required jira base url")
	} else if config.AuthTokens.GithubToken == "" {
		return nil, errors.New("missing required github token")
	} else if config.AuthTokens.JiraToken == "" {
		return nil, errors.New("missing required jira token")
	}

	return &config, nil
}

// overrideable func for mocking os.ReadFile
var readFile = func(file string) ([]byte, error) {
	return os.ReadFile(file)
}
