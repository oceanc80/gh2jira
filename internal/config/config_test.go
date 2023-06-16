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
	// Test out the config yaml struct and util methods
	Context("Config", func() {
		Describe("ReadConfigYaml", func() {
			var (
				expectedGhToken     string = "foo"
				expectedJiraToken   string = "bar"
				expectedJiraBaseURL string = "https://issues.jira.com"
				mockReadFileGood           = func(file string) ([]byte, error) {

					data := fmt.Sprintf(`
jiraBaseURL: %s
authTokens: 
 github: %s
 jira: %s
`,
						expectedJiraBaseURL,
						expectedGhToken,
						expectedJiraToken)
					return []byte(data), nil
				}
				mockReadFileBadFile = func(file string) ([]byte, error) {
					return nil, errors.New("oh no!")
				}
				mockReadFileBadYaml = func(file string) ([]byte, error) {

					data := `
jiraBaseURL: "https://"
authTokens: 
 github: foo
 jira= bar
`
					return []byte(data), nil
				}
				mockReadFileMissingGhToken = func(file string) ([]byte, error) {
					data := `
jiraBaseURL: "https://"
authTokens: 
 github: foo
`
					return []byte(data), nil
				}
				mockReadFileMissingJiraToken = func(file string) ([]byte, error) {
					data := `
jiraBaseURL: "https://"
authTokens: 
 jira: bar
`
					return []byte(data), nil
				}
			)
			It("should unmarshal given data into Tokens struct", func() {
				readFile = mockReadFileGood
				token, err := ReadConfigYaml("")
				Expect(err).NotTo(HaveOccurred())
				Expect(token.AuthTokens.GithubToken).To(Equal(expectedGhToken))
				Expect(token.AuthTokens.JiraToken).To(Equal(expectedJiraToken))
			})
			It("should handle and return any errors when reading files", func() {
				readFile = mockReadFileBadFile
				token, err := ReadConfigYaml("")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("oh no!"))
				Expect(token).To(BeNil())
			})
			It("should handle and return any errors when unmarshalling yaml", func() {
				readFile = mockReadFileBadYaml
				token, err := ReadConfigYaml("")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("could not find expected ':'"))
				Expect(token).To(BeNil())
			})
			It("should return an error when missing jira token", func() {
				readFile = mockReadFileMissingGhToken
				token, err := ReadConfigYaml("")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("missing required jira token"))
				Expect(token).To(BeNil())
			})
			It("should return an error when missing github token", func() {
				readFile = mockReadFileMissingJiraToken
				token, err := ReadConfigYaml("")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("missing required github token"))
				Expect(token).To(BeNil())
			})
		})
	})
})
