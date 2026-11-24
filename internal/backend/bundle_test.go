package backend

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBundleRootAnchoredToExecutable(t *testing.T) {
	p := t.TempDir()
	root := filepath.Join(p, "bundle")
	if e := os.MkdirAll(filepath.Join(root, "src"), 0700); e != nil {
		t.Fatal(e)
	}
	for _, n := range bundleScripts {
		if e := os.WriteFile(filepath.Join(root, "src", n), []byte("fixture"), 0600); e != nil {
			t.Fatal(e)
		}
	}
	for _, exe := range []string{filepath.Join(root, "dcr-manager.exe"), filepath.Join(root, "bin", "dcr-manager.exe")} {
		got, e := rootForExecutable(exe)
		if e != nil || got != root {
			t.Fatal(got, e)
		}
	}
	if _, e := rootForExecutable(filepath.Join(root, "unrelated", "dcr-manager.exe")); e == nil {
		t.Fatal("arbitrary parent accepted")
	}
	if e := os.Remove(filepath.Join(root, "src", "Bridge.ps1")); e != nil {
		t.Fatal(e)
	}
	if _, e := rootForExecutable(filepath.Join(root, "bin", "dcr-manager.exe")); e == nil {
		t.Fatal("incomplete bundle accepted")
	}
}
