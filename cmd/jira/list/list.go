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

package list

import (
	"fmt"

	"github.com/spf13/cobra"

	gojira "github.com/andygrunwald/go-jira"
	"github.com/oceanc80/gh2jira/pkg/config"
	"github.com/oceanc80/gh2jira/pkg/jira"
	"github.com/oceanc80/gh2jira/pkg/util"
)

var (
	query string
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List open Jira issues",
		Long:  "List open Jira issues filtered with optional additional JQL",
		RunE: func(cmd *cobra.Command, args []string) error {
			ff, err := util.NewFlagFeeder(cmd)
			if err != nil {
				return err
			}
			config := config.NewConfig(ff)
			err = config.Read()
			if err != nil {
				return err
			}

			jc, err := jira.NewConnection(
				jira.WithBaseURI(config.JiraBaseUrl),
				jira.WithAuthToken(config.Tokens.JiraToken),
			)
			if err != nil {
				return err
			}

			err = jc.Connect()
			if err != nil {
				return err
			}

			jql := ""
			switch {
			case config.JiraProject == "" && query == "":
				return fmt.Errorf("must provide either project or query")
			case config.JiraProject != "" && query != "":
				jql = "project=" + config.JiraProject + " AND " + query
			case config.JiraProject != "":
				jql = "project=" + config.JiraProject
			default:
				jql = query
			}
			jql += " and status != Closed"

			var result []gojira.Issue
			result, err = jc.SearchIssues(jql)
			if err != nil {
				return err
			}
			for _, i := range result {
				fmt.Printf("%s (%s/%s): %+v -> %s\n", i.Key, i.Fields.Type.Name, i.Fields.Priority.Name, i.Fields.Summary, i.Fields.Status.Name)
				if i.Fields.Assignee != nil {
					fmt.Printf("Assignee : %v\n", i.Fields.Assignee.DisplayName)
				} else {
					fmt.Printf("Assignee : Unassigned\n")
				}
				fmt.Printf("Reporter: %v\n", i.Fields.Reporter.DisplayName)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&query, "query", "", "Jira query (if provided, ANDed with project)")
	return cmd
}
