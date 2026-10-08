package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func syntheticAdapter(t *testing.T, mode, baseURL string) JiraClient {
	t.Helper()
	if mode == "cloud" {
		client, err := newCloudClient(baseURL, "synthetic@example.invalid", "synthetic-token")
		if err != nil {
			t.Fatal(err)
		}
		return client
	}
	client, err := newOnpremClient(baseURL, "synthetic-token")
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestAdaptersLabelsCreateAndSearch(t *testing.T) {
	for _, mode := range []string{"cloud", "dc"} {
		t.Run(mode, func(t *testing.T) {
			marker := sourceMarker("https://github.com/org/repo/issues/42")
			creates, searches := 0, 0
			jql := `project = "DST" AND issuetype = "Bug"`
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/issue") {
					creates++
					var body struct {
						Fields struct {
							Labels []string `json:"labels"`
						} `json:"fields"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					if len(body.Fields.Labels) != 1 || body.Fields.Labels[0] != marker {
						t.Errorf("creation lost marker: %+v", body)
					}
					w.WriteHeader(http.StatusCreated)
					fmt.Fprint(w, `{"id":"10","key":"DST-10"}`)
					return
				}
				if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/search") {
					searches++
					if r.URL.Query().Get("jql") != jql {
						t.Errorf("JQL changed: %q", r.URL.Query().Get("jql"))
					}
					issue := fmt.Sprintf(`{"id":"%d","key":"DST-%d","fields":{"summary":"synthetic","project":{"key":"DST"},"issuetype":{"name":"Bug"},"status":{"name":"Closed"},"labels":[%q]}}`, searches, searches, marker)
					if mode == "cloud" {
						if !strings.Contains(r.URL.Query().Get("fields"), "labels") {
							t.Error("labels not requested")
						}
						if searches == 1 {
							fmt.Fprintf(w, `{"issues":[%s],"isLast":false,"nextPageToken":"page2"}`, issue)
						} else {
							if r.URL.Query().Get("nextPageToken") != "page2" {
								t.Error("pagination token not sent")
							}
							fmt.Fprintf(w, `{"issues":[%s],"isLast":true}`, issue)
						}
					} else {
						offset := r.URL.Query().Get("startAt")
						if offset != fmt.Sprint(searches-1) && !(searches == 1 && offset == "") {
							t.Error("pagination offset not sent")
						}
						fmt.Fprintf(w, `{"issues":[%s],"startAt":%d,"total":2,"maxResults":1}`, issue, searches-1)
					}
					return
				}
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()
			client := syntheticAdapter(t, mode, server.URL)
			created, err := client.CreateIssue(context.Background(), &Issue{Fields: &IssueFields{
				Summary: "synthetic", Project: Project{Key: "DST"}, Type: IssueType{Name: "Bug"}, Labels: []string{marker},
			}})
			if err != nil || created == nil || created.Key != "DST-10" {
				t.Fatalf("create = %v, %v", created, err)
			}
			issues, err := client.SearchIssues(context.Background(), jql)
			if err != nil || len(issues) != 2 || creates != 1 || searches != 2 {
				t.Fatalf("search = %v, %v; calls=%d/%d", issues, err, creates, searches)
			}
			for _, issue := range issues {
				if issue.Fields == nil || len(issue.Fields.Labels) != 1 || issue.Fields.Labels[0] != marker || issue.Fields.Status.Name != "Closed" {
					t.Fatalf("search conversion lost fields: %+v", issue.Fields)
				}
			}
		})
	}
}

func TestAdaptersIncompleteSearchFailsClosed(t *testing.T) {
	for _, mode := range []string{"cloud", "dc"} {
		t.Run(mode, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if mode == "cloud" {
					fmt.Fprint(w, `{"issues":[],"isLast":false}`)
				} else {
					fmt.Fprint(w, `{"issues":[],"startAt":0,"total":1}`)
				}
			}))
			defer server.Close()
			issues, err := syntheticAdapter(t, mode, server.URL).SearchIssues(context.Background(), `project = "DST"`)
			if err == nil || issues != nil || !strings.Contains(err.Error(), "incomplete") {
				t.Fatalf("accepted incomplete search: %v, %v", issues, err)
			}
		})
	}
}

func TestAdaptersSearchCompleteness(t *testing.T) {
	issue := `{"id":"1","key":"DST-1","fields":{"labels":[]}}`
	dcFirst := fmt.Sprintf(`{"issues":[%s],"startAt":0,"total":2}`, issue)
	cloudFirst := fmt.Sprintf(`{"issues":[%s],"isLast":false,"nextPageToken":"page2"}`, issue)
	for _, tc := range []struct {
		name, mode  string
		pages       []string
		secondError bool
	}{
		{name: "skipped first offset", mode: "dc", pages: []string{fmt.Sprintf(`{"issues":[%s],"startAt":100,"total":101}`, issue)}},
		{name: "repeated offset", mode: "dc", pages: []string{dcFirst, dcFirst}},
		{name: "skipped second offset", mode: "dc", pages: []string{dcFirst, fmt.Sprintf(`{"issues":[%s],"startAt":2,"total":3}`, issue)}},
		{name: "missing metadata", mode: "dc", pages: []string{`{}`}},
		{name: "missing total", mode: "dc", pages: []string{`{"issues":[],"startAt":0}`}},
		{name: "missing offset", mode: "dc", pages: []string{`{"issues":[],"total":0}`}},
		{name: "null metadata", mode: "dc", pages: []string{`{"issues":[],"startAt":null,"total":null}`}},
		{name: "null issues", mode: "dc", pages: []string{`{"issues":null,"startAt":0,"total":0}`}},
		{name: "missing fields", mode: "cloud", pages: []string{`{"issues":[{"id":"1","key":"DST-1"}],"isLast":true}`}},
		{name: "null fields", mode: "cloud", pages: []string{`{"issues":[{"id":"1","key":"DST-1","fields":null}],"isLast":true}`}},
		{name: "null issues", mode: "cloud", pages: []string{`{"issues":null,"isLast":true}`}},
		{name: "repeated pagination token", mode: "cloud", pages: []string{cloudFirst, cloudFirst}},
		{name: "second page failure", mode: "cloud", pages: []string{cloudFirst, `{"errorMessages":["synthetic page failure"]}`}, secondError: true},
		{name: "second page failure", mode: "dc", pages: []string{dcFirst, `{"errorMessages":["synthetic page failure"]}`}, secondError: true},
	} {
		t.Run(tc.mode+"/"+tc.name, func(t *testing.T) {
			var requests, writes atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method != http.MethodGet || !strings.Contains(r.URL.Path, "/search") {
					writes.Add(1)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				n := int(requests.Add(1)) - 1
				if n >= len(tc.pages) {
					t.Error("search did not terminate")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				if tc.secondError && n == 1 {
					w.WriteHeader(http.StatusInternalServerError)
				}
				fmt.Fprint(w, tc.pages[n])
			}))
			defer server.Close()
			client := syntheticAdapter(t, tc.mode, server.URL)
			issues, err := client.SearchIssues(context.Background(), `project = "DST"`)
			if err == nil || issues != nil || int(requests.Load()) != len(tc.pages) {
				t.Fatalf("search did not fail closed: issues=%v err=%v requests=%d", issues, err, requests.Load())
			}
			requests.Store(0)
			_, err = (&Connection{jiraClient: client}).Clone(context.Background(), sourceIssue(), "DST", "Bug", false, WithIfMissing(), WithCloneWriter(io.Discard))
			if err == nil || writes.Load() != 0 {
				t.Fatalf("incomplete search permitted mutation: err=%v writes=%d", err, writes.Load())
			}
		})
	}
}

func TestAdaptersRemoteLinkPayloads(t *testing.T) {
	for _, mode := range []string{"cloud", "dc"} {
		for _, tc := range []struct {
			name, payload string
			wantErr       bool
			wantCount     int
		}{
			{name: "null", payload: `null`, wantErr: true},
			{name: "empty array", payload: `[]`},
			{name: "relationships", payload: `[{"relationship":"depends on","object":{"url":"https://github.com/org/repo/issues/42","title":"Org/Repo#42"}},{"object":{"url":"https://github.com/org/repo/issues/42","title":"legacy"}},{"relationship":"fixes","object":{"url":"https://github.com/org/repo/issues/42","title":"fix"}}]`, wantCount: 3},
		} {
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				var writes atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					if r.Method != http.MethodGet {
						writes.Add(1)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					if strings.HasSuffix(r.URL.Path, "/remotelink") {
						fmt.Fprint(w, tc.payload)
						return
					}
					issue := `{"id":"9","key":"DST-9","fields":{"labels":[]}}`
					if mode == "cloud" {
						fmt.Fprintf(w, `{"issues":[%s],"isLast":true}`, issue)
					} else {
						fmt.Fprintf(w, `{"issues":[%s],"startAt":0,"total":1}`, issue)
					}
				}))
				defer server.Close()
				client := syntheticAdapter(t, mode, server.URL)
				links, err := client.GetRemoteLinks(context.Background(), "DST-9")
				if tc.wantErr {
					if err == nil || links != nil {
						t.Fatalf("null links accepted: %v, %v", links, err)
					}
					_, err = (&Connection{jiraClient: client}).Clone(context.Background(), sourceIssue(), "DST", "Bug", false, WithIfMissing(), WithCloneWriter(io.Discard))
					if err == nil || writes.Load() != 0 {
						t.Fatalf("null links permitted mutation: err=%v writes=%d", err, writes.Load())
					}
				} else {
					if err != nil || links == nil || len(links) != tc.wantCount {
						t.Fatalf("links=%v err=%v", links, err)
					}
					if tc.wantCount == 3 && (links[0].Relationship != "depends on" || links[1].Relationship != "" || links[2].Relationship != "fixes") {
						t.Fatalf("relationship conversion lost metadata: %+v", links)
					}
				}
			})
		}
	}
}

func TestAdaptersCreateResponseValidation(t *testing.T) {
	for _, mode := range []string{"cloud", "dc"} {
		for _, tc := range []struct {
			name, payload string
			wantErr       bool
		}{
			{name: "null", payload: `null`, wantErr: true},
			{name: "empty object", payload: `{}`, wantErr: true},
			{name: "null identifiers", payload: `{"id":null,"key":null}`, wantErr: true},
			{name: "key only", payload: `{"id":null,"key":"DST-9"}`},
		} {
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusCreated)
					fmt.Fprint(w, tc.payload)
				}))
				defer server.Close()
				created, err := syntheticAdapter(t, mode, server.URL).CreateIssue(context.Background(), &Issue{Fields: &IssueFields{Summary: "synthetic"}})
				if tc.wantErr {
					if err == nil || created != nil {
						t.Fatalf("malformed create accepted: %v, %v", created, err)
					}
				} else if err != nil || created == nil || created.Key != "DST-9" {
					t.Fatalf("key-only response rejected: %v, %v", created, err)
				}
			})
		}
	}
}
