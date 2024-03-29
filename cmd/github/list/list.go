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

package list

import (
	"github.com/spf13/cobra"

	"github.com/oceanc80/gh2jira/internal/config"
	"github.com/oceanc80/gh2jira/internal/gh"
	"github.com/oceanc80/gh2jira/pkg/util"
)

var (
	milestone string
	assignee  string
	label     []string
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Github issues",
		Long:  "List Github issues filtered by milestone, assignee, or label",
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

			gc, err := gh.NewConnection(gh.WithContext(cmd.Context()), gh.WithToken(config.Tokens.GithubToken))
			if err != nil {
				return err
			}
			err = gc.Connect()
			if err != nil {
				return err
			}

			issues, err := gc.ListIssues(
				gh.WithMilestone(milestone),
				gh.WithAssignee(assignee),
				gh.WithProject(config.GithubProject),
				gh.WithLabels(label...),
			)
			if err != nil {
				return err
			}

			// print the issues
			for _, issue := range issues {
				if issue.IsPullRequest() {
					// We have a PR, skipping
					continue
				}
				gh.PrintGithubIssue(issue, true, true)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&milestone, "milestone", "",
		"the milestone ID from the url, not the display name")
	cmd.Flags().StringVar(&assignee, "assignee", "", "username assigned the issue")
	cmd.Flags().StringSliceVar(&label, "label", nil,
		"label i.e. --label \"documentation,bug\" or --label doc --label bug (default: none)")

	return cmd
}
