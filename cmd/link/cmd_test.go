package link

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/oceanc80/gh2jira/pkg/jira"
	"github.com/spf13/cobra"
)

func TestInvalidBatchBeforeConfigReads(t *testing.T) {
	for _, args := range [][]string{
		{}, {"EXAMPLE-123"}, {"invalid", "https://github.com/owner/repo/issues/1"},
		{"EXAMPLE-123", "https://github.com/owner/repo/issues/1", "invalid"},
		{"EXAMPLE-123", "https://github.com/owner/repo/issues/1", "--relationship", "blocks"},
		{"EXAMPLE-123", "https://github.com/owner/repo/issues/1", "https://github.com.evil.invalid/owner/repo/pull/2", "--dryrun"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			loads := 0
			cmd := newCmd(func(*cobra.Command) (linker, error) {
				loads++
				return nil, errors.New("config loader called")
			})
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetArgs(args)
			if err := cmd.Execute(); err == nil {
				t.Fatal("invalid batch succeeded")
			}
			if loads != 0 {
				t.Fatal("invalid batch read configuration")
			}
		})
	}
}

type recordingLinker struct {
	key, relationship string
	urls              []string
	dryRun            bool
	err               error
}

func (r *recordingLinker) Link(_ context.Context, key string, urls []string, relationship string, dryRun bool) ([]jira.LinkResult, error) {
	r.key, r.urls, r.relationship, r.dryRun = key, urls, relationship, dryRun
	return []jira.LinkResult{{URL: urls[0], Relationship: relationship, Status: "would link"}}, r.err
}

func TestLinkCommand(t *testing.T) {
	for _, tt := range []struct {
		name, relationship string
		dryRun             bool
		err                error
	}{
		{name: "default", relationship: jira.DefaultRelationship},
		{name: "explicit dry run", relationship: "implements", dryRun: true},
		{name: "service error", relationship: jira.DefaultRelationship, err: errors.New("write failed")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := &recordingLinker{err: tt.err}
			cmd := newCmd(func(*cobra.Command) (linker, error) { return f, nil })
			args := []string{"EXAMPLE-123", "https://github.com/owner/repo/pull/1"}
			if tt.relationship != jira.DefaultRelationship {
				args = append(args, "--relationship", tt.relationship)
			}
			if tt.dryRun {
				args = append(args, "--dryrun")
			}
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetArgs(args)
			if err := cmd.Execute(); !errors.Is(err, tt.err) {
				t.Fatalf("error = %v, want %v", err, tt.err)
			}
			if f.key != "EXAMPLE-123" || len(f.urls) != 1 || f.relationship != tt.relationship || f.dryRun != tt.dryRun {
				t.Fatalf("wrong service arguments: %+v", f)
			}
			if !strings.Contains(out.String(), "EXAMPLE-123 "+tt.relationship+" https://github.com/owner/repo/pull/1") {
				t.Errorf("missing result: %q", out.String())
			}
		})
	}
}

func TestConfigErrorPropagates(t *testing.T) {
	want := errors.New("configuration failed")
	cmd := newCmd(func(*cobra.Command) (linker, error) { return nil, want })
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"EXAMPLE-123", "https://github.com/owner/repo/issues/1"})
	if err := cmd.Execute(); !errors.Is(err, want) {
		t.Fatalf("error = %v", err)
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func TestOutputFailurePreservesServiceError(t *testing.T) {
	serviceErr := errors.New("second link failed")
	outputErr := errors.New("output failed")
	f := &recordingLinker{err: serviceErr}
	cmd := newCmd(func(*cobra.Command) (linker, error) { return f, nil })
	cmd.SetOut(failingWriter{err: outputErr})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"EXAMPLE-123", "https://github.com/owner/repo/issues/1"})
	if err := cmd.Execute(); !errors.Is(err, serviceErr) || !errors.Is(err, outputErr) {
		t.Fatalf("error lost: %v", err)
	}
}
