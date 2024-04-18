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
	gojira "github.com/andygrunwald/go-jira"
)

// SearchIssues will query Jira API using the provided JQL string
func (c *Connection) SearchIssues(jql string) ([]gojira.Issue, error) {

	// fmt.Printf("Querying Jira with JQL: %s\n", jql)

	// lastIssue is the index of the last issue returned
	lastIssue := 0
	// Make a loop through amount of issues
	var result []gojira.Issue
	for {
		// Add a Search option which accepts maximum amount (1000)
		opt := &gojira.SearchOptions{
			MaxResults: 1000,      // Max amount
			StartAt:    lastIssue, // Make sure we start grabbing issues from last checkpoint
		}
		issues, resp, err := c.Client.Issue.Search(jql, opt)
		if err != nil {
			return nil, err
		}
		// Grab total amount from response
		total := resp.Total
		if issues == nil {
			// init the issues array with the correct amount of length
			result = make([]gojira.Issue, 0, total)
		}

		// Append found issues to result
		result = append(result, issues...)
		// Update checkpoint index by using the response StartAt variable
		lastIssue = resp.StartAt + len(issues)
		// Check if we have reached the end of the issues
		if lastIssue >= total {
			break
		}
	}

	return result, nil
}
