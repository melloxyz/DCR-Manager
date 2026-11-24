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

func TestCacheTokenLegacyParity(t *testing.T) {
	cache := t.TempDir()
	packageFixture(t, filepath.Join(cache, "_npx", "1234abcd"), "0.3.4", true)
	native := scanInstallations(context.Background(), t.TempDir(), "", cache)
	if !native.cacheKnown || len(native.caches) != 1 {
		t.Fatal("native fixture not found")
	}
	module, e := filepath.Abs("../../src/PackageCache.psm1")
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", "$ErrorActionPreference='Stop'; $WarningPreference='SilentlyContinue'; $ProgressPreference='SilentlyContinue'; [Console]::OutputEncoding=New-Object System.Text.UTF8Encoding($false); Import-Module $env:DCR_TEST_MODULE; ConvertTo-Json -InputObject @(Get-DcrT6Caches $env:DCR_TEST_CACHE | Select-Object id,version,token) -Compress")
	cmd.Env = append(os.Environ(), "DCR_TEST_MODULE="+module, "DCR_TEST_CACHE="+cache)
	configure(cmd)
	out, e := cmd.Output()
	if e != nil {
		t.Fatal("legacy fixture query failed", e)
	}
	var legacy []app.CacheCandidate
	if e := json.Unmarshal([]byte(strings.TrimPrefix(strings.TrimSpace(string(out)), "\uFEFF")), &legacy); e != nil {
		t.Fatal("legacy cache JSON invalid", e)
	}
	if len(legacy) != 1 {
		t.Fatal("legacy cache fixture count", len(legacy))
	}
	if legacy[0] != native.caches[0] {
		t.Fatalf("cache parity: idMatch=%t versionMatch=%t tokenMatch=%t", legacy[0].ID == native.caches[0].ID, legacy[0].Version == native.caches[0].Version, legacy[0].Token == native.caches[0].Token)
	}
}
