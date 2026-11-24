package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNativeLogPrivacyAndLimits(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "_logs")
	if e := os.Mkdir(dir, 0700); e != nil {
		t.Fatal(e)
	}
	write := func(name, body string) {
		t.Helper()
		if e := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); e != nil {
			t.Fatal(e)
		}
	}
	write("2026-11-24T11_43_31_950Z-debug-0.log", "@wonderwhy-er/desktop-commander EBUSY EBUSY EPERM token=fixture-private")
	write("2026-11-24T11_44_31_950Z-debug-0.log", "other package ENOTFOUND")
	write("fixture-private-debug.log", "@wonderwhy-er/desktop-commander EACCES")
	logs, known := nativeLogs(context.Background(), root)
	if !known || len(logs) != 1 || strings.Join(logs[0].Codes, ",") != "EBUSY,EPERM" {
		t.Fatal(logs, known)
	}
	raw, _ := json.Marshal(logs)
	if strings.Contains(string(raw), "fixture-private") || strings.Contains(string(raw), "ENOTFOUND") {
		t.Fatal("private or irrelevant data")
	}
	write("2026-11-24T11_45_31_950Z-debug-0.log", strings.Repeat("x", 2*1024*1024+1))
	logs, known = nativeLogs(context.Background(), root)
	if known || len(logs) != 1 {
		t.Fatal("partial scan expected", known, len(logs))
	}
}
func TestNativeLogEmptyUnknownAndCancelled(t *testing.T) {
	if _, known := nativeLogs(context.Background(), ""); known {
		t.Fatal("unknown root")
	}
	if _, known := nativeLogs(context.Background(), "relative"); known {
		t.Fatal("relative root")
	}
	root := t.TempDir()
	if logs, known := nativeLogs(context.Background(), root); !known || len(logs) != 0 {
		t.Fatal("empty log directory")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, known := nativeLogs(ctx, root); known {
		t.Fatal("cancelled scan")
	}
}
func TestNativeJSONAuditRedacted(t *testing.T) {
	p := filepath.Join(t.TempDir(), "mcp.json")
	raw := []byte("{\"mcpServers\":{\"private-name\":{\"command\":\"npx.cmd\",\"args\":[\"-y\",\"@wonderwhy-er/desktop-commander@latest\"],\"env\":{\"TOKEN\":\"fixture-private\"}},\"other\":{\"command\":\"npx\",\"args\":[\"@wonderwhy-er/desktop-commander-extra\"]}},\"servers\":{\"second\":{\"command\":\"node.exe\",\"args\":[\"C:/node_modules/@wonderwhy-er/desktop-commander/dist/index.js\"]}}}")
	if e := os.WriteFile(p, raw, 0600); e != nil {
		t.Fatal(e)
	}
	a := auditJSON("fixture", p)
	if a.Status != "checked" || a.Entries != 2 || a.Floating != 1 || a.Modes != "direct, npx" {
		t.Fatal(a)
	}
	j, _ := json.Marshal(a)
	if strings.Contains(string(j), "fixture-private") || strings.Contains(string(j), "private-name") {
		t.Fatal("leak")
	}
}
func TestNativeJSONAuditUnavailable(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "mcp.json")
	if a := auditJSON("fixture", p); a.Status != "not-found" {
		t.Fatal(a)
	}
	for _, raw := range []string{"{invalid", "[]", "null", "{\"servers\":[]}", strings.Repeat(" ", 5*1024*1024+1)} {
		if e := os.WriteFile(p, []byte(raw), 0600); e != nil {
			t.Fatal(e)
		}
		if a := auditJSON("fixture", p); a.Status != "unreadable-or-unsupported" || a.Entries != 0 {
			t.Fatal(a)
		}
	}
}

func TestNativeLogLatestTwenty(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "_logs")
	if e := os.Mkdir(dir, 0700); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 25; i++ {
		p := filepath.Join(dir, fmt.Sprintf("2026-11-24T11_43_31_950Z-debug-%d.log", i))
		if e := os.WriteFile(p, []byte("@wonderwhy-er/desktop-commander EACCES"), 0600); e != nil {
			t.Fatal(e)
		}
		ts := time.Unix(int64(1700000000+i*2), 0)
		if e := os.Chtimes(p, ts, ts); e != nil {
			t.Fatal(e)
		}
	}
	rows, known := nativeLogs(context.Background(), root)
	if !known || len(rows) != 20 || rows[0].Name != "2026-11-24T11_43_31_950Z-debug-24.log" || rows[19].Name != "2026-11-24T11_43_31_950Z-debug-5.log" {
		t.Fatal(rows, known)
	}
}
func TestNativeDiagnosticsRejectLinkedFiles(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "actual.json")
	if e := os.WriteFile(target, []byte("{\"servers\":{}}"), 0600); e != nil {
		t.Fatal(e)
	}
	link := filepath.Join(root, "linked.json")
	if e := os.Symlink(target, link); e != nil {
		t.Skip("symlink privilege unavailable")
	}
	if got := auditJSON("fixture", link); got.Status != "unreadable-or-unsupported" {
		t.Fatal(got)
	}
}
func TestNativeJSONMissingRootsNotHealthy(t *testing.T) {
	rows := jsonClientAudits("", "")
	if len(rows) != 6 {
		t.Fatal(rows)
	}
	for _, r := range rows {
		if r.Status != "unreadable-or-unsupported" {
			t.Fatal(r)
		}
	}
}
