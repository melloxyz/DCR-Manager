//go:build windows

package backend

import (
	"context"
	"dcr-manager/internal/app"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestNativeRestoreRecoveryParity(t *testing.T) {
	for _, kind := range []string{"package", "cache"} {
		t.Run(kind, func(t *testing.T) {
			root, state, cache, _, r := restoreFixture(t, kind)
			if _, e := restoreNativeBackup(context.Background(), root, state, cache, r, quarantineGuard, nativeRestoreOps()); e != nil {
				t.Fatal(e)
			}
			native, known := nativeRecovery(context.Background(), state, cache)
			if !known {
				t.Fatal("native inventory unavailable")
			}
			inv := scanInstallations(context.Background(), "", "", cache)
			if !inv.cacheKnown {
				t.Fatal("cache inventory unavailable")
			}
			module, e := filepath.Abs("../../src/RecoveryCenter.psm1")
			if e != nil {
				t.Fatal(e)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", "$ErrorActionPreference='Stop';$WarningPreference='SilentlyContinue';$ProgressPreference='SilentlyContinue';[Console]::OutputEncoding=New-Object System.Text.UTF8Encoding($false);Import-Module $env:DCR_TEST_MODULE;Import-Module (Join-Path (Split-Path $env:DCR_TEST_MODULE) 'PackageCache.psm1');[pscustomobject]@{backups=@(Get-DcrRecoveryInventory $env:DCR_TEST_STATE $env:DCR_TEST_CACHE);caches=@(Get-DcrT6Caches $env:DCR_TEST_CACHE|Select-Object id,version,token)}|ConvertTo-Json -Depth 5 -Compress")
			cmd.Env = append(legacyPowerShellEnv(), "DCR_TEST_MODULE="+module, "DCR_TEST_STATE="+state, "DCR_TEST_CACHE="+cache)
			configure(cmd)
			raw, e := cmd.Output()
			if e != nil {
				t.Fatal("legacy fixture query failed", e)
			}
			var probe struct {
				Backups []app.Backup
				Caches  []app.CacheCandidate
			}
			if json.Unmarshal([]byte(strings.TrimPrefix(strings.TrimSpace(string(raw)), "\uFEFF")), &probe) != nil {
				t.Fatal("legacy fixture output invalid")
			}
			if len(probe.Backups) != len(native) || len(probe.Caches) != len(inv.caches) {
				t.Fatal("restored inventory count mismatch")
			}
			by := map[string]app.Backup{}
			for _, b := range probe.Backups {
				by[b.Name] = b
			}
			for _, b := range native {
				old := by[b.Name]
				if b.Kind != old.Kind || b.Status != old.Status || b.Version != old.Version || b.Token != old.Token || b.Bytes != old.Bytes || b.SizeKnown != old.SizeKnown {
					t.Fatalf("restore recovery parity mismatch: kind=%s status=%s", b.Kind, b.Status)
				}
			}
			for i, c := range inv.caches {
				if probe.Caches[i] != c {
					t.Fatal("restored cache token mismatch")
				}
			}
		})
	}
}

func TestNativeRestoreWindowsLocks(t *testing.T) {
	for _, kind := range []string{"package", "cache"} {
		t.Run(kind, func(t *testing.T) {
			root, state, cache, source, r := restoreFixture(t, kind)
			ops := nativeRestoreOps()
			var handle windows.Handle
			lock := func(path string, directory bool) error {
				ptr, e := windows.UTF16PtrFromString(path)
				if e != nil {
					return e
				}
				flags := uint32(windows.FILE_ATTRIBUTE_NORMAL)
				if directory {
					flags = windows.FILE_FLAG_BACKUP_SEMANTICS
				}
				handle, e = windows.CreateFile(ptr, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, flags, 0)
				return e
			}
			defer func() {
				if handle != 0 && handle != windows.InvalidHandle {
					windows.CloseHandle(handle)
				}
			}()
			guard := quarantineGuard
			if kind == "package" {
				ops.switchOps.journal = func(stage string, j packageJournal) error {
					if j.State == "switching" {
						if e := lock(filepath.Join(root, "package.json"), false); e != nil {
							return e
						}
					}
					return writePackageJournal(stage, j)
				}
			} else {
				calls := 0
				guard = func(context.Context, string) error {
					calls++
					if calls == 2 {
						return lock(source, true)
					}
					return nil
				}
			}
			if _, e := restoreNativeBackup(context.Background(), root, state, cache, r, guard, ops); e == nil {
				t.Fatal("locked fixture unexpectedly moved")
			}
			if stageVersion(root) != "0.3.5" || stageVersion(source) != "0.3.4" {
				t.Fatal("locked restore lost data")
			}
			if kind == "cache" {
				if _, e := os.Lstat(filepath.Join(cache, "_npx", "1234abcd")); !os.IsNotExist(e) {
					t.Fatal("locked cache restored")
				}
			}
		})
	}
}
