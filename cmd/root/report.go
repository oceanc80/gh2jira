package root

import (
	"github.com/oceanc80/gh2jira/pkg/config"
	"github.com/oceanc80/gh2jira/pkg/gh"
	"github.com/oceanc80/gh2jira/pkg/jira"
	"github.com/oceanc80/gh2jira/pkg/report"
	"github.com/oceanc80/gh2jira/pkg/util"
	"github.com/spf13/cobra"
)

// NewReportCmd constructs the read-only, project-scoped correlation command.
func NewReportCmd() *cobra.Command {
	return newReportCmd(loadReportConfig, runReport)
}

type reportConfigLoader func(*util.FlagFeeder) (*config.Config, error)
type reportRunner func(*cobra.Command, *config.Config, string) ([]report.Row, error)

func newReportCmd(load reportConfigLoader, run reportRunner) *cobra.Command {
	var output, query string
	cmd := &cobra.Command{
		Use: "report", Short: "Report read-only GitHub/Jira correlations across repositories",
		Long: "Report independent Jira and GitHub states, including closed Jira items. Requires --jira-project or a --profile-name supplying a Jira project. No workflow mapping or mutations are performed.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := report.ValidateFormat(output); err != nil {
				return err
			}
			ff, err := util.NewFlagFeeder(cmd)
			if err != nil {
				return err
			}
			project := ff.JiraProject
			if project == "" && ff.ProfileName != "" {
				// Validate the raw query now; the profile project is validated
				// after local configuration is resolved, before any readers.
				project = "PROFILE"
			}
			if _, err := report.Scope(project, query); err != nil {
				return err
			}
			cfg, err := load(ff)
			if err != nil {
				return err
			}
			if _, err := report.Scope(cfg.JiraProject, query); err != nil {
				return err
			}
			rows, err := run(cmd, cfg, query)
			if err != nil {
				return err
			}
			return report.Write(cmd.OutOrStdout(), output, rows)
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "json", "output format (json, yaml, table)")
	cmd.Flags().StringVar(&query, "query", "", "JQL filter expression ANDed with the resolved Jira project (no ORDER BY)")
	return cmd
}

func loadReportConfig(ff *util.FlagFeeder) (*config.Config, error) {
	cfg := config.NewConfig(ff)
	// The general configuration's default project is not an explicit scope.
	cfg.JiraProject = ""
	if err := cfg.Read(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func runReport(cmd *cobra.Command, cfg *config.Config, query string) ([]report.Row, error) {
	gc, err := gh.NewConnection(gh.WithContext(cmd.Context()), gh.WithToken(cfg.Tokens.GithubToken))
	if err != nil {
		return nil, err
	}
	if err := gc.Connect(); err != nil {
		return nil, err
	}
	jc, err := jira.NewConnection(jira.WithBaseURI(cfg.JiraBaseUrl), jira.WithAuth(cfg.Tokens.JiraAuth))
	if err != nil {
		return nil, err
	}
	if err := jc.Connect(); err != nil {
		return nil, err
	}
	return report.Generate(cmd.Context(), cfg.JiraProject, query, jc, gc)
}
