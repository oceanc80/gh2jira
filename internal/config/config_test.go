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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	// Test out the Config yaml struct and util methods
	Context("Config", func() {
		Describe("ReadFile", func() {
			var (
				expectedGhToken   string = "foo"
				expectedJiraToken string = "bar"
				mockReadFileGood         = func(file string) ([]byte, error) {
					data := fmt.Sprintf(`
schema: gh2jira.config
tokens:
  github: %s
  jira: %s
`,
						expectedGhToken,
						expectedJiraToken)
					return []byte(data), nil
				}
				mockReadFileBadFile = func(file string) ([]byte, error) {
					return nil, errors.New("oh no!")
				}
				mockReadFileBadYaml = func(file string) ([]byte, error) {
					data := `
schema: gh2jira.config
githubToken: foo
tokens:
  jira= bar
`
					return []byte(data), nil
				}
				mockReadFileMissingGhToken = func(file string) ([]byte, error) {
					data := `
schema: gh2jira.config
tokens:
  github: foo
`
					return []byte(data), nil
				}
				mockReadFileMissingJiraToken = func(file string) ([]byte, error) {
					data := `
schema: gh2jira.config
jiraToken: bar
`
					return []byte(data), nil
				}
			)
			It("should unmarshal given data into Tokens struct", func() {
				readFile = mockReadFileGood
				config, err := ReadFile("")
				Expect(err).NotTo(HaveOccurred())
				Expect(config.Tokens.GithubToken).To(Equal(expectedGhToken))
				Expect(config.Tokens.JiraToken).To(Equal(expectedJiraToken))
			})
			It("should handle and return any errors when reading files", func() {
				readFile = mockReadFileBadFile
				config, err := ReadFile("")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("oh no!"))
				Expect(config).To(BeNil())
			})
			It("should handle and return any errors when unmarshalling yaml", func() {
				readFile = mockReadFileBadYaml
				config, err := ReadFile("")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("cannot unmarshal string into Go struct field"))
				Expect(config).To(BeNil())
			})
			It("should return an error when missing jira token", func() {
				readFile = mockReadFileMissingGhToken
				config, err := ReadFile("")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("missing required jira token"))
				Expect(config).To(BeNil())
			})
			It("should return an error when missing github token", func() {
				readFile = mockReadFileMissingJiraToken
				config, err := ReadFile("")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("missing required github token"))
				Expect(config).To(BeNil())
			})
		})
	})
})
