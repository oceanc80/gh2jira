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
package root

import (
	"fmt"

	"github.com/oceanc80/gh2jira/internal/config"
	"github.com/oceanc80/gh2jira/internal/gh"
	"github.com/oceanc80/gh2jira/internal/jira"
	"github.com/oceanc80/gh2jira/internal/reconcile"
	"github.com/oceanc80/gh2jira/pkg/util"
	"github.com/spf13/cobra"
)

func NewReconcileCmd() *cobra.Command {
	runCmd := &cobra.Command{
		Use:   "reconcile",
		Short: "reconcile github and jira issues",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ff, err := util.NewFlagFeeder(cmd)
			if err != nil {
				return err
			}

			config := config.NewConfig(ff)
			err = config.Read()
			if err != nil {
				return err
			}

			if config.JiraProject == "" {
				return fmt.Errorf("must specify jira project")
			}
			jql := fmt.Sprintf("project=%s and status != Closed", config.JiraProject)

			gc, err := gh.NewConnection(gh.WithContext(cmd.Context()), gh.WithToken(config.Tokens.GithubToken))
			if err != nil {
				return err
			}
			err = gc.Connect()
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

			err = reconcile.Reconcile(cmd.Context(), jql, jc, gc)
			if err != nil {
				return err
			}

			return nil
		},
	}

	return runCmd
}
