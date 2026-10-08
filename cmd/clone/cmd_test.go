package clone

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/google/go-github/v60/github"
	"github.com/spf13/cobra"

	"github.com/oceanc80/gh2jira/pkg/config"
	"github.com/oceanc80/gh2jira/pkg/gh"
	"github.com/oceanc80/gh2jira/pkg/jira"
)

func TestBatchValidationBeforeDependencies(t *testing.T) {
	for _, flag := range []string{"", "--if-missing"} {
		for _, bad := range []string{"0", "-1", "+1", "1.5", "", " 1", "one", "999999999999999999999999999999", "１２"} {
			t.Run(flag+"/"+bad, func(t *testing.T) {
				loads := 0
				cmd := newCmd(func(*cobra.Command) (*config.Config, issueGetter, issueCloner, error) {
					loads++
					return nil, nil, nil, errors.New("configuration must not be read")
				})
				cmd.SetOut(io.Discard)
				cmd.SetErr(io.Discard)
				args := []string{}
				if flag != "" {
					args = append(args, flag)
				}
				cmd.SetArgs(append(args, "--", "42", bad))
				err := cmd.Execute()
				if err == nil || !strings.Contains(err.Error(), "positive decimal integer") || loads != 0 {
					t.Fatalf("batch validation: err=%v dependency loads=%d", err, loads)
				}
			})
		}
	}
}

type fakeGetter struct{ gets int }

func (f *fakeGetter) GetIssue(number int, _ ...gh.ListOption) (*github.Issue, error) {
	f.gets++
	return &github.Issue{Number: github.Int(number), HTMLURL: github.String("https://github.com/org/repo/issues/42")}, nil
}

type failingJira struct {
	createErr               error
	creates, adds, searches int
}

func (f *failingJira) SearchIssues(context.Context, string) ([]jira.Issue, error) {
	f.searches++
	return nil, nil
}
func (f *failingJira) CreateIssue(context.Context, *jira.Issue) (*jira.Issue, error) {
	f.creates++
	return nil, f.createErr
}
func (f *failingJira) AddRemoteLink(context.Context, string, *jira.RemoteLink) error {
	f.adds++
	return nil
}
func (*failingJira) GetRemoteLinks(context.Context, string) ([]jira.RemoteLink, error) {
	return nil, nil
}

func TestCreationErrorReturnedFromCLI(t *testing.T) {
	for _, ifMissing := range []bool{false, true} {
		t.Run(map[bool]string{false: "legacy", true: "if-missing"}[ifMissing], func(t *testing.T) {
			wantErr := errors.New("synthetic Jira creation failure")
			client := &failingJira{createErr: wantErr}
			getter := &fakeGetter{}
			loads := 0
			cmd := newCmd(func(*cobra.Command) (*config.Config, issueGetter, issueCloner, error) {
				loads++
				conn, err := jira.NewConnection(
					jira.WithBaseURI("https://jira.invalid"), jira.WithAuth(config.JiraAuth{Token: "synthetic"}), jira.WithJiraClient(client),
				)
				return &config.Config{GithubProject: "org/repo", JiraProject: "DST", JiraIssueType: "Bug"}, getter, conn, err
			})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			args := []string{"42", "43"}
			if ifMissing {
				args = append([]string{"--if-missing"}, args...)
			}
			cmd.SetArgs(args)
			err := cmd.Execute()
			if !errors.Is(err, wantErr) || loads != 1 || getter.gets != 1 || client.creates != 1 || client.adds != 0 {
				t.Fatalf("creation failure swallowed: err=%v loads=%d gets=%d creates=%d adds=%d", err, loads, getter.gets, client.creates, client.adds)
			}
			if ifMissing != (client.searches == 1) {
				t.Fatalf("if-missing flag not forwarded: searches=%d", client.searches)
			}
		})
	}
}

func TestSourceIDs(t *testing.T) {
	ids, err := sourceIDs([]string{"1", "0042", "123"})
	if err != nil || len(ids) != 3 || ids[0] != 1 || ids[1] != 42 || ids[2] != 123 {
		t.Fatalf("valid IDs: %v, %v", ids, err)
	}
}
