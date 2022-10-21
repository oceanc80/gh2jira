package token

import (
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Token", func() {
	// Test out the Token yaml struct and util methods
	Context("Tokens", func() {
		Describe("ReadTokensYaml", func() {
			var (
				expectedGhToken   string = "foo"
				expectedJiraToken string = "bar"
				mockReadFileGood         = func(file string) ([]byte, error) {
					data := fmt.Sprintf(`
githubToken: %s
jiraToken: %s
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
githubToken: foo
jiraToken= bar
`
					return []byte(data), nil
				}
				mockReadFileMissingGhToken = func(file string) ([]byte, error) {
					data := `
githubToken: foo
`
					return []byte(data), nil
				}
				mockReadFileMissingJiraToken = func(file string) ([]byte, error) {
					data := `
jiraToken: bar
`
					return []byte(data), nil
				}
			)
			It("should unmarshal given data into Tokens struct", func() {
				readFile = mockReadFileGood
				token, err := ReadTokensYaml("")
				Expect(err).NotTo(HaveOccurred())
				Expect(token.GithubToken).To(Equal(expectedGhToken))
				Expect(token.JiraToken).To(Equal(expectedJiraToken))
			})
			It("should handle and return any errors when reading files", func() {
				readFile = mockReadFileBadFile
				token, err := ReadTokensYaml("")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("oh no!"))
				Expect(token).To(BeNil())
			})
			It("should handle and return any errors when unmarshalling yaml", func() {
				readFile = mockReadFileBadYaml
				token, err := ReadTokensYaml("")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("could not find expected ':'"))
				Expect(token).To(BeNil())
			})
			It("should return an error when missing jira token", func() {
				readFile = mockReadFileMissingGhToken
				token, err := ReadTokensYaml("")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("missing required jira token"))
				Expect(token).To(BeNil())
			})
			It("should return an error when missing github token", func() {
				readFile = mockReadFileMissingJiraToken
				token, err := ReadTokensYaml("")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("missing required github token"))
				Expect(token).To(BeNil())
			})
		})
	})
})
