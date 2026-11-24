//go:build windows

package backend

import (
	"context"
	"crypto/sha256"
	"dcr-manager/internal/app"
	"encoding/json"
	"fmt"
	"golang.org/x/sys/windows"
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
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", "$ErrorActionPreference='Stop'; $WarningPreference='SilentlyContinue'; $ProgressPreference='SilentlyContinue'; [Console]::OutputEncoding=New-Object System.Text.UTF8Encoding($false); Import-Module $env:DCR_TEST_MODULE; $probe=Get-FileHash -LiteralPath (Join-Path $env:DCR_TEST_CACHE '_npx/1234abcd/package.json') -Algorithm SHA256 -ErrorAction Stop; if(!$probe.Hash){throw 'hash probe failed'}; $d=Get-ChildItem -LiteralPath (Join-Path $env:DCR_TEST_CACHE '_npx') -Directory | Select-Object -First 1; [pscustomobject]@{rows=@(Get-DcrT6Caches $env:DCR_TEST_CACHE | Select-Object id,version,token);path=$d.FullName.ToLowerInvariant();created=[string]$d.CreationTimeUtc.Ticks;hash=$probe.Hash} | ConvertTo-Json -Depth 4 -Compress")
	cmd.Env = append(legacyPowerShellEnv(), "DCR_TEST_MODULE="+module, "DCR_TEST_CACHE="+cache)
	configure(cmd)
	out, e := cmd.Output()
	if e != nil {
		if exit, ok := e.(*exec.ExitError); ok {
			t.Fatalf("legacy fixture query failed: %v; fixture-only stderr: %s", e, exit.Stderr)
		}
		t.Fatal("legacy fixture query failed", e)
	}
	var probe struct {
		Rows                []app.CacheCandidate
		Path, Created, Hash string
	}
	if e := json.Unmarshal([]byte(strings.TrimPrefix(strings.TrimSpace(string(out)), "\uFEFF")), &probe); e != nil {
		t.Fatal("legacy cache JSON invalid", e)
	}
	legacy := probe.Rows
	if len(legacy) != 1 {
		t.Fatal("legacy cache fixture count", len(legacy))
	}
	if legacy[0] != native.caches[0] {
		dir := filepath.Join(cache, "_npx", "1234abcd")
		info, err := plainInfo(dir)
		if err != nil {
			t.Fatal(err)
		}
		ticks, ok := creationTicks(info)
		if !ok {
			t.Fatal("creation unavailable")
		}
		raw, err := readPlainFile(filepath.Join(dir, "package.json"), maxConfigBytes)
		if err != nil {
			t.Fatal(err)
		}
		t.Fatalf("cache parity: idMatch=%t versionMatch=%t pathMatch=%t pathLengths=%d/%d shortAlias=%t/%t ticksMatch=%t hashMatch=%t", legacy[0].ID == native.caches[0].ID, legacy[0].Version == native.caches[0].Version, strings.ToLower(dir) == probe.Path, len(dir), len(probe.Path), strings.Contains(dir, "~"), strings.Contains(probe.Path, "~"), fmt.Sprint(ticks) == probe.Created, fmt.Sprintf("%X", sha256.Sum256(raw)) == probe.Hash)
	}
}

func TestLegacyTokenPathShortAlias(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Long Directory For DCR Token Test")
	if e := os.Mkdir(dir, 0700); e != nil {
		t.Fatal(e)
	}
	info, e := plainInfo(dir)
	if e != nil {
		t.Fatal(e)
	}
	canonical, ok := legacyTokenPath(dir, info)
	if !ok {
		t.Fatal("canonical path unavailable")
	}
	ptr, e := windows.UTF16PtrFromString(dir)
	if e != nil {
		t.Fatal(e)
	}
	buf := make([]uint16, 32768)
	n, e := windows.GetShortPathName(ptr, &buf[0], uint32(len(buf)))
	if e != nil || n == 0 || n >= uint32(len(buf)) {
		t.Skip("8.3 aliases unavailable on fixture volume")
	}
	alias := windows.UTF16ToString(buf[:n])
	if strings.EqualFold(alias, dir) {
		t.Skip("8.3 alias not generated on fixture volume")
	}
	aliasInfo, e := plainInfo(alias)
	if e != nil {
		t.Fatal(e)
	}
	actual, ok := legacyTokenPath(alias, aliasInfo)
	if !ok || actual != canonical {
		t.Fatal("short alias changed token identity")
	}
}
