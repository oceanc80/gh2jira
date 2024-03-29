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

package jira

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	gojira "github.com/andygrunwald/go-jira"
	"github.com/google/go-github/v60/github"
)

func getWebURL(url string) string {
	// https://api.github.com/repos/operator-framework/operator-sdk/issues/3447
	// https://github.com/operator-framework/operator-sdk/issues/3447
	if url == "" {
		return url
	}
	return strings.Replace(strings.Replace(url, "api.github.com", "github.com", 1), "repos/", "", 1)
}

func (conn *Connection) Clone(fromIssue *github.Issue, project string, dryRun bool) (*gojira.Issue, error) {
	if conn.client == nil {
		// user attempted operation w/o connecting to remote first
		if err := conn.Connect(); err != nil {
			return nil, err
		}
	}

	ji := gojira.Issue{
		Fields: &gojira.IssueFields{
			// Assignee: &gojira.User{
			//     Name: "myuser",
			// },
			// Reporter: &gojira.User{
			//     Name: "youruser",
			// },
			Description: fromIssue.GetBody(),
			Type: gojira.IssueType{
				Name: "Story",
			},
			Project: gojira.Project{
				Key: project,
			},
			Summary: fmt.Sprintf("[UPSTREAM] %s #%d", fromIssue.GetTitle(), fromIssue.GetNumber()),
		},
	}

	var daIssue *gojira.Issue

	if dryRun {
		fmt.Println("\n############# DRY RUN MODE #############")
		fmt.Printf("Cloning issue #%d to jira project board: %s\n\n", fromIssue.GetNumber(), ji.Fields.Project.Key)
		fmt.Printf("Summary: %s\n", ji.Fields.Summary)
		fmt.Printf("Type: %s\n", ji.Fields.Type.Name)
		fmt.Println("Description:")
		fmt.Printf("%s\n", ji.Fields.Description)
		fmt.Println("\n############# DRY RUN MODE #############")
	} else {
		fmt.Printf("Cloning issue #%d to jira project board: %s\n\n", fromIssue.GetNumber(), ji.Fields.Project.Key)
		var err error

		daIssue, response, err := conn.client.Issue.Create(&ji)
		if err != nil {
			fmt.Printf("Error cloning issue: %v\n", err)
			reqBody, ioerr := io.ReadAll(response.Response.Body)
			if ioerr == nil {
				fmt.Println(string(reqBody))
			}
			return daIssue, err
		}

		if daIssue != nil {
			fmt.Printf("Issue cloned; see %s\n",
				fmt.Sprintf(filepath.Join(conn.baseUri, "browse/%s"), daIssue.Key))
		}
		// Add remote link to the upstream issue
		if _, _, err = conn.client.Issue.AddRemoteLink(daIssue.ID, &gojira.RemoteLink{
			Object: &gojira.RemoteLinkObject{
				URL:   getWebURL(fromIssue.GetURL()),
				Title: fmt.Sprintf("Upstream Issue #%v", fromIssue.GetNumber()),
			},
		}); err != nil {
			return nil, err
		}
	}

	return daIssue, nil
}
