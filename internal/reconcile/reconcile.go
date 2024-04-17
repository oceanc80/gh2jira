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
	"github.com/oceanc80/gh2jira/internal/gh"
	"github.com/oceanc80/gh2jira/internal/jira"
	"github.com/oceanc80/gh2jira/internal/workflow"
)

const greenStart string = "\033[32m"
const yellowStart string = "\033[33m"
const redStart string = "\033[31m"
const colorReset string = "\033[0m"

func Reconcile(ctx context.Context, jql string, jc *jira.Connection, gc *gh.Connection) error {
	if jc == nil || gc == nil {
		return errors.New("nil connection")
	}

	// compile a regex to match github issue URLs
	r, err := regexp.Compile(".*/github.com/.*/issues/([0-9]+)")
	if err != nil {
		return err
	}

	jiraIssues, err := jc.SearchIssues(jql)
	if err != nil {
		return err
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
		return err
	}

	// eval status of each jira and linked github issues for mismatch
	for _, ji := range jiraIssues {
		jstat := ji.Fields.Status.Name
		rlinks, response, err := jc.Client.Issue.GetRemoteLinksWithContext(ctx, ji.Key)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		for _, rlink := range *rlinks {
			if r.MatchString(rlink.Object.URL) {
				project, issue, err := splitIssueRef(rlink.Object.URL)
				if err != nil {
					return err
				}
				// fmt.Printf("\tproject %q issue #%d\n", project, issue)
				gi, err := gc.GetIssue(issue, gh.WithProject(project))
				if err != nil {
					return err
				}
				stateMatch, err := workflow.ValidateState(gi.GetState(), jstat)
				if err != nil {
					return err
				}
				if stateMatch {
					fmt.Printf("%s%s/(%s/%d)%s status (g: %q\tj: %q)\t%sMATCH%s\n",
						yellowStart, ji.Key, project, gi.GetNumber(), colorReset, gi.GetState(), jstat, greenStart, colorReset)
				} else {
					fmt.Printf("%s%s/(%s/%d)%s status (g: %q,\tj: %q)\t%sMISMATCH%s\n",
						yellowStart, ji.Key, project, gi.GetNumber(), colorReset, gi.GetState(), jstat, redStart, colorReset)
				}
				// fmt.Printf("\tgithub issue #%d status %v\n", gi.GetNumber(), gi.GetState())
			}
		}
	}

	return nil
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
