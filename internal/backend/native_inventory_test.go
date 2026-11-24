package backend

import (
	"context"
	"dcr-manager/internal/app"
	"os"
	"path/filepath"
	"testing"
)

func packageFixture(t *testing.T, root, version string, exclusive bool) {
	t.Helper()
	pkg := filepath.Join(root, "node_modules", "@wonderwhy-er", "desktop-commander")
	if e := os.MkdirAll(filepath.Join(pkg, "dist"), 0700); e != nil {
		t.Fatal(e)
	}
	deps := "\"@wonderwhy-er/desktop-commander\":\"" + version + "\""
	if !exclusive {
		deps += ",\"other\":\"1.0.0\""
	}
	for path, data := range map[string]string{
		filepath.Join(root, "package.json"):    "{\"dependencies\":{" + deps + "}}",
		filepath.Join(pkg, "package.json"):     "{\"name\":\"@wonderwhy-er/desktop-commander\",\"version\":\"" + version + "\",\"engines\":{\"node\":\">=22.12.0\"}}",
		filepath.Join(pkg, "dist", "index.js"): "// fixture",
	} {
		if e := os.WriteFile(path, []byte(data), 0600); e != nil {
			t.Fatal(e)
		}
	}
}
func TestNativeInstallCacheInventory(t *testing.T) {
	local := t.TempDir()
	cache := t.TempDir()
	packageFixture(t, filepath.Join(local, "DesktopCommander"), "0.3.4", true)
	// Global location is already a node_modules directory.
	globalRoot := filepath.Join(local, "global")
	packageFixture(t, globalRoot, "0.3.3", true)
	global := filepath.Join(globalRoot, "node_modules")
	a := filepath.Join(cache, "_npx", "1234abcd")
	packageFixture(t, a, "0.3.3", true)
	packageFixture(t, filepath.Join(cache, "_npx", "8765abcd"), "0.3.3", false)
	inv := scanInstallations(context.Background(), local, global, cache)
	if !inv.cacheKnown || inv.version != "0.3.4" || !inv.installed || len(inv.installations) != 3 || len(inv.caches) != 1 || inv.caches[0].ID != "1234abcd" {
		t.Fatal(inv)
	}
	if !cacheTokenPattern.MatchString(inv.caches[0].Token) {
		t.Fatal("invalid cache token")
	}
	old := inv.caches[0].Token
	if e := os.WriteFile(filepath.Join(a, "package.json"), []byte("{\"dependencies\":{\"@wonderwhy-er/desktop-commander\":\"latest\"}}"), 0600); e != nil {
		t.Fatal(e)
	}
	inv = scanInstallations(context.Background(), local, global, cache)
	if len(inv.caches) != 1 || inv.caches[0].Token == old {
		t.Fatal("stale token survived")
	}
}
func TestNativeInventoryUnknownAndEmpty(t *testing.T) {
	local := t.TempDir()
	cache := t.TempDir()
	inv := scanInstallations(context.Background(), local, "", cache)
	if inv.version != "absent" || inv.installed || !inv.cacheKnown || len(inv.caches) != 0 {
		t.Fatal(inv)
	}
	inv = scanInstallations(context.Background(), "", "", "")
	if inv.cacheKnown || inv.version != "unknown" {
		t.Fatal(inv)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if inv = scanInstallations(ctx, local, "", cache); inv.cacheKnown {
		t.Fatal("cancelled scan reported known")
	}
}
func TestCacheParityFailsClosed(t *testing.T) {
	a := app.CacheCandidate{ID: "1234abcd", Version: "0.3.4", Token: "a"}
	s := app.Snapshot{Status: app.Status{CacheKnown: true, CacheCandidates: 1}, Caches: []app.CacheCandidate{a}}
	inv := installInventory{cacheKnown: true, caches: []app.CacheCandidate{{ID: a.ID, Version: a.Version, Token: "b"}}}
	applyCacheParity(&s, inv, true)
	if s.Status.CacheKnown || len(s.Caches) != 0 {
		t.Fatal("mismatch authorized")
	}
	s.Status.CacheKnown = true
	s.Status.CacheCandidates = 1
	s.Caches = []app.CacheCandidate{a}
	inv.caches = []app.CacheCandidate{a}
	applyCacheParity(&s, inv, true)
	if !s.Status.CacheKnown || len(s.Caches) != 1 {
		t.Fatal("matching cache refused")
	}
}
