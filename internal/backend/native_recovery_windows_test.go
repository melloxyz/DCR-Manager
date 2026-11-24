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
)

func TestRecoveryTokenLegacyParity(t *testing.T) {
	state := t.TempDir()
	cache := t.TempDir()
	p := backupFixture(t, state, cacheBackup)
	backupFixture(t, state, pkgBackup)
	npx := filepath.Join(cache, "_npx")
	if e := os.Mkdir(npx, 0700); e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(map[string]string{"originalPath": filepath.Join(npx, "1234abcd"), "backupPath": p})
	writeFixture(t, p+".json", string(raw))
	native, known := nativeRecovery(context.Background(), state, cache)
	if !known || len(native) != 2 {
		t.Fatal("native recovery fixture not found")
	}
	module, e := filepath.Abs("../../src/RecoveryCenter.psm1")
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", "$ErrorActionPreference='Stop';$WarningPreference='SilentlyContinue';$ProgressPreference='SilentlyContinue';[Console]::OutputEncoding=New-Object System.Text.UTF8Encoding($false);Import-Module $env:DCR_TEST_MODULE;ConvertTo-Json -InputObject @(Get-DcrRecoveryInventory $env:DCR_TEST_STATE $env:DCR_TEST_CACHE) -Compress")
	cmd.Env = append(legacyTestEnv(), "DCR_TEST_MODULE="+module, "DCR_TEST_STATE="+state, "DCR_TEST_CACHE="+cache)
	configure(cmd)
	out, e := cmd.Output()
	if e != nil {
		t.Fatal("legacy fixture query failed", e)
	}
	var legacy []app.Backup
	if e = json.Unmarshal([]byte(strings.TrimPrefix(strings.TrimSpace(string(out)), "\uFEFF")), &legacy); e != nil {
		t.Fatal("legacy recovery JSON invalid", e)
	}
	if len(legacy) != 2 {
		t.Fatal("legacy recovery fixture count", len(legacy))
	}
	by := map[string]app.Backup{}
	for _, r := range legacy {
		by[r.Name] = r
	}
	for _, r := range native {
		old := by[r.Name]
		if r.Status != "ready" || old.Status != r.Status || r.Bytes != old.Bytes || r.Version != old.Version || r.Token != old.Token {
			t.Fatalf("recovery parity: kind=%s native=%s legacy=%s bytesMatch=%t versionMatch=%t tokenMatch=%t", r.Kind, r.Status, old.Status, r.Bytes == old.Bytes, r.Version == old.Version, r.Token == old.Token)
		}
	}
}
