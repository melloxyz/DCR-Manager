//go:build windows

package backend

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func quarantineGuard(context.Context, string) error { return nil }

func TestNativeCacheQuarantineFailures(t *testing.T) {
	for _, mode := range []string{"sidecar-failure", "invalid-sidecar", "occupied-backup", "move-failure", "post-version", "post-sidecar", "post-tree", "canceled-after-sidecar", "canceled-after-move"} {
		t.Run(mode, func(t *testing.T) {
			cache, state, source, token := quarantineFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ops := nativeQuarantineOps()
			moves := 0
			ops.sidecar = func(backup string, meta cacheSidecar) error {
				if mode == "sidecar-failure" {
					return errors.New("injected")
				}
				if e := writeCacheSidecar(backup, meta); e != nil {
					return e
				}
				switch mode {
				case "invalid-sidecar":
					writeFixture(t, backup+".json", "{}")
				case "occupied-backup":
					if e := os.Mkdir(backup, 0700); e != nil {
						t.Fatal(e)
					}
					writeFixture(t, filepath.Join(backup, "occupant"), "preserve")
				case "canceled-after-sidecar":
					cancel()
				}
				return nil
			}
			ops.move = func(a, b string) error {
				moves++
				if mode == "move-failure" {
					return errors.New("injected")
				}
				if e := movePackageNoReplace(a, b); e != nil {
					return e
				}
				switch mode {
				case "post-version":
					stageFixture(t, b, "0.3.5")
				case "post-sidecar":
					raw, _ := json.Marshal(cacheSidecar{OriginalPath: filepath.Join(cache, "_npx", "9988abcd"), BackupPath: b})
					writeFixture(t, b+".json", string(raw))
				case "post-tree":
					writeFixture(t, filepath.Join(b, "unexpected"), "changed")
				case "canceled-after-move":
					cancel()
				}
				return nil
			}
			backup, e := quarantineNativeCache(ctx, cache, state, "1234abcd", token, quarantineGuard, ops)
			if mode == "canceled-after-move" {
				if e != nil {
					t.Fatal("cancellation interrupted verification", e)
				}
				return
			}
			if e == nil {
				t.Fatal("injected failure accepted")
			}
			if mode == "post-version" || mode == "post-sidecar" || mode == "post-tree" {
				if _, e := os.Stat(backup); e != nil {
					t.Fatal("backup evidence lost")
				}
			} else {
				if stageVersion(source) != "0.3.4" {
					t.Fatal("failed quarantine changed source")
				}
				if mode != "move-failure" && moves != 0 {
					t.Fatal("invalid preflight attempted rename")
				}
			}
			if mode == "occupied-backup" {
				raw, _ := os.ReadFile(filepath.Join(backup, "occupant"))
				if string(raw) != "preserve" {
					t.Fatal("occupied backup changed")
				}
			}
		})
	}
}

func quarantineFixture(t *testing.T) (string, string, string, string) {
	t.Helper()
	base := t.TempDir()
	cache, state := filepath.Join(base, "cache"), filepath.Join(base, "state")
	source := filepath.Join(cache, "_npx", "1234abcd")
	stageFixture(t, source, "0.3.4")
	other := filepath.Join(cache, "_npx", "8765abcd")
	stageFixture(t, other, "0.3.5")
	sentinel := filepath.Join(other, "unrelated")
	writeFixture(t, sentinel, "preserve")
	t.Cleanup(func() {
		raw, e := os.ReadFile(sentinel)
		if e != nil || string(raw) != "preserve" {
			t.Error("unselected cache changed")
		}
	})
	inv := scanInstallations(context.Background(), "", "", cache)
	if !inv.cacheKnown || len(inv.caches) != 2 {
		t.Fatal("fixture inventory unavailable")
	}
	for _, c := range inv.caches {
		if c.ID == "1234abcd" {
			return cache, state, source, c.Token
		}
	}
	t.Fatal("fixture candidate missing")
	return "", "", "", ""
}

func TestNativeCacheQuarantine(t *testing.T) {
	cache, state, source, token := quarantineFixture(t)
	identity, e := plainInfo(source)
	if e != nil || !os.SameFile(identity, identity) {
		t.Fatal("fixture identity unavailable")
	}
	calls := 0
	backup, e := quarantineNativeCache(context.Background(), cache, state, "1234abcd", token, func(_ context.Context, p string) error {
		calls++
		if p != source {
			t.Fatal("wrong guard target")
		}
		return nil
	}, nativeQuarantineOps())
	if e != nil {
		t.Fatal(e)
	}
	if calls != 2 || filepath.Dir(backup) != state || !recoveryName.MatchString(filepath.Base(backup)) {
		t.Fatal("invalid backup or guard coverage")
	}
	if _, e = os.Lstat(source); !os.IsNotExist(e) {
		t.Fatal("source was not quarantined")
	}
	now, e := plainInfo(backup)
	if e != nil || !os.SameFile(identity, now) {
		t.Fatal("quarantine did not preserve directory identity")
	}
	raw, e := os.ReadFile(backup + ".json")
	if e != nil {
		t.Fatal(e)
	}
	var meta struct{ OriginalPath, BackupPath string }
	if json.Unmarshal(raw, &meta) != nil || meta.OriginalPath != source || meta.BackupPath != backup {
		t.Fatal("invalid restore sidecar")
	}
	rows, known := nativeRecovery(context.Background(), state, cache)
	if !known || len(rows) != 1 || rows[0].Status != "ready" || rows[0].Version != "0.3.4" || rows[0].Token == "" {
		t.Fatal("backup not restorable")
	}
}

func TestNativeCacheQuarantinePreconditions(t *testing.T) {
	for _, mode := range []string{"invalid-id", "traversal", "invalid-token", "stale-token", "relative-cache", "relative-state", "overlap", "ancestor-state", "mixed", "missing-manifest", "missing-package", "linked-source", "nested-link", "linked-state", "canceled", "early-active", "late-active", "guard-canceled", "guard-stale", "sidecar-stale", "sidecar-replaced-source"} {
		t.Run(mode, func(t *testing.T) {
			cache, state, source, token := quarantineFixture(t)
			id := "1234abcd"
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch mode {
			case "invalid-id":
				id = "arbitrary"
			case "traversal":
				id = "../1234abcd"
			case "invalid-token":
				token = ""
			case "stale-token":
				token = strings.Repeat("0", 64)
			case "relative-cache":
				cache = "cache"
			case "relative-state":
				state = "state"
			case "overlap":
				state = filepath.Join(cache, "state")
			case "ancestor-state":
				state = filepath.Dir(cache)
			case "mixed":
				writeFixture(t, filepath.Join(source, "package.json"), `{"dependencies":{"other":"1.0.0"}}`)
			case "missing-manifest":
				if e := os.Remove(filepath.Join(source, "package.json")); e != nil {
					t.Fatal(e)
				}
			case "missing-package":
				if e := os.Rename(filepath.Join(source, "node_modules"), filepath.Join(source, "previous-modules")); e != nil {
					t.Fatal(e)
				}
			case "linked-source":
				target := source + "-preserved"
				if e := os.Rename(source, target); e != nil {
					t.Fatal(e)
				}
				if e := os.Symlink(target, source); e != nil {
					t.Skip("symlink unavailable")
				}
			case "nested-link":
				if e := os.Symlink(t.TempDir(), filepath.Join(source, "node_modules", "linked")); e != nil {
					t.Skip("symlink unavailable")
				}
			case "linked-state":
				if e := os.Symlink(t.TempDir(), state); e != nil {
					t.Skip("symlink unavailable")
				}
			case "canceled":
				cancel()
			}
			ops := nativeQuarantineOps()
			moves := 0
			calls := 0
			ops.move = func(a, b string) error { moves++; return movePackageNoReplace(a, b) }
			guard := func(context.Context, string) error {
				calls++
				if mode == "early-active" || mode == "late-active" && calls == 2 {
					return maintenanceError("ACTIVE")
				}
				if mode == "guard-canceled" {
					cancel()
				}
				if mode == "guard-stale" {
					stageFixture(t, source, "0.3.5")
				}
				return nil
			}
			ops.sidecar = func(backup string, meta cacheSidecar) error {
				if e := writeCacheSidecar(backup, meta); e != nil {
					return e
				}
				if mode == "sidecar-stale" {
					stageFixture(t, source, "0.3.5")
				}
				if mode == "sidecar-replaced-source" {
					if e := os.Rename(source, source+"-preserved"); e != nil {
						t.Fatal(e)
					}
					stageFixture(t, source, "0.3.4")
				}
				return nil
			}
			if _, e := quarantineNativeCache(ctx, cache, state, id, token, guard, ops); e == nil || moves != 0 {
				t.Fatal("unsafe precondition accepted", e, moves)
			}
		})
	}
}

func TestNativeCacheSidecarNoOverwrite(t *testing.T) {
	for _, mode := range []string{"file", "directory", "backup"} {
		t.Run(mode, func(t *testing.T) {
			state := t.TempDir()
			backup := filepath.Join(state, "cache-backup-"+strings.Repeat("a", 32))
			path := backup + ".json"
			if mode == "file" {
				writeFixture(t, path, "evidence")
			} else {
				if mode == "backup" {
					path = backup
				}
				if e := os.Mkdir(path, 0700); e != nil {
					t.Fatal(e)
				}
				writeFixture(t, filepath.Join(path, "evidence"), "preserve")
			}
			if e := writeCacheSidecar(backup, cacheSidecar{OriginalPath: filepath.Join(state, "fixture"), BackupPath: backup}); e == nil {
				t.Fatal("occupied evidence accepted")
			}
			if mode == "file" {
				raw, _ := os.ReadFile(path)
				if string(raw) != "evidence" {
					t.Fatal("sidecar overwritten")
				}
			} else {
				raw, _ := os.ReadFile(filepath.Join(path, "evidence"))
				if string(raw) != "preserve" {
					t.Fatal("evidence overwritten")
				}
			}
		})
	}
}

func TestNativeCacheQuarantineAllowsIncompleteEntry(t *testing.T) {
	cache, state, source, token := quarantineFixture(t)
	if e := os.Remove(filepath.Join(packageDir(filepath.Join(source, "node_modules")), "dist", "index.js")); e != nil {
		t.Fatal(e)
	}
	if _, e := quarantineNativeCache(context.Background(), cache, state, "1234abcd", token, quarantineGuard, nativeQuarantineOps()); e != nil {
		t.Fatal("legacy-valid candidate refused", e)
	}
	rows, known := nativeRecovery(context.Background(), state, cache)
	if !known || len(rows) != 1 || rows[0].Status != "incomplete" || rows[0].Token != "" {
		t.Fatal("incomplete cache became restorable")
	}
}

func TestNativeCacheQuarantineWithoutLockfile(t *testing.T) {
	cache, state, source, token := quarantineFixture(t)
	if e := os.Remove(filepath.Join(source, "package-lock.json")); e != nil {
		t.Fatal(e)
	}
	if _, e := quarantineNativeCache(context.Background(), cache, state, "1234abcd", token, quarantineGuard, nativeQuarantineOps()); e != nil {
		t.Fatal("cache without optional lockfile refused", e)
	}
	rows, known := nativeRecovery(context.Background(), state, cache)
	if !known || len(rows) != 1 || rows[0].Status != "ready" {
		t.Fatal("cache without lockfile not restorable")
	}
}

func TestNativeCacheQuarantineMoveErrorAfterRename(t *testing.T) {
	cache, state, source, token := quarantineFixture(t)
	ops := nativeQuarantineOps()
	ops.move = func(a, b string) error {
		if e := movePackageNoReplace(a, b); e != nil {
			return e
		}
		return errors.New("injected error after rename")
	}
	backup, e := quarantineNativeCache(context.Background(), cache, state, "1234abcd", token, quarantineGuard, ops)
	if e == nil {
		t.Fatal("move error hidden")
	}
	if _, e := os.Lstat(source); !os.IsNotExist(e) {
		t.Fatal("source recreated")
	}
	if stageVersion(backup) != "0.3.4" {
		t.Fatal("backup content lost")
	}
	rows, known := nativeRecovery(context.Background(), state, cache)
	if !known || len(rows) != 1 || rows[0].Status != "ready" {
		t.Fatal("uncertain move lost recovery evidence")
	}
}
