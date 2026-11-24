//go:build windows

package backend

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"dcr-manager/internal/app"
	"golang.org/x/sys/windows"
)

func TestWindowsIsolatedLifecycle(t *testing.T) {
	node, err := exec.LookPath("node.exe")
	if err != nil {
		t.Skip("Node.js is required for isolated lifecycle fixture")
	}
	node, err = filepath.Abs(node)
	if err != nil {
		t.Fatal(err)
	}
	// Share the production mutex so no manager can act on this short-lived fixture.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	name, _ := windows.UTF16PtrFromString("Local\\DCRManager")
	lock, err := windows.CreateMutex(nil, false, name)
	if err != nil && lock == 0 {
		t.Fatal(err)
	}
	defer windows.CloseHandle(lock)
	state, err := windows.WaitForSingleObject(lock, 0)
	if err != nil || state != windows.WAIT_OBJECT_0 && state != windows.WAIT_ABANDONED {
		t.Fatal("maintenance mutex unavailable")
	}
	defer windows.ReleaseMutex(lock)
	dir := t.TempDir()
	entry := filepath.Join(dir, "node_modules", "@wonderwhy-er", "desktop-commander", "dist", "index.js")
	if err = os.MkdirAll(filepath.Dir(entry), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(entry, []byte("setInterval(()=>{},1000);\n"), 0600); err != nil {
		t.Fatal(err)
	}
	spec := launchSpec{node: node, entry: entry, dir: dir}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	// A second launch after verified exit exercises start-stop-start-stop.
	for cycle := 0; cycle < 2; cycle++ {
		func() {
			h, id, e := launchRemote(spec)
			if e != nil {
				t.Fatal(e)
			}
			defer windows.CloseHandle(h)
			defer func() { _ = terminateBound(context.Background(), h) }()
			args, e := commandArgs(h)
			if e != nil {
				t.Fatal(e)
			}
			if !dcrEntry(args) || len(args) < 3 || args[2] != "remote" {
				t.Fatal("fixture command not recognized")
			}
			target := nativeCandidate{ID: id, Args: args}
			bad := target
			bad.ID.Created++
			if x, e := bindTarget(bad); e == nil {
				windows.CloseHandle(x)
				t.Fatal("reused identity accepted")
			}
			if s, e := windows.WaitForSingleObject(h, 200); e != nil || s != uint32(windows.WAIT_TIMEOUT) {
				t.Fatal("fixture did not remain running")
			}
			if err := stopExact(ctx, []nativeCandidate{target, bad}, func() error { return nil }); err == nil {
				t.Fatal("invalid second target accepted")
			}
			if state, _ := windows.WaitForSingleObject(h, 0); state != uint32(windows.WAIT_TIMEOUT) {
				t.Fatal("preflight failure stopped first target")
			}
			if err := stopExact(ctx, []nativeCandidate{target}, func() error { return errors.New("set changed") }); err == nil {
				t.Fatal("verification failure ignored")
			}
			if state, _ := windows.WaitForSingleObject(h, 0); state != uint32(windows.WAIT_TIMEOUT) {
				t.Fatal("failed verification stopped target")
			}
			if err := stopExact(ctx, []nativeCandidate{target}, func() error { return nil }); err != nil {
				t.Fatal(err)
			}
			if s, e := windows.WaitForSingleObject(h, 0); e != nil || s != windows.WAIT_OBJECT_0 {
				t.Fatal("fixture did not stop")
			}
		}()
	}
}

func TestWindowsNativeRefusesMismatchedIdentity(t *testing.T) {
	// Start with a non-empty expectation is rejected before accessing the OS.
	op := Operate("")
	if err := op(context.Background(), app.Start, []app.ProcessIdentity{{PID: 1, Created: 1}}); err == nil {
		t.Fatal("invalid start accepted")
	}
}

func TestWindowsNativeLiveMismatchRefused(t *testing.T) {
	// A fabricated identity must never match a real Windows creation FILETIME.
	err := Operate("")(context.Background(), app.Stop, []app.ProcessIdentity{{PID: 2147483647, Created: 1}})
	if err == nil || !strings.Contains(err.Error(), "identities changed") {
		t.Fatalf("expected live identity refusal, got %v", err)
	}
}
