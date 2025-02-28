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
package reconcile

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	gojira "github.com/andygrunwald/go-jira"
	"github.com/oceanc80/gh2jira/pkg/gh"
	"github.com/oceanc80/gh2jira/pkg/jira"
	"github.com/oceanc80/gh2jira/pkg/workflow"
)

const unassigned_issue string = "unassigned"

type IssueStatus struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Assignee string `json:"assignee"`
}

type PairResult struct {
	Jira IssueStatus `json:"jira"`
	Git  IssueStatus `json:"github"`
}

type PairResults []PairResult
type TypeResults struct {
	Matches    PairResults `json:"matches"`
	Mismatches PairResults `json:"mismatches"`
}

type Outcome string

const (
	OutcomeMatch    Outcome = "MATCH"
	OutcomeMismatch Outcome = "MISMATCH"
)

func Reconcile(ctx context.Context, jql string, jc *jira.Connection, gc *gh.Connection) (*TypeResults, error) {
	results := &TypeResults{
		Matches:    make(PairResults, 0),
		Mismatches: make(PairResults, 0),
	}

	if jc == nil || gc == nil {
		return nil, errors.New("nil connection")
	}

	// compile a regex to match github issue URLs
	r, err := regexp.Compile(".*/github.com/.*/issues/([0-9]+)")
	if err != nil {
		return nil, err
	}

	jiraIssues, err := jc.SearchIssues(jql)
	if err != nil {
		return nil, err
	}

	// reduce the list to just those jira which have a github issue link
	jiraIssues = slices.DeleteFunc(jiraIssues, func(issue gojira.Issue) bool {
		rlinks, response, err := jc.Client.Issue.GetRemoteLinksWithContext(ctx, issue.Key)
		if err != nil {
			return false
		}
		defer response.Body.Close()
		if rlinks != nil {
			found := false
			for _, rlink := range *rlinks {
				if r.MatchString(rlink.Object.URL) {
					found = true
				}
			}
			return !found
		}
		return false
	})

	// jira.PrintJiraIssues(ctx, jc, jiraIssues)

	err = workflow.ReadWorkflows()
	if err != nil {
		return nil, err
	}

	// eval status of each jira and linked github issues for mismatch
	for _, ji := range jiraIssues {
		jstat := ji.Fields.Status.Name
		rlinks, response, err := jc.Client.Issue.GetRemoteLinksWithContext(ctx, ji.Key)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		for _, rlink := range *rlinks {
			if r.MatchString(rlink.Object.URL) {
				project, issue, err := splitIssueRef(rlink.Object.URL)
				if err != nil {
					return nil, err
				}
				// fmt.Printf("\tproject %q issue #%d\n", project, issue)
				gi, err := gc.GetIssue(issue, gh.WithProject(project))
				if err != nil {
					return nil, err
				}
				stateMatch, err := workflow.ValidateState(gi.GetState(), jstat)
				if err != nil {
					return nil, err
				}
				var ghAssignee string = unassigned_issue
				if gi.GetAssignee() != nil {
					ghAssignee = *gi.GetAssignee().Login
				}
				var jiAssignee string = unassigned_issue
				if ji.Fields.Assignee != nil {
					jiAssignee = ji.Fields.Assignee.DisplayName
				}

				pair := PairResult{
					Jira: IssueStatus{Name: ji.Key, Status: jstat, Assignee: jiAssignee},
					Git:  IssueStatus{Name: fmt.Sprintf("%s/%d", project, gi.GetNumber()), Status: gi.GetState(), Assignee: ghAssignee},
				}
				if stateMatch {
					results.Matches = append(results.Matches, pair)
				} else {
					results.Mismatches = append(results.Mismatches, pair)
				}
			}
		}
	}

	return results, nil
}

func splitIssueRef(ref string) (string, int, error) {
	// split the ref into project (owner/repo), and issue number
	s := strings.Split(ref, "/")
	if len(s) <= 4 {
		return "", 0, fmt.Errorf("unable to extract issue attributes from URL: %v", ref)
	}

	// URLs to this point have been manicured to follow the general schema
	// https://www.github.com/(owner/repo)/issues/(num)
	//       project...........^^^^^^^^^^
	//       issue number..........................^^^
	// split on '/', their offsets into the resulting slice are
	//                         len(s)-4  len(s)-3  len(s)-1
	project := fmt.Sprintf("%s/%s", s[len(s)-4], s[len(s)-3])
	num, err := strconv.Atoi(s[len(s)-1])
	if err != nil {
		return "", 0, err
	}

	return project, num, nil
}
