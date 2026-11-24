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

func TestNativeCacheQuarantineRecoveryParity(t *testing.T) {
	for _, mode := range []string{"ready", "incomplete", "destination-blocked"} {
		t.Run(mode, func(t *testing.T) {
			cache, state, source, token := quarantineFixture(t)
			ops := nativeQuarantineOps()
			if mode == "incomplete" {
				if e := os.Remove(filepath.Join(packageDir(filepath.Join(source, "node_modules")), "dist", "index.js")); e != nil {
					t.Fatal(e)
				}
			}
			if mode == "destination-blocked" {
				ops.move = func(a, b string) error {
					if e := movePackageNoReplace(a, b); e != nil {
						return e
					}
					return os.Mkdir(a, 0700)
				}
			}
			backup, e := quarantineNativeCache(context.Background(), cache, state, "1234abcd", token, quarantineGuard, ops)
			if (e == nil) != (mode != "destination-blocked") {
				t.Fatal("unexpected transaction result", e)
			}
			native, known := nativeRecovery(context.Background(), state, cache)
			if !known || len(native) != 1 || native[0].Status != mode {
				t.Fatal("unexpected native recovery status")
			}
			module, e := filepath.Abs("../../src/RecoveryCenter.psm1")
			if e != nil {
				t.Fatal(e)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", "$ErrorActionPreference='Stop';$WarningPreference='SilentlyContinue';$ProgressPreference='SilentlyContinue';[Console]::OutputEncoding=New-Object System.Text.UTF8Encoding($false);Import-Module $env:DCR_TEST_MODULE;Import-Module (Join-Path (Split-Path $env:DCR_TEST_MODULE) 'Recovery.psm1');$rows=@(Get-DcrRecoveryInventory $env:DCR_TEST_STATE $env:DCR_TEST_CACHE);$target='';if($rows.Count -eq 1 -and $rows[0].status -eq 'ready'){$target=Get-DcrCacheRestoreTarget $env:DCR_TEST_STATE $env:DCR_TEST_CACHE $env:DCR_TEST_BACKUP};[pscustomobject]@{rows=$rows;target=$target}|ConvertTo-Json -Depth 4 -Compress")
			cmd.Env = append(legacyPowerShellEnv(), "DCR_TEST_MODULE="+module, "DCR_TEST_STATE="+state, "DCR_TEST_CACHE="+cache, "DCR_TEST_BACKUP="+filepath.Base(backup))
			configure(cmd)
			out, e := cmd.Output()
			if e != nil {
				t.Fatal("legacy fixture query failed", e)
			}
			var probe struct {
				Rows   []app.Backup
				Target string
			}
			if e = json.Unmarshal([]byte(strings.TrimPrefix(strings.TrimSpace(string(out)), "\uFEFF")), &probe); e != nil || len(probe.Rows) != 1 {
				t.Fatal("legacy fixture output invalid")
			}
			old, r := probe.Rows[0], native[0]
			if old.Name != r.Name || old.Status != r.Status || old.Version != r.Version || old.Token != r.Token || old.Bytes != r.Bytes || old.SizeKnown != r.SizeKnown {
				t.Fatalf("quarantine recovery parity mismatch: status=%s", mode)
			}
			if mode == "ready" && probe.Target != source {
				t.Fatal("legacy restore target mismatch")
			}
		})
	}
}

func TestNativeCacheQuarantineLockedDirectory(t *testing.T) {
	cache, state, source, token := quarantineFixture(t)
	ops := nativeQuarantineOps()
	var handle windows.Handle
	ops.sidecar = func(backup string, meta cacheSidecar) error {
		if e := writeCacheSidecar(backup, meta); e != nil {
			return e
		}
		path, e := windows.UTF16PtrFromString(source)
		if e != nil {
			return e
		}
		handle, e = windows.CreateFile(path, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
		return e
	}
	defer func() {
		if handle != 0 && handle != windows.InvalidHandle {
			windows.CloseHandle(handle)
		}
	}()
	backup, e := quarantineNativeCache(context.Background(), cache, state, "1234abcd", token, quarantineGuard, ops)
	if e == nil || stageVersion(source) != "0.3.4" {
		t.Fatal("locked source was not preserved", e)
	}
	if _, e := os.Lstat(backup); !os.IsNotExist(e) {
		t.Fatal("locked source moved")
	}
	if _, e := os.Stat(backup + ".json"); e != nil {
		t.Fatal("sidecar evidence lost")
	}
}
