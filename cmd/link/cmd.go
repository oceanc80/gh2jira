package link

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/oceanc80/gh2jira/pkg/config"
	"github.com/oceanc80/gh2jira/pkg/jira"
	"github.com/oceanc80/gh2jira/pkg/util"
)

type linker interface {
	Link(context.Context, string, []string, string, bool) ([]jira.LinkResult, error)
}

// NewCmd links public GitHub issue or pull request URLs to an existing Jira item.
func NewCmd() *cobra.Command {
	return newCmd(func(cmd *cobra.Command) (linker, error) {
		ff, err := util.NewFlagFeeder(cmd)
		if err != nil {
			return nil, err
		}
		cfg := config.NewConfig(ff)
		if err := cfg.Read(); err != nil {
			return nil, err
		}
		return jira.NewConnection(jira.WithBaseURI(cfg.JiraBaseUrl), jira.WithAuth(cfg.Tokens.JiraAuth))
	})
}

func newCmd(loadClient func(*cobra.Command) (linker, error)) *cobra.Command {
	var dryRun bool
	var relationship string
	cmd := &cobra.Command{
		Use:   "link <JIRA_KEY> <GITHUB_URL> [GITHUB_URL ...]",
		Short: "Link existing GitHub issues or pull requests to an existing Jira item",
		Long:  "Add Jira remote links across GitHub repositories without copying descriptions or comments. GitHub is never modified. Dry runs read Jira links but do not write.",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.MinimumNArgs(2)(cmd, args); err != nil {
				return err
			}
			_, err := jira.ValidateLinkArguments(args[0], args[1:], relationship)
			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := loadClient(cmd)
			if err != nil {
				return err
			}
			results, err := client.Link(cmd.Context(), args[0], args[1:], relationship, dryRun)
			// Also show successful links preceding a write failure.
			for _, result := range results {
				if _, outputErr := fmt.Fprintf(cmd.OutOrStdout(), "%s: %s %s %s\n", result.Status, args[0], result.Relationship, result.URL); outputErr != nil {
					return errors.Join(err, outputErr)
				}
			}
			return err
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dryrun", false, "show planned links without writing to Jira")
	cmd.Flags().StringVar(&relationship, "relationship", jira.DefaultRelationship, "relationship: relates to, implements, fixes, or depends on (from the Jira item to the GitHub URL)")
	return cmd
}
