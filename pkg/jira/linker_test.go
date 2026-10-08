package jira

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestValidateLinkArguments(t *testing.T) {
	const valid = "https://github.com/owner/repo/issues/123"
	for _, tt := range []struct {
		name, key, relationship string
		urls                    []string
		want                    []string
	}{
		{"issue", "EXAMPLE-123", "relates to", []string{valid}, []string{valid}},
		{"normalize and deduplicate", "EXAMPLE-123", "implements", []string{"https://github.com/Owner/Repo/pull/00042", "https://github.com/owner/repo/pull/42"}, []string{"https://github.com/owner/repo/pull/42"}},
		{"multiple repositories and kinds", "EXAMPLE_2-123", "fixes", []string{valid, "https://github.com/another-owner/repo_2.git/pull/9", "https://github.com/owner/repo/pull/123"}, []string{valid, "https://github.com/another-owner/repo_2.git/pull/9", "https://github.com/owner/repo/pull/123"}},
		{"dependency", "EXAMPLE-123", "depends on", []string{valid}, []string{valid}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			links, err := ValidateLinkArguments(tt.key, tt.urls, tt.relationship)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, link := range links {
				got = append(got, link.Object.URL)
				if link.Relationship != tt.relationship || link.Object.Title == "" || !strings.HasPrefix(link.GlobalID, "gh2jira:github:") {
					t.Errorf("incomplete link: %+v", link)
				}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("URLs = %v, want %v", got, tt.want)
			}
		})
	}

	for _, key := range []string{"", "example-123", "EXAMPLE", "EXAMPLE-0", "EXAMPLE-001", "1EXAMPLE-123", "EXAMPLE-123/other", "EXAMPLE-123 ", "EXAMPLE-123 OR key=OTHER-1"} {
		t.Run("invalid key "+key, func(t *testing.T) {
			if _, err := ValidateLinkArguments(key, []string{valid}, DefaultRelationship); err == nil {
				t.Fatal("accepted invalid Jira key")
			}
		})
	}
	for _, raw := range []string{
		"", "not a URL", "http://github.com/owner/repo/issues/1", "https://github.com.evil.invalid/owner/repo/issues/1",
		"https://www.github.com/owner/repo/issues/1", "https://github.com:443/owner/repo/issues/1",
		"https://user@github.com/owner/repo/issues/1", "https://github.com@evil.invalid/owner/repo/issues/1",
		"https://github.com./owner/repo/issues/1", "https://github.com/owner/repo/issues/1?x=1", "https://github.com/owner/repo/issues/1?",
		"https://github.com/owner/repo/issues/1#comment", "https://github.com/owner/repo/issues/1#",
		"https://github.com/owner%2Frepo/issues/1", "https://github.com/owner/repo/issues/%31", "https://github.com/owner/repo/issues/1%2f",
		"https://github.com/owner/repo/issues/1%5c", "https://github.com/owner/repo/issues/1%252f",
		"https://github.com/owner/repo/issues/0", "https://github.com/owner/repo/issues/000", "https://github.com/owner/repo/issues/-1",
		"https://github.com/owner/repo/issues/+1", "https://github.com/owner/repo/issues/1.0", "https://github.com/owner/repo/issues/one",
		"https://github.com/owner/repo/issues/1/", "https://github.com/owner/repo/pull/1/files", "https://github.com/owner/repo/pulls/1",
		"https://github.com//repo/issues/1", "https://github.com/owner//issues/1", "https://github.com/owner/../issues/1",
		"https://github.com/owner/./issues/1", "https://github.com/-owner/repo/issues/1", "https://github.com/owner_/repo/issues/1",
		"https://github.com/owner/re po/issues/1", "https://github.com/owner/repo\\other/issues/1", "https://github.com/owner/repo/issues/1\n",
	} {
		t.Run("invalid URL "+raw, func(t *testing.T) {
			// A valid first URL must not make an invalid batch acceptable.
			if _, err := ValidateLinkArguments("EXAMPLE-123", []string{valid, raw}, DefaultRelationship); err == nil {
				t.Fatal("accepted invalid URL batch")
			}
		})
	}
	for _, relationship := range []string{"", "blocks", "Fixes", "relates-to"} {
		if _, err := ValidateLinkArguments("EXAMPLE-123", []string{valid}, relationship); err == nil {
			t.Errorf("accepted relationship %q", relationship)
		}
	}
	if _, err := ValidateLinkArguments("EXAMPLE-123", nil, DefaultRelationship); err == nil {
		t.Error("accepted empty URL batch")
	}
}

func TestLinkGlobalIDStableFixtures(t *testing.T) {
	// Fixed fixtures pin the identity scheme without recomputing it in the test.
	const neutralID = "gh2jira:github:9da1eea2e6a674cda7a1233f5f02c462f8d6119eae7b5230c3f1dd11e0788852"
	const fixesID = "gh2jira:github:5d56996242b66cb8b96d752e5b8d31c37c1b2e6c720aa6fc8988b1fa0d8aa608"
	for _, tt := range []struct {
		name, raw, relationship, wantID string
	}{
		{"canonical", "https://github.com/owner/repo/pull/42", DefaultRelationship, neutralID},
		{"case and leading zeros", "https://github.com/Owner/REPO/pull/00042", DefaultRelationship, neutralID},
		{"different relationship", "https://github.com/owner/repo/pull/42", "fixes", fixesID},
		{"equivalent different relationship", "https://github.com/OWNER/Repo/pull/0042", "fixes", fixesID},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Separate calls ensure deduplication cannot hide a normalization bug.
			links, err := ValidateLinkArguments("EXAMPLE-123", []string{tt.raw}, tt.relationship)
			if err != nil {
				t.Fatal(err)
			}
			if len(links) != 1 || links[0].GlobalID != tt.wantID {
				t.Fatalf("global ID changed: links=%+v, want %s", links, tt.wantID)
			}
		})
	}
}

type fakeLinkClient struct {
	failAt                           int
	links                            []RemoteLink
	reads, writes, creates, searches int
	readErr, writeErr                error
	writeKeys                        []string
}

func (f *fakeLinkClient) SearchIssues(context.Context, string) ([]Issue, error) {
	f.searches++
	return nil, errors.New("unexpected issue search")
}

func (f *fakeLinkClient) CreateIssue(context.Context, *Issue) (*Issue, error) {
	f.creates++
	return nil, errors.New("unexpected issue creation")
}

func (f *fakeLinkClient) GetRemoteLinks(context.Context, string) ([]RemoteLink, error) {
	f.reads++
	return f.links, f.readErr
}

func (f *fakeLinkClient) AddRemoteLink(_ context.Context, key string, link *RemoteLink) error {
	f.writes++
	f.writeKeys = append(f.writeKeys, key)
	if f.writeErr != nil && (f.failAt == 0 || f.writes == f.failAt) {
		return f.writeErr
	}
	f.links = append(f.links, *link)
	return nil
}

func TestLink(t *testing.T) {
	const raw = "https://github.com/Owner/Repo/pull/00042"
	const normalized = "https://github.com/owner/repo/pull/42"
	lookupErr := errors.New("lookup failed")
	writeErr := errors.New("write failed")
	for _, tt := range []struct {
		name              string
		links             []RemoteLink
		dryRun            bool
		readErr, writeErr error
		wantStatus        string
		wantWrites        int
		wantErr           error
	}{
		{name: "new", wantStatus: "linked", wantWrites: 1},
		{name: "dry run", dryRun: true, wantStatus: "would link"},
		{name: "legacy neutral", links: []RemoteLink{{Object: &RemoteLinkObject{URL: raw, Title: "keep legacy title"}}}, wantStatus: "already linked"},
		{name: "legacy duplicates", links: []RemoteLink{{Object: &RemoteLinkObject{URL: raw}}, {Object: &RemoteLinkObject{URL: normalized}}}, wantStatus: "already linked"},
		{name: "legacy explicit relationship", links: []RemoteLink{{GlobalID: "foreign-id", Relationship: DefaultRelationship, Object: &RemoteLinkObject{URL: raw}}}, wantStatus: "already linked"},
		{name: "different relationship", links: []RemoteLink{{Relationship: "fixes", Object: &RemoteLinkObject{URL: normalized}}}, wantStatus: "linked", wantWrites: 1},
		{name: "unrelated and nil objects", links: []RemoteLink{{GlobalID: "foreign-id"}, {Object: &RemoteLinkObject{URL: "https://example.invalid/private"}}}, wantStatus: "linked", wantWrites: 1},
		{name: "lookup error", readErr: lookupErr, wantErr: lookupErr},
		{name: "dry run lookup error", dryRun: true, readErr: lookupErr, wantErr: lookupErr},
		{name: "write error", writeErr: writeErr, wantWrites: 1, wantErr: writeErr},
	} {
		t.Run(tt.name, func(t *testing.T) {
			before := append([]RemoteLink(nil), tt.links...)
			for i := range before {
				if before[i].Object != nil {
					object := *before[i].Object
					before[i].Object = &object
				}
			}
			f := &fakeLinkClient{links: append([]RemoteLink(nil), tt.links...), readErr: tt.readErr, writeErr: tt.writeErr}
			conn := &Connection{jiraClient: f}
			results, err := conn.Link(context.Background(), "EXAMPLE-123", []string{raw, normalized}, DefaultRelationship, tt.dryRun)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if f.writes != tt.wantWrites || f.reads != 1 || f.creates != 0 || f.searches != 0 {
				t.Fatalf("unexpected calls: %+v", f)
			}
			if !reflect.DeepEqual(f.links[:len(before)], before) && len(before) > 0 {
				t.Fatal("modified existing links")
			}
			if err == nil && (len(results) != 1 || results[0].Status != tt.wantStatus || results[0].URL != normalized) {
				t.Fatalf("results = %+v", results)
			}
			for _, key := range f.writeKeys {
				if key != "EXAMPLE-123" {
					t.Errorf("write key = %s", key)
				}
			}
		})
	}
}

func TestLinkRepeatedAndDifferentRelationships(t *testing.T) {
	f := &fakeLinkClient{}
	conn := &Connection{jiraClient: f}
	for _, relationship := range []string{DefaultRelationship, DefaultRelationship, "fixes", "fixes"} {
		if _, err := conn.Link(context.Background(), "EXAMPLE-123", []string{"https://github.com/Owner/Repo/issues/001"}, relationship, false); err != nil {
			t.Fatal(err)
		}
	}
	if f.writes != 2 || len(f.links) != 2 || f.links[0].GlobalID == f.links[1].GlobalID || f.creates != 0 {
		t.Fatalf("not idempotent per relationship: %+v", f)
	}
}

func TestLinkValidationBeforeConnection(t *testing.T) {
	f := &fakeLinkClient{}
	for _, conn := range []*Connection{{jiraClient: f}, {}} {
		_, err := conn.Link(context.Background(), "EXAMPLE-123", []string{"https://github.com/owner/repo/issues/1", "invalid"}, DefaultRelationship, false)
		if err == nil || !strings.Contains(err.Error(), "invalid GitHub URL") {
			t.Fatalf("validation must precede connection: %v", err)
		}
	}
	if f.reads != 0 || f.writes != 0 || f.creates != 0 {
		t.Fatal("invalid batch reached Jira")
	}
}

func TestLinkIDConflictBeforeWrites(t *testing.T) {
	urls := []string{"https://github.com/owner/repo/issues/1", "https://github.com/owner/repo/pull/2"}
	planned, err := ValidateLinkArguments("EXAMPLE-123", urls, DefaultRelationship)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeLinkClient{links: []RemoteLink{{GlobalID: planned[1].GlobalID, Object: &RemoteLinkObject{URL: "https://example.invalid/unrelated"}}}}
	conn := &Connection{jiraClient: f}
	if _, err := conn.Link(context.Background(), "EXAMPLE-123", urls, DefaultRelationship, false); err == nil {
		t.Fatal("accepted conflicting global ID")
	}
	if f.writes != 0 {
		t.Fatal("wrote before planning entire batch")
	}
}

func TestLinkPartialFailureAndRetry(t *testing.T) {
	writeErr := errors.New("second write failed")
	f := &fakeLinkClient{writeErr: writeErr, failAt: 2}
	conn := &Connection{jiraClient: f}
	urls := []string{"https://github.com/owner/repo/issues/1", "https://github.com/owner/another-repo/pull/2"}
	results, err := conn.Link(context.Background(), "EXAMPLE-123", urls, "fixes", false)
	if !errors.Is(err, writeErr) || len(results) != 1 || results[0].Status != "linked" || len(f.links) != 1 {
		t.Fatalf("partial progress lost: results=%+v links=%+v error=%v", results, f.links, err)
	}
	f.writeErr = nil
	results, err = conn.Link(context.Background(), "EXAMPLE-123", urls, "fixes", false)
	if err != nil || len(results) != 2 || results[0].Status != "already linked" || results[1].Status != "linked" || len(f.links) != 2 || f.writes != 3 || f.creates != 0 {
		t.Fatalf("retry failed: results=%+v calls=%+v error=%v", results, f, err)
	}
}
