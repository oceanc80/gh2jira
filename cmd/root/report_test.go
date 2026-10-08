package root

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/oceanc80/gh2jira/pkg/config"
	"github.com/oceanc80/gh2jira/pkg/report"
	"github.com/oceanc80/gh2jira/pkg/util"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func reportRoot(run reportRunner) *cobra.Command {
	return reportRootWithLoader(func(ff *util.FlagFeeder) (*config.Config, error) {
		return &config.Config{JiraProject: ff.JiraProject}, nil
	}, run)
}

func reportRootWithLoader(load reportConfigLoader, run reportRunner) *cobra.Command {
	root := NewCmd()
	for _, child := range root.Commands() {
		if child.Name() == "report" {
			root.RemoveCommand(child)
		}
	}
	root.AddCommand(newReportCmd(load, run))
	root.SilenceErrors = true
	root.SilenceUsage = true
	return root
}

func TestReportFormats(t *testing.T) {
	for _, format := range []string{"json", "yaml", "table"} {
		t.Run(format, func(t *testing.T) {
			calls := 0
			root := reportRoot(func(_ *cobra.Command, cfg *config.Config, query string) ([]report.Row, error) {
				calls++
				require.Equal(t, "TEST", cfg.JiraProject)
				require.Equal(t, "status = Open", query)
				return []report.Row{{Jira: report.JiraItem{Key: "TEST-1", Status: "Open"}, LinkStatus: "no_supported_github_links"}}, nil
			})
			var output bytes.Buffer
			root.SetOut(&output)
			root.SetArgs([]string{"report", "--jira-project", "TEST", "--query", "status = Open", "--output", format})
			require.NoError(t, root.Execute())
			require.Equal(t, 1, calls)
			require.Contains(t, output.String(), "TEST-1")
		})
	}
}

func TestReportValidationBeforeConfig(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"--jira-project", "TEST", "--output", "xml"}, "unsupported report output"},
		{nil, "Jira project key"},
		{[]string{"--profile-name", "should-not-be-read", "--output", "xml"}, "unsupported report output"},
		{[]string{"--profile-name", "should-not-be-read", "--query", "status = Open)"}, "unbalanced parentheses"},
		{[]string{"--jira-project", "TEST OR project = OTHER"}, "Jira project key"},
		{[]string{"--jira-project", "TEST", "--query", "status = Open) OR project = OTHER"}, "unbalanced parentheses"},
		{[]string{"--jira-project", "TEST", "--query", "status = Open ORDER BY key"}, "without ORDER BY"},
		{[]string{"unexpected"}, "unknown command"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			root := NewCmd()
			root.SilenceErrors = true
			root.SilenceUsage = true
			args := []string{"report", "--token-file", "nonexistent-report-test-tokenstore.yaml"}
			root.SetArgs(append(args, tc.args...))
			err := root.Execute()
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
		})
	}
}

type failingReportWriter struct{ err error }

func (w failingReportWriter) Write([]byte) (int, error) { return 0, w.err }

func TestReportPropagatesRunnerAndOutputErrors(t *testing.T) {
	failure := errors.New("report failure")
	for _, outputFailure := range []bool{false, true} {
		root := reportRoot(func(*cobra.Command, *config.Config, string) ([]report.Row, error) {
			if !outputFailure {
				return nil, failure
			}
			return []report.Row{}, nil
		})
		var output bytes.Buffer
		root.SetOut(&output)
		if outputFailure {
			root.SetOut(failingReportWriter{failure})
		}
		root.SetArgs([]string{"report", "--jira-project", "TEST"})
		require.ErrorIs(t, root.Execute(), failure)
		require.Empty(t, output.String())
	}
}

func TestReportProfileScope(t *testing.T) {
	// Synthetic, credential-free local configuration; never read source configs.
	profiles := filepath.Join(t.TempDir(), "profiles.yaml")
	require.NoError(t, os.WriteFile(profiles, []byte(`profiles:
- description: scoped
  jiraConfig:
    project: PROFILE_1
- description: empty
  jiraConfig:
    project: ""
- description: unsafe
  jiraConfig:
    project: 'PROJECT OR project = OTHER'
`), 0600))
	for _, tc := range []struct {
		name, profile, override, wantProject string
		wantErr                              bool
	}{
		{"profile provided project", "scoped", "", "PROFILE_1", false},
		{"explicit override", "scoped", "OVERRIDE", "OVERRIDE", false},
		{"missing profile project", "empty", "", "", true},
		{"unsafe profile project", "unsafe", "", "", true},
		{"unknown profile", "absent", "", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			root := reportRootWithLoader(loadReportConfig, func(_ *cobra.Command, cfg *config.Config, query string) ([]report.Row, error) {
				calls++
				require.Equal(t, tc.wantProject, cfg.JiraProject)
				scope, err := report.Scope(cfg.JiraProject, query)
				require.NoError(t, err)
				require.Equal(t, `project = "`+tc.wantProject+`" AND (status = Open OR status = Closed)`, scope)
				return []report.Row{}, nil
			})
			var output bytes.Buffer
			root.SetOut(&output)
			args := []string{"report", "--profile-name", tc.profile, "--profiles-file", profiles, "--token-file", "", "--query", "status = Open OR status = Closed"}
			if tc.override != "" {
				args = append(args, "--jira-project", tc.override)
			}
			root.SetArgs(args)
			err := root.Execute()
			if tc.wantErr {
				require.Error(t, err)
				require.Zero(t, calls)
				require.Empty(t, output.String())
			} else {
				require.NoError(t, err)
				require.Equal(t, 1, calls)
			}
		})
	}
}

func TestReportNeverFallsBackToDefaultScope(t *testing.T) {
	cfg, err := loadReportConfig(&util.FlagFeeder{})
	require.NoError(t, err)
	require.Empty(t, cfg.JiraProject)
	_, err = report.Scope(cfg.JiraProject, "")
	require.Error(t, err)
}
