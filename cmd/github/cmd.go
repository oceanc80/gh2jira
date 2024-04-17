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
package github

import (
	"github.com/spf13/cobra"

	"github.com/oceanc80/gh2jira/cmd/github/list"
)

func NewCmd() *cobra.Command {
	runCmd := &cobra.Command{
		Use:   "github",
		Short: "Run a github subcommand",
		Args:  cobra.NoArgs,
		Run:   func(_ *cobra.Command, _ []string) {}, // adding an empty function here to preserve non-zero exit status for misstated subcommands/flags for the command hierarchy
	}

	runCmd.AddCommand(list.NewCmd())

	return runCmd
}
