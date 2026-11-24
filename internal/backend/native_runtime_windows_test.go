//go:build windows

package backend

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNpmVersionCommandFixedArguments(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "fixture with spaces (x64)")
	if e := os.Mkdir(dir, 0700); e != nil {
		t.Fatal(e)
	}
	p := filepath.Join(dir, "npm.cmd")
	if e := os.WriteFile(p, []byte("@echo off\r\nif not \"%1\"==\"--version\" exit /b 9\r\necho 11.12.1\r\n"), 0600); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd, e := npmVersionCommand(ctx, p)
	if e != nil {
		t.Fatal(e)
	}
	out, e := cmd.Output()
	if e != nil || strings.TrimSpace(string(out)) != "11.12.1" {
		t.Fatal("fixture invocation failed", e)
	}
	for _, p := range []string{"npm.cmd", filepath.Join(dir, "other.cmd"), filepath.Join(dir, "bad%PATH%", "npm.cmd"), filepath.Join(dir, "bad&", "npm.cmd")} {
		if _, e := npmVersionCommand(ctx, p); e == nil {
			t.Fatal("unsafe launcher accepted")
		}
	}
}
