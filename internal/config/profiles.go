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
	"io"
	"os"

	"sigs.k8s.io/yaml"
)

type DomainConfig struct {
	Project   string `json:"project"`
	Lifecycle string `json:"lifecycle"`
}

type Profile struct {
	Description      string       `json:"description,omitempty"`
	GithubConfig     DomainConfig `json:"githubConfig"`
	JiraConfig       DomainConfig `json:"jiraConfig"`
	LifecycleMapping string       `json:"lifecycleMapping"`
	TokenStore       string       `json:"tokensStore,omitempty"`
}

type Profiles struct {
	Profiles []Profile `json:"profiles"`
}

func ReadProfiles(filename string) (*Profiles, error) {

	reader, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var m Profiles
	err = yaml.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

func (p *Profiles) GetProfile(projectName string) *Profile {
	for _, profile := range p.Profiles {
		if profile.Description == projectName {
			return &profile
		}
	}
	return nil
}
