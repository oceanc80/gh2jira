package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	cloud "github.com/andygrunwald/go-jira/v2/cloud"
	onpremise "github.com/andygrunwald/go-jira/v2/onpremise"
)

func remoteLinkAdapters() map[string]func(*httptest.Server) (JiraClient, error) {
	return map[string]func(*httptest.Server) (JiraClient, error){
		"cloud": func(server *httptest.Server) (JiraClient, error) {
			client, err := cloud.NewClient(server.URL, server.Client())
			return &cloudClient{client: client}, err
		},
		"onprem": func(server *httptest.Server) (JiraClient, error) {
			client, err := onpremise.NewClient(server.URL, server.Client())
			return &onpremClient{client: client}, err
		},
	}
}

func TestRemoteLinkAdapterConversion(t *testing.T) {
	for name, newClient := range remoteLinkAdapters() {
		t.Run(name, func(t *testing.T) {
			var posted map[string]interface{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/rest/api/2/issue/EXAMPLE-123/remotelink" {
					t.Errorf("unexpected endpoint %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				switch r.Method {
				case http.MethodGet:
					json.NewEncoder(w).Encode([]map[string]interface{}{
						{"globalId": "synthetic-global-id", "relationship": "implements", "object": map[string]interface{}{"url": "https://github.com/owner/repo/pull/42", "title": "owner/repo pull request #42"}},
						{"globalId": "legacy-id"},
					})
				case http.MethodPost:
					if err := json.NewDecoder(r.Body).Decode(&posted); err != nil {
						t.Error(err)
					}
					w.WriteHeader(http.StatusCreated)
					json.NewEncoder(w).Encode(map[string]interface{}{"id": 1})
				default:
					t.Errorf("unexpected method %s", r.Method)
					w.WriteHeader(http.StatusMethodNotAllowed)
				}
			}))
			defer server.Close()
			client, err := newClient(server)
			if err != nil {
				t.Fatal(err)
			}
			links, err := client.GetRemoteLinks(context.Background(), "EXAMPLE-123")
			if err != nil {
				t.Fatal(err)
			}
			want := []RemoteLink{{GlobalID: "synthetic-global-id", Relationship: "implements", Object: &RemoteLinkObject{URL: "https://github.com/owner/repo/pull/42", Title: "owner/repo pull request #42"}}, {GlobalID: "legacy-id"}}
			if !reflect.DeepEqual(links, want) {
				t.Fatalf("links = %+v, want %+v", links, want)
			}
			if err := client.AddRemoteLink(context.Background(), "EXAMPLE-123", &want[0]); err != nil {
				t.Fatal(err)
			}
			wantJSON := map[string]interface{}{"globalId": "synthetic-global-id", "relationship": "implements", "object": map[string]interface{}{"url": "https://github.com/owner/repo/pull/42", "title": "owner/repo pull request #42"}}
			if !reflect.DeepEqual(posted, wantJSON) {
				t.Errorf("POST = %#v, want %#v", posted, wantJSON)
			}
		})
	}
}

func TestRemoteLinkAdapterErrors(t *testing.T) {
	for name, newClient := range remoteLinkAdapters() {
		for _, status := range []int{http.StatusNotFound, http.StatusForbidden, http.StatusInternalServerError} {
			t.Run(name+"/"+http.StatusText(status), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					json.NewEncoder(w).Encode(map[string]interface{}{"errorMessages": []string{"synthetic failure"}})
				}))
				defer server.Close()
				client, err := newClient(server)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := client.GetRemoteLinks(context.Background(), "EXAMPLE-123"); err == nil {
					t.Fatal("lookup error lost")
				}
				if err := client.AddRemoteLink(context.Background(), "EXAMPLE-123", &RemoteLink{Object: &RemoteLinkObject{URL: "https://github.com/owner/repo/issues/1", Title: "synthetic"}}); err == nil {
					t.Fatal("write error lost")
				}
			})
		}
	}
}

func TestConnectionLinkAdapterRepeatedWrites(t *testing.T) {
	for name, newClient := range remoteLinkAdapters() {
		t.Run(name, func(t *testing.T) {
			var mu sync.Mutex
			links := []cloud.RemoteLink{}
			reads, writes := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				if r.URL.Path != "/rest/api/2/issue/EXAMPLE-123/remotelink" {
					t.Errorf("unexpected endpoint: %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				switch r.Method {
				case http.MethodGet:
					reads++
					if err := json.NewEncoder(w).Encode(links); err != nil {
						t.Error(err)
					}
				case http.MethodPost:
					var posted cloud.RemoteLink
					if err := json.NewDecoder(r.Body).Decode(&posted); err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					if posted.GlobalID == "" || posted.Relationship == "" || posted.Object == nil || posted.Object.URL != "https://github.com/owner/repo/pull/42" {
						t.Error("incomplete or unnormalized remote link POST")
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					writes++
					found := false
					for i := range links {
						if links[i].GlobalID == posted.GlobalID {
							links[i] = posted
							found = true
							break
						}
					}
					if !found {
						links = append(links, posted)
					}
					w.WriteHeader(http.StatusCreated)
					if err := json.NewEncoder(w).Encode(map[string]int{"id": len(links)}); err != nil {
						t.Error(err)
					}
				default:
					t.Errorf("unexpected method: %s", r.Method)
					w.WriteHeader(http.StatusMethodNotAllowed)
				}
			}))
			defer server.Close()
			client, err := newClient(server)
			if err != nil {
				t.Fatal(err)
			}
			conn := &Connection{jiraClient: client}
			for i, step := range []struct {
				raw, relationship, status string
				dryRun                    bool
				writes, links             int
			}{
				{"https://github.com/Owner/REPO/pull/00042", DefaultRelationship, "linked", false, 1, 1},
				{"https://github.com/owner/repo/pull/42", DefaultRelationship, "already linked", false, 1, 1},
				{"https://github.com/owner/repo/pull/42", "fixes", "linked", false, 2, 2},
				{"https://github.com/OWNER/Repo/pull/0042", "fixes", "already linked", false, 2, 2},
				{"https://github.com/owner/repo/pull/42", "implements", "would link", true, 2, 2},
			} {
				results, err := conn.Link(context.Background(), "EXAMPLE-123", []string{step.raw}, step.relationship, step.dryRun)
				if err != nil {
					t.Fatal(err)
				}
				if len(results) != 1 || results[0].Status != step.status {
					t.Fatalf("step %d: results=%+v", i, results)
				}
				mu.Lock()
				gotReads, gotWrites, gotLinks := reads, writes, len(links)
				mu.Unlock()
				if gotReads != i+1 || gotWrites != step.writes || gotLinks != step.links {
					t.Fatalf("step %d: reads=%d writes=%d links=%d", i, gotReads, gotWrites, gotLinks)
				}
			}
		})
	}
}
