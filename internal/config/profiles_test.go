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
	"strings"
	"testing"
)

func TestReadProfiles(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Profiles
		wantErr  bool
	}{
		{
			name: "valid input",
			input: `
profiles:
  - description: Project 1
    githubConfig:
      project: project1
      lifecycle: agile
    jiraConfig:
      project: project1
      lifecycle: agile
    lifecycleMapping: mapping1
    tokensStore: store1
  - description: Project 2
    githubConfig:
      project: somedomain/project2
      lifecycle: waterfail
    jiraConfig:
      project: somedomain/project2
      lifecycle: waterfail
    lifecycleMapping: mapping2
`,
			expected: &Profiles{
				Profiles: []Profile{
					{
						Description: "Project 1",
						GithubConfig: DomainConfig{
							Project:   "project1",
							Lifecycle: "dev",
						},
						JiraConfig: DomainConfig{
							Project:   "project1",
							Lifecycle: "development",
						},
						LifecycleMapping: "mapping1",
						TokenStore:       "store1",
					},
					{
						Description: "Project 2",
						GithubConfig: DomainConfig{
							Project:   "project2",
							Lifecycle: "prod",
						},
						JiraConfig: DomainConfig{
							Project:   "project2",
							Lifecycle: "production",
						},
						LifecycleMapping: "mapping2",
						TokenStore:       "store2",
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "invalid input",
			input:    "invalid yaml",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			actual, err := ReadProfiles(reader)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReadProfiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if len(actual.Profiles) != len(tt.expected.Profiles) {
				t.Errorf("ReadProfiles() actual profiles length = %d, expected profiles length = %d", len(actual.Profiles), len(tt.expected.Profiles))
				return
			}

			for i := range actual.Profiles {
				if actual.Profiles[i].Description != tt.expected.Profiles[i].Description {
					t.Errorf("ReadProfiles() actual profile description = %s, expected profile description = %s", actual.Profiles[i].Description, tt.expected.Profiles[i].Description)
				}
			}
		})
	}
}

func TestProfiles_GetProfile(t *testing.T) {
	profiles := &Profiles{
		Profiles: []Profile{
			{
				Description: "Project 1",
				GithubConfig: DomainConfig{
					Project:   "project1",
					Lifecycle: "dev",
				},
				JiraConfig: DomainConfig{
					Project:   "project1",
					Lifecycle: "development",
				},
				LifecycleMapping: "mapping1",
				TokenStore:       "store1",
			},
			{
				Description: "Project 2",
				GithubConfig: DomainConfig{
					Project:   "project2",
					Lifecycle: "prod",
				},
				JiraConfig: DomainConfig{
					Project:   "project2",
					Lifecycle: "production",
				},
				LifecycleMapping: "mapping2",
				TokenStore:       "store2",
			},
		},
	}

	tests := []struct {
		name          string
		projectName   string
		expected      *Profile
		expectedFound bool
	}{
		{
			name:          "existing project",
			projectName:   "Project 1",
			expected:      &profiles.Profiles[0],
			expectedFound: true,
		},
		{
			name:          "non-existing project",
			projectName:   "Project 3",
			expected:      nil,
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := profiles.GetProfile(tt.projectName)

			if (actual != nil) != tt.expectedFound {
				t.Errorf("GetProfile() actual found = %v, expected found = %v", actual != nil, tt.expectedFound)
				return
			}

			if actual != nil && actual.Description != tt.expected.Description {
				t.Errorf("GetProfile() actual profile description = %s, expected profile description = %s", actual.Description, tt.expected.Description)
			}
		})
	}
}
