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

package clone

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/go-github/v60/github"
	"github.com/spf13/cobra"

	"github.com/oceanc80/gh2jira/pkg/config"
	"github.com/oceanc80/gh2jira/pkg/gh"
	"github.com/oceanc80/gh2jira/pkg/jira"
	"github.com/oceanc80/gh2jira/pkg/util"
)

type issueGetter interface {
	GetIssue(int, ...gh.ListOption) (*github.Issue, error)
}

type issueCloner interface {
	Clone(context.Context, *github.Issue, string, string, bool, ...jira.CloneOption) (*jira.Issue, error)
}

type loadDependencies func(*cobra.Command) (*config.Config, issueGetter, issueCloner, error)

func connect(cmd *cobra.Command) (*config.Config, issueGetter, issueCloner, error) {
	ff, err := util.NewFlagFeeder(cmd)
	if err != nil {
		return nil, nil, nil, err
	}
	cfg := config.NewConfig(ff)
	if err := cfg.Read(); err != nil {
		return nil, nil, nil, err
	}
	gc, err := gh.NewConnection(gh.WithContext(cmd.Context()), gh.WithToken(cfg.Tokens.GithubToken))
	if err != nil {
		return nil, nil, nil, err
	}
	if err := gc.Connect(); err != nil {
		return nil, nil, nil, err
	}
	jc, err := jira.NewConnection(jira.WithBaseURI(cfg.JiraBaseUrl), jira.WithAuth(cfg.Tokens.JiraAuth))
	if err != nil {
		return nil, nil, nil, err
	}
	if err := jc.Connect(); err != nil {
		return nil, nil, nil, err
	}
	return cfg, gc, jc, nil
}

func sourceIDs(args []string) ([]int, error) {
	ids := make([]int, 0, len(args))
	for _, arg := range args {
		valid := arg != ""
		for _, digit := range arg {
			if digit < '0' || digit > '9' {
				valid = false
			}
		}
		id, err := strconv.Atoi(arg)
		if !valid || err != nil || id <= 0 {
			return nil, fmt.Errorf("invalid GitHub source ID %q: expected a positive decimal integer", arg)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func NewCmd() *cobra.Command {
	return newCmd(connect)
}

func newCmd(load loadDependencies) *cobra.Command {
	var dryRun, ifMissing bool
	cmd := &cobra.Command{
		Use:   "clone <ISSUE_ID> [ISSUE_ID ...]",
		Short: "Clone given Github issues to Jira",
		Long: `Clone given Github issues to Jira.
WARNING! This will write to your jira instance. Use --dryrun to see what will happen`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			ids, err := sourceIDs(args)
			if err != nil {
				return err
			}

			cfg, gc, jc, err := load(cmd)
			if err != nil {
				return err
			}

			options := []jira.CloneOption{jira.WithCloneWriter(cmd.OutOrStdout())}
			if ifMissing {
				options = append(options, jira.WithIfMissing())
			}
			for _, id := range ids {
				issue, err := gc.GetIssue(id, gh.WithProject(cfg.GithubProject))
				if err != nil {
					return err
				}

				_, err = jc.Clone(cmd.Context(), issue, cfg.JiraProject, cfg.JiraIssueType, dryRun, options...)
				if err != nil {
					return err
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dryrun", false, "display what would happen without taking actually doing it")
	cmd.Flags().BoolVar(&ifMissing, "if-missing", false, "reuse a unique existing source relationship, or create and link if missing")

	return cmd
}
