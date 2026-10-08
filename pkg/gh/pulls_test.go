package gh

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type pullTransport func(*http.Request) (*http.Response, error)

func (f pullTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGetPullRequest(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		merged     bool
		wantErr    bool
	}{
		{"merged", `{"number":1,"state":"closed","merged":true}`, 200, true, false},
		{"unmerged", `{"number":1,"state":"open","merged":false}`, 200, false, false},
		{"closed unmerged", `{"number":1,"state":"closed","merged":false}`, 200, false, false},
		{"not found", `{"message":"Not Found"}`, 404, false, true},
		{"server failure", `{"message":"Unavailable"}`, 500, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reads := 0
			ctx := context.WithValue(context.Background(), struct{}{}, "report")
			client := &http.Client{Transport: pullTransport(func(r *http.Request) (*http.Response, error) {
				reads++
				require.Equal(t, "GET", r.Method)
				require.Equal(t, "https://api.github.com/repos/a/b/pulls/1", r.URL.String())
				require.Equal(t, "report", r.Context().Value(struct{}{}))
				return &http.Response{StatusCode: tc.status, Status: fmt.Sprint(tc.status), Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(tc.body)), Request: r}, nil
			})}
			conn, err := NewConnection(WithTransport(client))
			require.NoError(t, err)
			require.NoError(t, conn.Connect())
			pr, err := conn.GetPullRequest(ctx, "a", "b", 1)
			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, pr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, pr.Merged)
				require.Equal(t, tc.merged, pr.GetMerged())
			}
			require.Equal(t, 1, reads)
		})
	}
}

func TestGetPullRequestDisconnected(t *testing.T) {
	conn, err := NewConnection()
	require.NoError(t, err)
	_, err = conn.GetPullRequest(context.Background(), "a", "b", 1)
	require.ErrorContains(t, err, "not connected")
}
