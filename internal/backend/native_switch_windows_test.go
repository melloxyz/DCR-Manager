//go:build windows

package backend

import (
	"context"
	"dcr-manager/internal/app"
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestNativePackageSwitchRecoveryParity(t *testing.T) {
	for _, mode := range []string{"completed", "switch-failed", "rollback-incomplete"} {
		t.Run(mode, func(t *testing.T) {
			root, state, p := switchFixture(t)
			ops := nativeSwitchOps()
			calls := 0
			ops.move = func(a, b string) error {
				calls++
				if mode == "switch-failed" && calls == 3 || mode == "rollback-incomplete" && calls >= 3 {
					return errors.New("injected")
				}
				return movePackageNoReplace(a, b)
			}
			_, e := switchNativePackage(context.Background(), root, state, p, "0.3.4", func(context.Context, string) error { return nil }, ops)
			if (e == nil) != (mode == "completed") {
				t.Fatal("unexpected transaction result", e)
			}
			if switchStatus(t, p.Stage) != mode {
				t.Fatal("unexpected journal state")
			}
			cache := t.TempDir()
			native, known := nativeRecovery(context.Background(), state, cache)
			if !known || len(native) != 2 {
				t.Fatal("native recovery inventory incomplete")
			}
			module, e := filepath.Abs("../../src/RecoveryCenter.psm1")
			if e != nil {
				t.Fatal(e)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", "$ErrorActionPreference='Stop';$WarningPreference='SilentlyContinue';$ProgressPreference='SilentlyContinue';[Console]::OutputEncoding=New-Object System.Text.UTF8Encoding($false);Import-Module $env:DCR_TEST_MODULE;ConvertTo-Json -InputObject @(Get-DcrRecoveryInventory $env:DCR_TEST_STATE $env:DCR_TEST_CACHE) -Compress")
			cmd.Env = append(legacyPowerShellEnv(), "DCR_TEST_MODULE="+module, "DCR_TEST_STATE="+state, "DCR_TEST_CACHE="+cache)
			configure(cmd)
			out, e := cmd.Output()
			if e != nil {
				t.Fatal("legacy fixture inventory failed", e)
			}
			var legacy []app.Backup
			if e = json.Unmarshal([]byte(strings.TrimPrefix(strings.TrimSpace(string(out)), "\uFEFF")), &legacy); e != nil {
				t.Fatal("invalid legacy fixture JSON")
			}
			if len(legacy) != len(native) {
				t.Fatal("inventory counts differ")
			}
			by := map[string]app.Backup{}
			for _, r := range legacy {
				by[r.Name] = r
			}
			for _, r := range native {
				old := by[r.Name]
				if r.Status != old.Status || r.Version != old.Version || r.Token != old.Token || r.Bytes != old.Bytes {
					t.Fatalf("recovery parity mismatch: kind=%s status=%s", r.Kind, r.Status)
				}
			}
		})
	}
}

func TestNativePackageSwitchLockedFile(t *testing.T) {
	root, state, p := switchFixture(t)
	path, e := windows.UTF16PtrFromString(filepath.Join(root, "package.json"))
	if e != nil {
		t.Fatal(e)
	}
	handle, e := windows.CreateFile(path, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if e != nil {
		t.Fatal(e)
	}
	defer windows.CloseHandle(handle)
	_, e = switchNativePackage(context.Background(), root, state, p, "0.3.4", func(context.Context, string) error { return nil }, nativeSwitchOps())
	if e == nil || stageVersion(root) != "0.3.4" || switchStatus(t, p.Stage) != "switch-failed" {
		t.Fatal("locked fixture not safely restored", e)
	}
}
