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
	"github.com/spf13/cobra"

	"github.com/oceanc80/gh2jira/cmd/clone"
	"github.com/oceanc80/gh2jira/cmd/github"
	"github.com/oceanc80/gh2jira/cmd/jira"
)

const defaultTokensFile string = "tokenstore.yaml"
const defaultProfilesFile string = "profiles.yaml"
const defaultJiraBaseURL string = "https://issues.redhat.com/"

var (
	tokensFile   string
	profilesFile string
	profileName  string
	ghProject    string
	jProject     string
	jUrl         string
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gh2jira",
		Short: "github <--> jira issue reconciler",
		Long:  "",
		Run:   func(_ *cobra.Command, _ []string) {}, // adding an empty function here to preserve non-zero exit status for misstated subcommands/flags for the command hierarchy
	}
	// add the child commands
	cmd.AddCommand(github.NewCmd())
	cmd.AddCommand(jira.NewCmd())
	cmd.AddCommand(clone.NewCmd())
	cmd.AddCommand(NewReconcileCmd())

	cmd.PersistentFlags().StringVar(&tokensFile, "token-file", defaultTokensFile, "file containing authentication tokens, if different than profile")
	cmd.PersistentFlags().StringVar(&profilesFile, "profiles-file", defaultProfilesFile, "filename containing optional profile attributes")

	// profile / project names must not have default values since they will always be used as if they were user-specified values, overriding all default values given
	cmd.PersistentFlags().StringVar(&profileName, "profile-name", "", "profile name to use (implies profiles-file)")
	cmd.PersistentFlags().StringVar(&ghProject, "github-project", "", "Github project domain to list if not using a profile, e.g.: operator-framework/operator-lifecycle-manager")
	cmd.PersistentFlags().StringVar(&jProject, "jira-project", "", "Jira project if not using a profile, e.g.: OCPBUGS")
	cmd.PersistentFlags().StringVar(&jUrl, "jira-base-url", defaultJiraBaseURL, "Jira base URL, e.g.: https://issues.redhat.com")

	return cmd
}
