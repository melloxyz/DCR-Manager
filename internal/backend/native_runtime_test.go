package backend

import (
	"context"
	"dcr-manager/internal/app"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTOMLAuditPrivacyAndSyntax(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.toml")
	raw := "[mcp_servers.\"private-server\"]\ncommand='npx.cmd'\nargs=['-y','@wonderwhy-er/desktop-commander@latest']\n[mcp_servers.\"private-server\".env]\nTOKEN='fixture-secret'\n[mcp_servers.direct]\ncommand='node.exe'\nargs=['C:\\node_modules\\@wonderwhy-er\\desktop-commander\\dist\\index.js']\n[mcp_servers.other]\ncommand='npx'\nargs=['@wonderwhy-er/desktop-commander-extra']\n"
	if e := os.WriteFile(p, []byte(raw), 0600); e != nil {
		t.Fatal(e)
	}
	a := auditTOML("Codex", p)
	if a.Status != "checked" || a.Entries != 2 || a.Floating != 1 || a.Modes != "direct, npx" {
		t.Fatal(a)
	}
	data, _ := json.Marshal(a)
	if strings.Contains(string(data), "private-server") || strings.Contains(string(data), "fixture-secret") {
		t.Fatal("leaked config")
	}
}
func TestTOMLAuditInvalidAndMissing(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.toml")
	if a := auditTOML("Codex", p); a.Status != "not-found" {
		t.Fatal(a)
	}
	for _, raw := range []string{"[mcp_servers.bad", "mcp_servers = []", "[mcp_servers.a]\ncommand='npx'\ncommand='node'", strings.Repeat(" ", maxConfigBytes+1)} {
		if e := os.WriteFile(p, []byte(raw), 0600); e != nil {
			t.Fatal(e)
		}
		if a := auditTOML("Codex", p); a.Status != "unreadable-or-unsupported" || a.Entries != 0 {
			t.Fatal(a)
		}
	}
	if e := os.WriteFile(p, []byte("model='fixture'\n"), 0600); e != nil {
		t.Fatal(e)
	}
	if a := auditTOML("Codex", p); a.Status != "checked" || a.Entries != 0 {
		t.Fatal(a)
	}
}
func TestRuntimeAssessment(t *testing.T) {
	for _, c := range []struct{ v, r, want string }{{"v24.15.0", ">=18", "meets-minimum"}, {"v18.0.0", ">=18.1", "below-minimum"}, {"v18.1.1", ">=18.1.2", "below-minimum"}, {"18.1.2", ">=18.1.2", "meets-minimum"}, {"v20.0.0", "^18 || ^20", "unknown"}, {"v20.0.0-rc.1", ">=18", "unknown"}, {"v999999999999999999999.0.0", ">=18", "unknown"}} {
		if got := assessNode(c.v, c.r); got != c.want {
			t.Fatal(c, got)
		}
	}
}
func TestNativeRuntimeSanitizesOutput(t *testing.T) {
	s := app.Snapshot{Installations: []app.Installation{{Kind: "dedicated", NodeRange: ">=18"}}}
	run := func(_ context.Context, name string) (string, error) {
		if name == "node.exe" {
			return "v24.15.0\r\n", nil
		}
		return "fixture-secret", nil
	}
	applyRuntime(context.Background(), &s, run)
	if !s.Status.NodeAvailable || s.Status.NpmAvailable || s.Runtime.Node != "v24.15.0" || s.Runtime.Npm != "unknown" || s.Runtime.Assessment != "meets-minimum" {
		t.Fatal(s.Runtime, s.Status)
	}
	run = func(context.Context, string) (string, error) { return "", errRuntimeMissing }
	applyRuntime(context.Background(), &s, run)
	if s.Status.NodeAvailable || s.Status.NpmAvailable || s.Runtime.Node != "missing" {
		t.Fatal(s)
	}
	run = func(context.Context, string) (string, error) { return "fixture-secret", errors.New("fixture-private") }
	applyRuntime(context.Background(), &s, run)
	if s.Runtime.Node != "unknown" || s.Runtime.Npm != "unknown" {
		t.Fatal(s.Runtime)
	}
}
