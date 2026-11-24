//go:build windows

package backend

import (
	"context"
	"dcr-manager/internal/app"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
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
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", "$ErrorActionPreference='Stop'; Import-Module $env:DCR_TEST_MODULE; ConvertTo-Json -InputObject @(Get-DcrT6Caches $env:DCR_TEST_CACHE | Select-Object id,version,token) -Compress")
	cmd.Env = append(os.Environ(), "DCR_TEST_MODULE="+module, "DCR_TEST_CACHE="+cache)
	configure(cmd)
	out, e := cmd.Output()
	if e != nil {
		t.Fatal("legacy fixture query failed", e)
	}
	var legacy []app.CacheCandidate
	if json.Unmarshal(out, &legacy) != nil || len(legacy) != 1 || legacy[0] != native.caches[0] {
		t.Fatal("legacy and native cache token differ")
	}
}
