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
	"encoding/json"
	"fmt"
	"os"

	"github.com/oceanc80/gh2jira/internal/config"
	"github.com/oceanc80/gh2jira/internal/gh"
	"github.com/oceanc80/gh2jira/internal/jira"
	"github.com/oceanc80/gh2jira/internal/reconcile"
	"github.com/oceanc80/gh2jira/pkg/util"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

var porcelain bool
var output string = "json"

const (
	greenStart  string = "\033[32m"
	yellowStart string = "\033[33m"
	redStart    string = "\033[31m"
	colorReset  string = "\033[0m"
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

			if output != "yaml" && output != "json" {
				return fmt.Errorf("invalid output format %q (accepted formats are 'yaml', 'json')", output)
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

			results, err := reconcile.Reconcile(cmd.Context(), jql, jc, gc)
			if err != nil {
				return err
			}

			if porcelain {
				b, _ := json.MarshalIndent(results, "", "  ")
				if output == "json" {
					fmt.Println(string(b))
				} else {
					yamlData, err := yaml.JSONToYAML(b)
					if err != nil {
						return err
					}
					yamlData = append([]byte("---\n"), yamlData...)
					_, err = os.Stdout.Write(yamlData)
					if err != nil {
						return err
					}
				}
			} else {
				for _, pair := range results {
					if pair.Result == reconcile.ResultMatch {
						fmt.Printf("%s%s/(%s)%s status (g: %q\tj: %q)\t%sMATCH%s\n",
							yellowStart, pair.Jira.Name, pair.Git.Name, colorReset, pair.Git.Status, pair.Jira.Status, greenStart, colorReset)
					} else {
						fmt.Printf("%s%s/(%s)%s status (g: %q,\tj: %q)\t%sMISMATCH%s\n",
							yellowStart, pair.Jira.Name, pair.Git.Name, colorReset, pair.Git.Status, pair.Jira.Status, redStart, colorReset)
					}
				}
			}

			return nil
		},
	}

	runCmd.Flags().BoolVar(&porcelain, "porcelain", false, "display output in an easy-to-parse format for scripts")
	runCmd.Flags().StringVarP(&output, "output", "o", "json", "output format for porcelain display (json or yaml)")

	return runCmd
}
