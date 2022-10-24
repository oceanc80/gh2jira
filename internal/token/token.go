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

package token

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

type Tokens struct {
	GithubToken string `yaml:"githubToken"`
	JiraToken   string `yaml:"jiraToken"`
}

func ReadTokensYaml(file string) (*Tokens, error) {
	data, err := readFile(file)
	if err != nil {
		return nil, err
	}

	var tokens Tokens
	err = yaml.Unmarshal([]byte(data), &tokens)
	if err != nil {
		return nil, err
	} else if tokens.GithubToken == "" {
		return nil, errors.New("missing required github token")
	} else if tokens.JiraToken == "" {
		return nil, errors.New("missing required jira token")
	}

	return &tokens, nil
}

// overrideable func for mocking os.ReadFile
var readFile = func(file string) ([]byte, error) {
	return os.ReadFile(file)
}
