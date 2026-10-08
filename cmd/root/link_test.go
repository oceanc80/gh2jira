package root

import (
	"bytes"
	"errors"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinkInvalidURLDiagnosticsRedactInput(t *testing.T) {
	for _, tt := range []struct {
		name, raw, reason string
		secrets           []string
	}{
		{"userinfo", "https://synthetic-user:synthetic-userinfo-secret@github.com/owner/repo/issues/1", "userinfo is not supported", []string{"synthetic-user", "synthetic-userinfo-secret"}},
		{"query", "https://github.com/owner/repo/issues/1?synthetic-querytoken=synthetic-query-secret", "queries are not supported", []string{"synthetic-querytoken", "synthetic-query-secret"}},
		{"malformed escape", "https://github.com/owner/repo/issues/synthetic-malformed-secret%zz", "malformed URL", []string{"synthetic-malformed-secret", "%zz"}},
		{"malformed host", "https://github.com:synthetic-port-secret/owner/repo/issues/1", "malformed URL", []string{"synthetic-port-secret"}},
		{"invalid path", "https://github.com/owner/synthetic-path-secret/issues/not-a-number", "expected owner/repo/issues/N or owner/repo/pull/N", []string{"synthetic-path-secret", "not-a-number"}},
		{"nonpositive number", "https://github.com/owner/synthetic-zero-secret/issues/000", "number must be positive", []string{"synthetic-zero-secret"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, dryRun := range []bool{false, true} {
				cmd := NewCmd()
				var out, stderr bytes.Buffer
				cmd.SetOut(&out)
				cmd.SetErr(&stderr)
				// Missing config files make any premature config read fail with
				// a distinct error. Never use the user's default files.
				dir := t.TempDir()
				args := []string{"link", "EXAMPLE-123", "https://github.com/owner/repo/pull/42", tt.raw,
					"--profile-name", "synthetic", "--profiles-file", filepath.Join(dir, "missing-profiles.yaml"),
					"--token-file", filepath.Join(dir, "missing-tokenstore.yaml")}
				if dryRun {
					args = append(args, "--dryrun")
				}
				cmd.SetArgs(args)
				err := cmd.Execute()
				if err == nil {
					t.Fatal("invalid URL must return an error for a nonzero CLI exit")
				}
				if !strings.Contains(err.Error(), "invalid GitHub URL at argument 3") || !strings.Contains(err.Error(), tt.reason) {
					t.Fatal("expected positional validation failure before config loading")
				}
				var parseErr *url.Error
				if errors.As(err, &parseErr) {
					t.Fatal("raw URL parser error escaped validation")
				}
				for name, diagnostic := range map[string]string{"returned error": err.Error(), "stdout": out.String(), "stderr": stderr.String()} {
					for _, sensitive := range append(tt.secrets, tt.raw) {
						if strings.Contains(diagnostic, sensitive) {
							t.Errorf("%s contains rejected URL content", name)
						}
					}
				}
			}
		})
	}
}
