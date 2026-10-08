package report

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/google/go-github/v60/github"
	"github.com/oceanc80/gh2jira/pkg/gh"
	"github.com/oceanc80/gh2jira/pkg/jira"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

type fakeJira struct {
	issues    []jira.Issue
	links     map[string][]jira.RemoteLink
	jql       string
	searchErr error
	linksErr  error
	mutations int
}

func (f *fakeJira) SearchIssues(_ context.Context, jql string) ([]jira.Issue, error) {
	f.jql = jql
	return f.issues, f.searchErr
}

func (f *fakeJira) GetRemoteLinks(_ context.Context, key string) ([]jira.RemoteLink, error) {
	return f.links[key], f.linksErr
}

// Include mutation methods as tripwires; none belongs to the report interface.
func (f *fakeJira) CreateIssue(context.Context, *jira.Issue) (*jira.Issue, error) {
	f.mutations++
	return nil, errors.New("unexpected mutation")
}

func (f *fakeJira) AddRemoteLink(context.Context, string, *jira.RemoteLink) error {
	f.mutations++
	return errors.New("unexpected mutation")
}

type fakeGitHub struct {
	issues  map[string]*github.Issue
	pulls   map[string]*github.PullRequest
	reads   map[string]int
	err     error
	pullErr error
}

func (f *fakeGitHub) GetIssue(number int, opts ...gh.ListOption) (*github.Issue, error) {
	var spec gh.ListSpec
	for _, opt := range opts {
		if err := opt(&spec); err != nil {
			return nil, err
		}
	}
	key := (reference{owner: spec.GetGithubOrg(), repo: spec.GetGithubRepo(), number: number}).key()
	f.reads["issue:"+key]++
	return f.issues[key], f.err
}

func (f *fakeGitHub) GetPullRequest(_ context.Context, owner, repo string, number int) (*github.PullRequest, error) {
	key := (reference{owner: owner, repo: repo, number: number}).key()
	f.reads["pull:"+key]++
	if f.pullErr != nil {
		return nil, f.pullErr
	}
	return f.pulls[key], f.err
}

func remote(raw string) jira.RemoteLink {
	return jira.RemoteLink{Object: &jira.RemoteLinkObject{URL: raw, Title: "PRIVATE CUSTOMER"}}
}

func TestGenerateStatesAndCache(t *testing.T) {
	jc := &fakeJira{
		issues: []jira.Issue{
			{Key: "TEST-2", Fields: &jira.IssueFields{Status: &jira.Status{Name: "Closed"}}},
			{Key: "TEST-1", Fields: &jira.IssueFields{Status: &jira.Status{Name: "Open"}, Summary: "PRIVATE CUSTOMER", Description: "SECRET"}},
		},
		links: map[string][]jira.RemoteLink{
			"TEST-1": {
				remote("https://github.com/Org/Repo/issues/001?utm=x#comment"),
				remote("https://github.com/org/repo/issues/1/"),
				remote("https://github.com/org/repo/issues/2"), // issue URL resolving to PR
				remote("https://github.com/org/repo/pull/2#discussion"),
				remote("https://github.com/else/other/pull/3"),
			},
			"TEST-2": {remote("https://github.com/org/repo/issues/2"), remote("https://github.com/org/repo/issues/1")},
		},
	}
	gc := &fakeGitHub{
		issues: map[string]*github.Issue{
			"org/repo/1": {State: github.String("closed")},
			"org/repo/2": {State: github.String("closed"), PullRequestLinks: &github.PullRequestLinks{}},
		},
		pulls: map[string]*github.PullRequest{
			"org/repo/2":   {State: github.String("closed"), Merged: github.Bool(true)},
			"else/other/3": {State: github.String("open"), Merged: github.Bool(false)},
		},
		reads: map[string]int{},
	}
	rows, err := Generate(context.Background(), "TEST", `status = Open OR status = Closed`, jc, gc)
	require.NoError(t, err)
	require.Equal(t, `project = "TEST" AND (status = Open OR status = Closed)`, jc.jql)
	require.Len(t, rows, 5)
	require.Equal(t, "TEST-1", rows[0].Jira.Key)
	require.Equal(t, "else/other", rows[0].GitHub.Repository)
	require.Equal(t, "pull_request", rows[0].GitHub.Type)
	require.Equal(t, github.Bool(false), rows[0].GitHub.Merged)
	require.Equal(t, "Open", rows[1].Jira.Status)
	require.Equal(t, "closed", rows[1].GitHub.State) // normal independent state, never a mismatch
	require.Equal(t, "linked", rows[1].LinkStatus)
	require.Nil(t, rows[1].GitHub.Merged)
	require.Equal(t, "https://github.com/org/repo/issues/1", rows[1].GitHub.URL)
	require.Equal(t, "pull_request", rows[2].GitHub.Type)
	require.Equal(t, github.Bool(true), rows[2].GitHub.Merged)
	require.Equal(t, "https://github.com/org/repo/pull/2", rows[2].GitHub.URL)
	require.Equal(t, "Closed", rows[3].Jira.Status)
	require.Equal(t, map[string]int{"issue:org/repo/1": 1, "issue:org/repo/2": 1, "pull:org/repo/2": 1, "pull:else/other/3": 1}, gc.reads)
	require.Zero(t, jc.mutations)
	var first bytes.Buffer
	require.NoError(t, Write(&first, "json", rows))
	require.NotContains(t, first.String(), "PRIVATE CUSTOMER")
	require.NotContains(t, first.String(), "SECRET")
	require.NotContains(t, first.String(), "mismatch")
	encoded := make(map[string]string)
	for _, format := range []string{"json", "yaml"} {
		var buf bytes.Buffer
		require.NoError(t, Write(&buf, format, rows))
		checkMachineRows(t, format, buf.Bytes(), rows)
		encoded[format] = buf.String()
	}
	// Reverse API order; machine output must still be identical.
	jc.issues[0], jc.issues[1] = jc.issues[1], jc.issues[0]
	for key, links := range jc.links {
		for i, j := 0, len(links)-1; i < j; i, j = i+1, j-1 {
			links[i], links[j] = links[j], links[i]
		}
		jc.links[key] = links
	}
	rows, err = Generate(context.Background(), "TEST", "", jc, gc)
	require.NoError(t, err)
	var second bytes.Buffer
	require.NoError(t, Write(&second, "json", rows))
	require.Equal(t, first.String(), second.String())
	for _, format := range []string{"json", "yaml"} {
		var buf bytes.Buffer
		require.NoError(t, Write(&buf, format, rows))
		require.Equal(t, encoded[format], buf.String())
	}
}

func TestOrdinaryIssueCannotValidatePullReference(t *testing.T) {
	for _, acrossItems := range []bool{false, true} {
		for _, pullFirst := range []bool{false, true} {
			name := "same Jira/issue first"
			if acrossItems {
				name = "cross Jira/issue first"
			}
			if pullFirst {
				name += "/pull first"
			}
			t.Run(name, func(t *testing.T) {
				links := []jira.RemoteLink{remote("https://github.com/a/b/issues/7"), remote("https://github.com/a/b/pull/7")}
				if pullFirst {
					links[0], links[1] = links[1], links[0]
				}
				jc := &fakeJira{issues: []jira.Issue{{Key: "TEST-1"}}, links: map[string][]jira.RemoteLink{"TEST-1": links}}
				if acrossItems {
					jc.issues = append(jc.issues, jira.Issue{Key: "TEST-2"})
					jc.links["TEST-1"], jc.links["TEST-2"] = links[:1], links[1:]
				}
				failure := errors.New("pull request not found")
				gc := &fakeGitHub{issues: map[string]*github.Issue{"a/b/7": {State: github.String("open")}}, reads: map[string]int{}, pullErr: failure}
				rows, err := Generate(context.Background(), "TEST", "", jc, gc)
				require.ErrorIs(t, err, failure)
				require.Nil(t, rows)
				require.Equal(t, 1, gc.reads["pull:a/b/7"])
				if !pullFirst {
					require.Equal(t, 1, gc.reads["issue:a/b/7"])
				}
				require.Zero(t, jc.mutations)
			})
		}
	}
}

func TestPRAliasesStillShareCache(t *testing.T) {
	for _, pullFirst := range []bool{false, true} {
		links := []jira.RemoteLink{remote("https://github.com/a/b/issues/7"), remote("https://github.com/A/B/pull/007#alias")}
		if pullFirst {
			links[0], links[1] = links[1], links[0]
		}
		jc := &fakeJira{issues: []jira.Issue{{Key: "TEST-2"}, {Key: "TEST-1"}}, links: map[string][]jira.RemoteLink{"TEST-1": links, "TEST-2": links}}
		gc := &fakeGitHub{
			issues: map[string]*github.Issue{"a/b/7": {State: github.String("closed"), PullRequestLinks: &github.PullRequestLinks{}}},
			pulls:  map[string]*github.PullRequest{"a/b/7": {State: github.String("closed"), Merged: github.Bool(true)}}, reads: map[string]int{},
		}
		rows, err := Generate(context.Background(), "TEST", "", jc, gc)
		require.NoError(t, err)
		require.Len(t, rows, 2)
		require.Equal(t, "TEST-1", rows[0].Jira.Key)
		require.Equal(t, "pull_request", rows[0].GitHub.Type)
		require.Equal(t, 1, gc.reads["pull:a/b/7"])
		if pullFirst {
			require.Zero(t, gc.reads["issue:a/b/7"])
		} else {
			require.Equal(t, 1, gc.reads["issue:a/b/7"])
		}
	}
}

func checkMachineRows(t *testing.T, format string, data []byte, want []Row) {
	t.Helper()
	if format == "yaml" {
		var err error
		data, err = yaml.YAMLToJSON(data)
		require.NoError(t, err)
	}
	var typed []Row
	require.NoError(t, json.Unmarshal(data, &typed))
	require.Equal(t, want, typed)
	var decoded []map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Len(t, decoded, len(want))
	for i, row := range decoded {
		fieldCount := 2 // jira and link_status are always present.
		jiraFields := row["jira"].(map[string]interface{})
		require.Equal(t, map[string]interface{}{"key": want[i].Jira.Key, "status": want[i].Jira.Status}, jiraFields)
		if want[i].GitHub == nil {
			require.NotContains(t, row, "github")
		} else {
			fieldCount++
			fields := row["github"].(map[string]interface{})
			source := want[i].GitHub
			expected := map[string]interface{}{"repository": source.Repository, "type": source.Type, "number": float64(source.Number), "url": source.URL, "state": source.State}
			if source.Merged != nil {
				expected["merged"] = *source.Merged
			}
			require.Equal(t, expected, fields) // numeric IDs, boolean merge flags, no extra fields
		}
		if want[i].Problem == "" {
			require.NotContains(t, row, "problem")
		} else {
			fieldCount++
			require.Equal(t, want[i].Problem, row["problem"])
		}
		require.Len(t, row, fieldCount)
		require.Equal(t, want[i].LinkStatus, row["link_status"])
	}
}

func TestParseReference(t *testing.T) {
	cases := []struct {
		url       string
		candidate bool
		valid     bool
	}{
		{"https://github.com/Org/Repo/issues/007#fragment", true, true},
		{"https://GITHUB.COM/a/b/pull/1/?x=y", true, true},
		{"https://example.com/a/b/issues/1", false, false},
		{"https://example.com/github.com/a/b/issues/1", false, false},
		{"https://example.com/?url=https://github.com/a/b/issues/1", false, false},
		{"https://example.com/#github.com", false, false},
		{"example.com/?url=https://github.com/a/b/issues/1", false, false},
		{"https://github.com.evil/a/b/issues/1", true, false},
		{"https://evilgithub.com/a/b/issues/1", true, false},
		{"https://github.com@evil.test/a/b/issues/1", true, false},
		{"https://secret@github.com/a/b/issues/1", true, false},
		{"https://github.com:443/a/b/issues/1", true, false},
		{"http://github.com/a/b/issues/1", true, false},
		{"//github.com/a/b/issues/1", true, false},
		{"github.com/a/b/issues/1", true, false},
		{"https://github.com/a/b/issues/0", true, false},
		{"https://github.com/a/b/issues/-1", true, false},
		{"https://github.com/a/b/pull/1/files", true, false},
		{"https://github.com/a/b/issues/1/extra", true, false},
		{"https://github.com/a/b/issues/999999999999999999999999999999", true, false},
		{"https://github.com/a/b/issues/%31", true, false},
		{"https://github.com/a/%2e%2e/issues/1", true, false},
		{"https://github.com/a/../issues/1", true, false},
		{"https://github.com/a/b/issue/1", true, false},
		{"https://github.com/a/b/issues/1//", true, false},
		{"https://github.com/%zz", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.url, func(t *testing.T) {
			ref, candidate, problem := parseReference(tc.url)
			require.Equal(t, tc.candidate, candidate)
			require.Equal(t, tc.valid, candidate && problem == "")
			if tc.valid {
				require.Positive(t, ref.number)
			} else if candidate {
				require.NotEmpty(t, problem)
			}
		})
	}
}

func TestGenerateMissingMalformedAndNilFields(t *testing.T) {
	jc := &fakeJira{
		issues: []jira.Issue{{Key: "TEST-1"}, {Key: "TEST-2", Fields: &jira.IssueFields{}}, {Key: "TEST-3"}},
		links: map[string][]jira.RemoteLink{
			"TEST-1": {{}, remote("https://example.org/unrelated")},
			"TEST-2": {remote("https://secret@github.com/a/b/issues/1"), remote("https://secret@github.com/a/b/issues/1")},
		},
	}
	gc := &fakeGitHub{reads: map[string]int{}}
	rows, err := Generate(context.Background(), "TEST", "", jc, gc)
	require.NoError(t, err)
	require.Len(t, rows, 4)
	require.Equal(t, "no_supported_github_links", rows[0].LinkStatus)
	require.Equal(t, "malformed_github_reference", rows[1].LinkStatus)
	require.Equal(t, "no_supported_github_links", rows[2].LinkStatus)
	require.Empty(t, gc.reads)
	require.Zero(t, jc.mutations)
	var buf bytes.Buffer
	require.NoError(t, Write(&buf, "json", rows))
	require.NotContains(t, buf.String(), "secret")
	checkMachineRows(t, "json", buf.Bytes(), rows)
	buf.Reset()
	require.NoError(t, Write(&buf, "yaml", rows))
	checkMachineRows(t, "yaml", buf.Bytes(), rows)
}

func TestGenerateReadFailures(t *testing.T) {
	failure := errors.New("read failed")
	for _, stage := range []string{"search", "links", "issue", "pull", "resolved pull", "nil issue", "nil pull", "missing merged", "missing state"} {
		t.Run(stage, func(t *testing.T) {
			jc := &fakeJira{issues: []jira.Issue{{Key: "TEST-1"}}, links: map[string][]jira.RemoteLink{"TEST-1": {remote("https://github.com/a/b/issues/1")}}}
			gc := &fakeGitHub{issues: map[string]*github.Issue{"a/b/1": {State: github.String("open")}}, pulls: map[string]*github.PullRequest{}, reads: map[string]int{}}
			switch stage {
			case "search":
				jc.searchErr = failure
			case "links":
				jc.linksErr = failure
			case "issue":
				gc.err = failure
			case "pull", "nil pull", "missing merged":
				jc.links["TEST-1"] = []jira.RemoteLink{remote("https://github.com/a/b/pull/1")}
				if stage == "pull" {
					gc.err = failure
				} else if stage == "missing merged" {
					gc.pulls["a/b/1"] = &github.PullRequest{State: github.String("closed")}
				}
			case "resolved pull":
				gc.issues["a/b/1"].PullRequestLinks = &github.PullRequestLinks{}
			case "nil issue":
				gc.issues["a/b/1"] = nil
			case "missing state":
				gc.issues["a/b/1"] = &github.Issue{}
			}
			rows, err := Generate(context.Background(), "TEST", "", jc, gc)
			require.Error(t, err)
			require.Nil(t, rows)
			if stage == "search" || stage == "links" || stage == "issue" || stage == "pull" {
				require.ErrorIs(t, err, failure)
			}
			require.Zero(t, jc.mutations)
		})
	}
}

func TestGenerateReopenedUpstreamAndClosedUnmergedPR(t *testing.T) {
	jc := &fakeJira{
		issues: []jira.Issue{{Key: "TEST-1", Fields: &jira.IssueFields{Status: &jira.Status{Name: "Closed"}}}},
		links: map[string][]jira.RemoteLink{"TEST-1": {
			remote("https://github.com/a/b/issues/1"),
			remote("https://github.com/a/b/pull/2"),
		}},
	}
	gc := &fakeGitHub{
		issues: map[string]*github.Issue{"a/b/1": {State: github.String("open")}},
		pulls:  map[string]*github.PullRequest{"a/b/2": {State: github.String("closed"), Merged: github.Bool(false)}},
		reads:  map[string]int{},
	}
	rows, err := Generate(context.Background(), "TEST", "", jc, gc)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, "Closed", rows[0].Jira.Status)
	require.Equal(t, "open", rows[0].GitHub.State)
	require.Equal(t, "closed", rows[1].GitHub.State)
	require.Equal(t, github.Bool(false), rows[1].GitHub.Merged)
	require.Zero(t, jc.mutations)
}

func TestScope(t *testing.T) {
	for _, tc := range []struct {
		project, query, want string
		valid                bool
	}{
		{"TEST", "", `project = "TEST"`, true},
		{"TEST", "   ", `project = "TEST"`, true},
		{"TEST_1", `status = Open OR status = Closed`, `project = "TEST_1" AND (status = Open OR status = Closed)`, true},
		{"TEST", `summary ~ "(order by)"`, `project = "TEST" AND (summary ~ "(order by)")`, true},
		{"TEST", `status in (Open, Closed)`, `project = "TEST" AND (status in (Open, Closed))`, true},
		{"", "", "", false},
		{"TEST OR project = OTHER", "", "", false},
		{" TEST ", "", "", false},
		{"TEST", `status = Open) OR project = OTHER`, "", false},
		{"TEST", `(status = Open`, "", false},
		{"TEST", `status = "Open`, "", false},
		{"TEST", `status = Open ORDER BY key`, "", false},
		{"TEST", `status = Open; project = OTHER`, "", false},
		{"TEST", `status = Open -- comment`, "", false},
		{"TEST", `status = "Open\`, "", false},
	} {
		t.Run(tc.project+"/"+tc.query, func(t *testing.T) {
			got, err := Scope(tc.project, tc.query)
			if tc.valid {
				require.NoError(t, err)
				require.Equal(t, tc.want, got)
			} else {
				require.Error(t, err)
			}
		})
	}
}

type errorWriter struct{ err error }

func (w errorWriter) Write([]byte) (int, error) { return 0, w.err }

func TestOutput(t *testing.T) {
	rows := []Row{{Jira: JiraItem{Key: "TEST-1", Status: "Open\nInjected"}, GitHub: &Source{Repository: "a/b", Type: "pull_request", Number: 1, State: "closed", Merged: github.Bool(false)}, LinkStatus: "linked"}}
	for _, format := range []string{"json", "yaml", "table"} {
		t.Run(format, func(t *testing.T) {
			var buf bytes.Buffer
			require.NoError(t, Write(&buf, format, rows))
			require.Contains(t, buf.String(), "TEST-1")
			require.Contains(t, buf.String(), "false")
			require.NotContains(t, buf.String(), "\nInjected")
			failure := errors.New("broken pipe")
			require.ErrorIs(t, Write(errorWriter{failure}, format, rows), failure)
			require.ErrorIs(t, Write(errorWriter{}, format, rows), io.ErrShortWrite)
		})
	}
	var empty bytes.Buffer
	require.NoError(t, Write(&empty, "json", nil))
	require.Equal(t, "[]\n", empty.String())
	require.Error(t, Write(io.Discard, "xml", rows))
}

func TestGenerateCanceledAndEmpty(t *testing.T) {
	jc := &fakeJira{issues: []jira.Issue{{Key: "TEST-1"}}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rows, err := Generate(ctx, "TEST", "", jc, &fakeGitHub{})
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, rows)
	jc.issues = nil
	rows, err = Generate(context.Background(), "TEST", "", jc, &fakeGitHub{})
	require.NoError(t, err)
	require.NotNil(t, rows)
	require.Empty(t, rows)
	require.False(t, strings.Contains(jc.jql, "status"))
}
