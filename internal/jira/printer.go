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
	"context"
	"fmt"

	gojira "github.com/andygrunwald/go-jira"
)

func PrintJiraIssue(ctx context.Context, jc *Connection, jiraIssue gojira.Issue) {
	fmt.Printf("%s (%s/%s): %+v -> %s\n", jiraIssue.Key, jiraIssue.Fields.Type.Name, jiraIssue.Fields.Priority.Name, jiraIssue.Fields.Summary, jiraIssue.Fields.Status.Name)
	if jiraIssue.Fields.Assignee != nil {
		fmt.Printf("\tAssignee : %v\n", jiraIssue.Fields.Assignee.DisplayName)
	} else {
		fmt.Printf("\tAssignee : Unassigned\n")
	}
	fmt.Printf("\tReporter: %v\n", jiraIssue.Fields.Reporter.DisplayName)
	fmt.Printf("\tSummary: %s\n", jiraIssue.Fields.Summary)
	rlinks, response, err := jc.Client.Issue.GetRemoteLinksWithContext(ctx, jiraIssue.Key)
	if err != nil {
		return
	}
	defer response.Body.Close()
	fmt.Printf("\tLinks:\n")
	for _, rlink := range *rlinks {
		fmt.Printf("\t\t%s\n", rlink.Object.URL)
	}
	fmt.Println("")
}

func PrintJiraIssues(ctx context.Context, jc *Connection, jiraIssues []gojira.Issue) {
	for _, ji := range jiraIssues {
		PrintJiraIssue(ctx, jc, ji)
	}
}
