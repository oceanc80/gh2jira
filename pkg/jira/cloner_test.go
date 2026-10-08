package jira

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/google/go-github/v60/github"
)

type fakeJiraClient struct {
	issues                         []Issue
	links                          map[string][]RemoteLink
	searchErr, readErr, createErr  error
	linkErr                        error
	creates, adds, searches, reads int
	jql                            string
	badCreate                      bool
	readErrKey                     string
}

func (f *fakeJiraClient) SearchIssues(_ context.Context, jql string) ([]Issue, error) {
	f.searches++
	f.jql = jql
	return f.issues, f.searchErr
}

func (f *fakeJiraClient) CreateIssue(_ context.Context, issue *Issue) (*Issue, error) {
	f.creates++
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.badCreate {
		return nil, nil
	}
	created := *issue
	created.ID = fmt.Sprint(f.creates)
	created.Key = fmt.Sprintf("DST-%d", f.creates)
	f.issues = append(f.issues, created)
	return &created, nil
}

func (f *fakeJiraClient) AddRemoteLink(_ context.Context, id string, link *RemoteLink) error {
	f.adds++
	if f.linkErr != nil {
		return f.linkErr
	}
	if f.links == nil {
		f.links = make(map[string][]RemoteLink)
	}
	for _, issue := range f.issues {
		if issue.ID == id || issue.Key == id {
			f.links[issue.Key] = append(f.links[issue.Key], *link)
			return nil
		}
	}
	return errors.New("unknown Jira issue")
}

func (f *fakeJiraClient) GetRemoteLinks(_ context.Context, key string) ([]RemoteLink, error) {
	f.reads++
	if f.readErrKey == "" || f.readErrKey == key {
		return f.links[key], f.readErr
	}
	return f.links[key], nil
}

func sourceIssue() *github.Issue {
	return &github.Issue{
		Number: github.Int(42), HTMLURL: github.String("https://github.com/org/repo/issues/42"),
		Title: github.String("PRIVATE TITLE"), Body: github.String("PRIVATE BODY"), NodeID: github.String("synthetic-node"),
	}
}

func existingIssue(key string, marked bool) Issue {
	issue := Issue{ID: key, Key: key, Fields: &IssueFields{
		Project: Project{Key: "DST"}, Type: IssueType{Name: "Bug"}, Summary: "unrelated title",
	}}
	if marked {
		issue.Fields.Labels = []string{sourceMarker(sourceIssue().GetHTMLURL())}
	}
	return issue
}

func sourceLink(raw string) RemoteLink {
	return RemoteLink{Object: &RemoteLinkObject{URL: raw, Title: "not identity"}}
}

func TestCloneIfMissing(t *testing.T) {
	boom := errors.New("synthetic failure")
	for _, tc := range []struct {
		name       string
		setup      func(*fakeJiraClient)
		runs       int
		wantCreate int
		wantAdd    int
		wantErr    string
	}{
		{name: "first create", runs: 1, wantCreate: 1, wantAdd: 1},
		{name: "repeat reuse", runs: 2, wantCreate: 1, wantAdd: 1},
		{name: "legacy exact canonical link", setup: func(f *fakeJiraClient) {
			f.issues = []Issue{existingIssue("DST-9", false)}
			f.links = map[string][]RemoteLink{"DST-9": {sourceLink("https://GITHUB.com/ORG/REPO/issues/42/")}}
		}, runs: 1},
		{name: "different repo same number", setup: func(f *fakeJiraClient) {
			f.issues = []Issue{existingIssue("DST-9", false)}
			f.links = map[string][]RemoteLink{"DST-9": {sourceLink("https://github.com/org/other/issues/42")}}
		}, runs: 1, wantCreate: 1, wantAdd: 1},
		{name: "closed existing record", setup: func(f *fakeJiraClient) {
			i := existingIssue("DST-9", true)
			i.Fields.Status = &Status{Name: "Closed"}
			f.issues = []Issue{i}
			f.links = map[string][]RemoteLink{"DST-9": {sourceLink(sourceIssue().GetHTMLURL())}}
		}, runs: 1},
		{name: "ambiguous legacy relationships", setup: func(f *fakeJiraClient) {
			f.issues = []Issue{existingIssue("DST-9", false), existingIssue("DST-8", false)}
			f.links = map[string][]RemoteLink{
				"DST-9": {sourceLink(sourceIssue().GetHTMLURL())}, "DST-8": {sourceLink(sourceIssue().GetHTMLURL())},
			}
		}, runs: 1, wantErr: "existing-record link command"},
		{name: "marker plus legacy ambiguity", setup: func(f *fakeJiraClient) {
			f.issues = []Issue{existingIssue("DST-9", true), existingIssue("DST-8", false)}
			f.links = map[string][]RemoteLink{"DST-8": {sourceLink(sourceIssue().GetHTMLURL())}}
		}, runs: 1, wantErr: "ambiguous"},
		{name: "recovery preserves other source links", setup: func(f *fakeJiraClient) {
			f.issues = []Issue{existingIssue("DST-9", true)}
			f.links = map[string][]RemoteLink{"DST-9": {sourceLink("https://github.com/org/other/issues/42")}}
		}, runs: 2, wantAdd: 1},
		{name: "search failure", setup: func(f *fakeJiraClient) { f.searchErr = boom }, runs: 1, wantErr: "searching clone destination"},
		{name: "link read failure", setup: func(f *fakeJiraClient) {
			f.issues = []Issue{existingIssue("DST-9", true)}
			f.readErr = boom
		}, runs: 1, wantErr: "reading remote links"},
		{name: "late link read failure", setup: func(f *fakeJiraClient) {
			f.issues = []Issue{existingIssue("DST-9", true), existingIssue("DST-8", false)}
			f.readErr = boom
			f.readErrKey = "DST-8"
		}, runs: 1, wantErr: "reading remote links"},
		{name: "incomplete search record", setup: func(f *fakeJiraClient) { f.issues = []Issue{{Key: "DST-9"}} }, runs: 1, wantErr: "incomplete"},
		{name: "create failure", setup: func(f *fakeJiraClient) { f.createErr = boom }, runs: 1, wantCreate: 1, wantErr: "synthetic failure"},
		{name: "malformed create response", setup: func(f *fakeJiraClient) { f.badCreate = true }, runs: 1, wantCreate: 1, wantErr: "no issue ID or key"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeJiraClient{}
			if tc.setup != nil {
				tc.setup(f)
			}
			conn := &Connection{jiraClient: f}
			var output bytes.Buffer
			for n := 0; n < tc.runs; n++ {
				issue, err := conn.Clone(context.Background(), sourceIssue(), "DST", "Bug", false, WithIfMissing(), WithCloneWriter(&output))
				if tc.wantErr != "" {
					if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
						t.Fatalf("error = %v, want %q", err, tc.wantErr)
					}
				} else if err != nil || issue == nil {
					t.Fatalf("issue = %v, error = %v", issue, err)
				}
			}
			if f.creates != tc.wantCreate || f.adds != tc.wantAdd {
				t.Fatalf("creates/adds = %d/%d, want %d/%d", f.creates, f.adds, tc.wantCreate, tc.wantAdd)
			}
			if f.jql != `project = "DST" AND issuetype = "Bug"` {
				t.Fatalf("unscoped or status-filtered JQL: %q", f.jql)
			}
			if strings.Contains(output.String(), "PRIVATE") {
				t.Fatalf("private source fields in output: %s", output.String())
			}
			if tc.name == "recovery preserves other source links" && len(f.links["DST-9"]) != 2 {
				t.Fatal("existing many-to-many relationship was changed")
			}
		})
	}
}

func TestCloneLinkFailureRecovery(t *testing.T) {
	f := &fakeJiraClient{linkErr: errors.New("link failed")}
	conn := &Connection{jiraClient: f}
	options := []CloneOption{WithIfMissing(), WithCloneWriter(io.Discard)}
	created, err := conn.Clone(context.Background(), sourceIssue(), "DST", "Bug", false, options...)
	if err == nil || created == nil || f.creates != 1 || len(f.issues[0].Fields.Labels) != 1 {
		t.Fatalf("partial success must retain durable marker: created=%v err=%v", created, err)
	}
	f.linkErr = nil
	// A fresh connection models a later process; recovery uses Jira state only.
	conn = &Connection{jiraClient: f}
	recovered, err := conn.Clone(context.Background(), sourceIssue(), "DST", "Bug", false, options...)
	if err != nil || recovered.Key != created.Key || f.creates != 1 || f.adds != 2 {
		t.Fatalf("recovery created a duplicate: issue=%v err=%v creates=%d adds=%d", recovered, err, f.creates, f.adds)
	}
	_, err = conn.Clone(context.Background(), sourceIssue(), "DST", "Bug", false, options...)
	if err != nil || f.creates != 1 || f.adds != 2 {
		t.Fatal("recovered relationship is not repeatable")
	}
}

func TestCloneDryRun(t *testing.T) {
	for _, tc := range []struct {
		name, want     string
		marked, linked bool
	}{
		{name: "create", want: "would create"},
		{name: "reuse", want: "would reuse Jira issue", marked: true, linked: true},
		{name: "recover", want: "add missing source link", marked: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeJiraClient{}
			if tc.marked {
				f.issues = []Issue{existingIssue("DST-9", true)}
			}
			if tc.linked {
				f.links = map[string][]RemoteLink{"DST-9": {sourceLink(sourceIssue().GetHTMLURL())}}
			}
			var output bytes.Buffer
			_, err := (&Connection{jiraClient: f}).Clone(context.Background(), sourceIssue(), "DST", "Bug", true, WithIfMissing(), WithCloneWriter(&output))
			if err != nil || f.creates != 0 || f.adds != 0 || !strings.Contains(output.String(), tc.want) {
				t.Fatalf("dry run: err=%v creates=%d adds=%d output=%q", err, f.creates, f.adds, output.String())
			}
			if strings.Contains(output.String(), "PRIVATE") {
				t.Fatal("private source fields in output")
			}
		})
	}
}

func TestCloneSourceValidation(t *testing.T) {
	for _, mode := range []bool{false, true} {
		for _, tc := range []struct {
			name  string
			issue *github.Issue
		}{
			{name: "nil"},
			{name: "empty", issue: &github.Issue{}},
			{name: "negative", issue: &github.Issue{Number: github.Int(-1)}},
			{name: "short URL", issue: &github.Issue{Number: github.Int(42), HTMLURL: github.String("x")}},
			{name: "bad URL", issue: &github.Issue{Number: github.Int(42), HTMLURL: github.String("https://%")}},
			{name: "mismatch", issue: &github.Issue{Number: github.Int(43), HTMLURL: sourceIssue().HTMLURL}},
			{name: "credentials", issue: &github.Issue{Number: github.Int(42), HTMLURL: github.String("https://secret@github.com/org/repo/issues/42")}},
		} {
			t.Run(fmt.Sprintf("%t/%s", mode, tc.name), func(t *testing.T) {
				f := &fakeJiraClient{}
				var options []CloneOption
				if mode {
					options = append(options, WithIfMissing())
				}
				_, err := (&Connection{jiraClient: f}).Clone(context.Background(), tc.issue, "DST", "Bug", false, options...)
				if err == nil || f.creates != 0 || f.searches != 0 || f.adds != 0 {
					t.Fatalf("invalid source touched Jira: err=%v fake=%+v", err, f)
				}
				// Validation also precedes lazy connection setup.
				_, err = (&Connection{}).Clone(context.Background(), tc.issue, "DST", "Bug", false, options...)
				if err == nil {
					t.Fatal("invalid source accepted")
				}
			})
		}
	}
}

func TestLegacyCloneAlwaysCreates(t *testing.T) {
	f := &fakeJiraClient{}
	conn := &Connection{jiraClient: f}
	for n := 0; n < 2; n++ {
		if _, err := conn.Clone(context.Background(), sourceIssue(), "DST", "Bug", false, WithCloneWriter(io.Discard)); err != nil {
			t.Fatal(err)
		}
	}
	if f.creates != 2 || f.adds != 2 || f.searches != 0 || len(f.issues[0].Fields.Labels) != 0 {
		t.Fatalf("legacy semantics changed: %+v", f)
	}
}

func TestCloneJQLQuoting(t *testing.T) {
	f := &fakeJiraClient{}
	_, err := (&Connection{jiraClient: f}).Clone(context.Background(), sourceIssue(), `DST" OR project = "OTHER`, `Bug\"`, true, WithIfMissing(), WithCloneWriter(io.Discard))
	if err != nil {
		t.Fatal(err)
	}
	if f.jql != `project = "DST\" OR project = \"OTHER" AND issuetype = "Bug\\\""` {
		t.Fatalf("JQL not quoted: %s", f.jql)
	}
}

func TestCloneRelationshipEligibility(t *testing.T) {
	for _, tc := range []struct {
		name          string
		relationships []string
		secondMatch   bool
		wantCreate    int
		wantErr       bool
	}{
		{name: "dependency only", relationships: []string{"depends on"}, wantCreate: 1},
		{name: "unknown explicit role", relationships: []string{"relates to"}, wantCreate: 1},
		{name: "implements", relationships: []string{"implements"}},
		{name: "fixes case insensitive", relationships: []string{" FiXeS "}},
		{name: "legacy", relationships: []string{""}},
		{name: "dependency before legacy", relationships: []string{"depends on", ""}},
		{name: "dependency after legacy", relationships: []string{"", "depends on"}},
		{name: "different roles cannot hide ambiguity", relationships: []string{"", "depends on"}, secondMatch: true, wantErr: true},
		{name: "intentional version matches", relationships: []string{"implements"}, secondMatch: true, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeJiraClient{issues: []Issue{existingIssue("DST-9", false)}, links: make(map[string][]RemoteLink)}
			for _, relationship := range tc.relationships {
				link := sourceLink(sourceIssue().GetHTMLURL())
				link.Relationship = relationship
				f.links["DST-9"] = append(f.links["DST-9"], link)
			}
			if tc.secondMatch {
				f.issues = append(f.issues, existingIssue("DST-8", false))
				link := sourceLink(sourceIssue().GetHTMLURL())
				link.Relationship = "fixes"
				f.links["DST-8"] = []RemoteLink{link}
			}
			result, err := (&Connection{jiraClient: f}).Clone(context.Background(), sourceIssue(), "DST", "Bug", false, WithIfMissing(), WithCloneWriter(io.Discard))
			if tc.wantErr {
				if err == nil || !strings.Contains(err.Error(), "ambiguous") {
					t.Fatalf("ambiguity hidden: %v", err)
				}
			} else if err != nil || result == nil || (tc.wantCreate == 0 && result.Key != "DST-9") {
				t.Fatalf("unexpected reuse/create: result=%v err=%v", result, err)
			}
			if f.creates != tc.wantCreate || f.adds != tc.wantCreate || len(f.links["DST-9"]) != len(tc.relationships) {
				t.Fatalf("unexpected mutations: %+v", f)
			}
			for n, relationship := range tc.relationships {
				if f.links["DST-9"][n].Relationship != relationship {
					t.Fatal("existing relationship changed")
				}
			}
		})
	}
}

func TestClonePreservesSourcePresentation(t *testing.T) {
	for _, ifMissing := range []bool{false, true} {
		t.Run(fmt.Sprint(ifMissing), func(t *testing.T) {
			issue := sourceIssue()
			issue.HTMLURL = github.String("https://GitHub.com/Org42/Repo42/issues/42")
			issue.Title = github.String("Original CASE #42")
			issue.Body = github.String("Intro CASE\r\n- [ ] #7 needs #9 later\r\n- [x] #8 done\r\n- [ ] #42\r\nEnd")
			f := &fakeJiraClient{}
			options := []CloneOption{WithCloneWriter(io.Discard)}
			if ifMissing {
				options = append(options, WithIfMissing())
			}
			created, err := (&Connection{jiraClient: f}).Clone(context.Background(), issue, "DST", "Bug", false, options...)
			if err != nil {
				t.Fatal(err)
			}
			wantBody := "Intro CASE\n * [Org42/Repo42#7|https://GitHub.com/Org42/Repo42/issues/7] needs #9 later\n- [x] #8 done\n * [Org42/Repo42#42|https://GitHub.com/Org42/Repo42/issues/42] \nEnd"
			if created.Fields.Summary != "[UPSTREAM] Original CASE #42 #42" || created.Fields.Description != wantBody {
				t.Fatalf("source presentation changed: %+v", created.Fields)
			}
			link := f.links[created.Key][0]
			if link.Object.URL != issue.GetHTMLURL() || link.Object.Title != "Org42/Repo42#42" {
				t.Fatalf("link presentation normalized: %+v", link.Object)
			}
			if ifMissing {
				wantMarker := sourceMarker("https://github.com/org42/repo42/issues/42")
				if len(created.Fields.Labels) != 1 || created.Fields.Labels[0] != wantMarker {
					t.Fatal("identity must still be normalized")
				}
				issue.HTMLURL = github.String("https://github.com/org42/repo42/issues/42")
				if _, err := (&Connection{jiraClient: f}).Clone(context.Background(), issue, "DST", "Bug", false, options...); err != nil || f.creates != 1 || f.adds != 1 {
					t.Fatalf("case normalization no longer reuses: %v", err)
				}
			}
		})
	}
}

func TestCloneLinkCreateIdentifiers(t *testing.T) {
	for _, tc := range []struct {
		name    string
		issue   *Issue
		wantErr bool
	}{
		{name: "nil issue", wantErr: true},
		{name: "missing identifiers", issue: &Issue{}, wantErr: true},
		{name: "key only", issue: &Issue{Key: "DST-9"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeJiraClient{}
			if tc.issue != nil {
				f.issues = []Issue{*tc.issue}
			}
			err := (&Connection{jiraClient: f}).linkClone(context.Background(), tc.issue, sourceIssue().GetHTMLURL(), 42)
			if tc.wantErr {
				if err == nil || f.adds != 0 {
					t.Fatalf("unsafe identifier used: %v", err)
				}
			} else if err != nil || f.adds != 1 || len(f.links["DST-9"]) != 1 {
				t.Fatalf("key fallback failed: %v", err)
			}
		})
	}
}
