//go:build windows

package backend

import (
	"context"
	"dcr-manager/internal/app"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func restoreFixture(t *testing.T, kind string) (string, string, string, string, app.MaintenanceRequest) {
	t.Helper()
	base := t.TempDir()
	root, state, cache := filepath.Join(base, "install"), filepath.Join(base, "state"), filepath.Join(base, "cache")
	stageFixture(t, root, "0.3.5")
	if e := os.MkdirAll(filepath.Join(cache, "_npx"), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.Mkdir(state, 0700); e != nil {
		t.Fatal(e)
	}
	name := pkgBackup
	if kind == "cache" {
		name = cacheBackup
	}
	source := backupFixture(t, state, name)
	if kind == "cache" {
		raw, e := json.Marshal(cacheSidecar{OriginalPath: filepath.Join(cache, "_npx", "1234abcd"), BackupPath: source})
		if e != nil {
			t.Fatal(e)
		}
		writeFixture(t, source+".json", string(raw))
	}
	if e := os.Mkdir(filepath.Join(root, "Remote"), 0700); e != nil {
		t.Fatal(e)
	}
	sentinel := filepath.Join(root, "Remote", "credentials")
	writeFixture(t, sentinel, "fixture-secret")
	t.Cleanup(func() {
		raw, e := os.ReadFile(sentinel)
		if e != nil || string(raw) != "fixture-secret" {
			t.Error("credential fixture changed")
		}
	})
	row := inspectRecovery(context.Background(), state, cache, name, time.Now())
	if row.Status != "ready" {
		t.Fatal("restore fixture not ready")
	}
	return root, state, cache, source, app.MaintenanceRequest{Action: app.RestoreBackup, CacheID: name, Expected: row.Token, CurrentVersion: "0.3.5"}
}

func TestNativePackageRestore(t *testing.T) {
	root, state, cache, source, request := restoreFixture(t, "package")
	writeFixture(t, filepath.Join(source, "unrelated-shim.cmd"), "preserve")
	request.Expected = inspectRecovery(context.Background(), state, cache, request.CacheID, time.Now()).Token
	raw, e := os.ReadFile(filepath.Join(source, "package-lock.json"))
	if e != nil {
		t.Fatal(e)
	}
	previous, e := restoreNativeBackup(context.Background(), root, state, cache, request, quarantineGuard, nativeRestoreOps())
	if e != nil {
		t.Fatal(e)
	}
	if stageVersion(root) != "0.3.4" || stageVersion(previous) != "0.3.5" || stageVersion(source) != "0.3.4" {
		t.Fatal("restore versions incorrect")
	}
	got, e := os.ReadFile(filepath.Join(root, "package-lock.json"))
	if e != nil || string(got) != string(raw) {
		t.Fatal("legacy lockfile not preserved")
	}
	row := inspectRecovery(context.Background(), state, cache, request.CacheID, time.Now())
	if row.Status != "ready" || row.Token != request.Expected {
		t.Fatal("source package backup changed")
	}
	if _, e := os.Lstat(filepath.Join(root, "unrelated-shim.cmd")); !os.IsNotExist(e) {
		t.Fatal("untracked backup file activated")
	}
	if raw, e := os.ReadFile(filepath.Join(source, "unrelated-shim.cmd")); e != nil || string(raw) != "preserve" {
		t.Fatal("untracked source file changed")
	}
}

func TestNativeCacheRestore(t *testing.T) {
	root, state, cache, source, r := restoreFixture(t, "cache")
	identity, e := plainInfo(source)
	if e != nil || !os.SameFile(identity, identity) {
		t.Fatal("fixture identity unavailable")
	}
	metadata, e := os.ReadFile(source + ".json")
	if e != nil {
		t.Fatal(e)
	}
	var guarded []string
	target, e := restoreNativeBackup(context.Background(), root, state, cache, r, func(_ context.Context, p string) error { guarded = append(guarded, p); return nil }, nativeRestoreOps())
	if e != nil {
		t.Fatal(e)
	}
	if target != filepath.Join(cache, "_npx", "1234abcd") || len(guarded) != 2 || guarded[0] != source || guarded[1] != target {
		t.Fatal("restore target or guards incorrect")
	}
	now, e := plainInfo(target)
	if e != nil || !os.SameFile(identity, now) || stageVersion(target) != "0.3.4" {
		t.Fatal("cache directory not preserved")
	}
	raw, e := os.ReadFile(source + ".json")
	if e != nil || string(raw) != string(metadata) {
		t.Fatal("sidecar lost")
	}
	if _, e = os.Lstat(source); !os.IsNotExist(e) {
		t.Fatal("backup not moved")
	}
	inv := scanInstallations(context.Background(), "", "", cache)
	if !inv.cacheKnown || len(inv.caches) != 1 {
		t.Fatal("restored cache not discovered")
	}
}

func TestNativeRestorePreconditions(t *testing.T) {
	for _, kind := range []string{"package", "cache"} {
		for _, mode := range []string{"stale-token", "invalid-name", "stage-name", "wrong-action", "relative-state", "overlap", "mixed", "missing-entry", "nested-link", "linked-state", "early-active", "late-active", "canceled", "late-canceled", "guard-stale", "occupied-target", "sidecar-change", "wrong-current"} {
			if kind == "package" && (mode == "occupied-target" || mode == "sidecar-change") || kind == "cache" && mode == "wrong-current" {
				continue
			}
			t.Run(kind+"/"+mode, func(t *testing.T) {
				root, state, cache, source, r := restoreFixture(t, kind)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				switch mode {
				case "stale-token":
					r.Expected = strings.Repeat("0", 64)
				case "invalid-name":
					r.CacheID = "../" + r.CacheID
				case "stage-name":
					r.CacheID = txnStage
				case "wrong-action":
					r.Action = app.QuarantineCache
				case "relative-state":
					state = "state"
				case "overlap":
					if kind == "package" {
						root = state
					} else {
						cache = state
					}
				case "mixed":
					writeFixture(t, filepath.Join(source, "package.json"), `{"dependencies":{"other":"1.0.0"}}`)
				case "missing-entry":
					if e := os.Remove(filepath.Join(packageDir(filepath.Join(source, "node_modules")), "dist", "index.js")); e != nil {
						t.Fatal(e)
					}
				case "nested-link":
					if e := os.Symlink(t.TempDir(), filepath.Join(source, "node_modules", "linked")); e != nil {
						t.Skip("symlink unavailable")
					}
				case "linked-state":
					old := state
					state = filepath.Join(filepath.Dir(state), "linked-state")
					if e := os.Symlink(old, state); e != nil {
						t.Skip("symlink unavailable")
					}
				case "canceled":
					cancel()
				case "occupied-target":
					if e := os.Mkdir(filepath.Join(cache, "_npx", "1234abcd"), 0700); e != nil {
						t.Fatal(e)
					}
				case "sidecar-change":
					writeFixture(t, source+".json", "{}")
				case "wrong-current":
					r.CurrentVersion = "0.3.3"
				}
				ops := nativeRestoreOps()
				moves := 0
				ops.move = func(a, b string) error { moves++; return movePackageNoReplace(a, b) }
				ops.switchOps.move = ops.move
				calls := 0
				guard := func(context.Context, string) error {
					calls++
					if mode == "early-active" || mode == "late-active" && calls == 2 {
						return maintenanceError("ACTIVE")
					}
					if mode == "late-canceled" && calls == 2 {
						cancel()
					}
					if mode == "guard-stale" {
						writeFixture(t, filepath.Join(source, "package-lock.json"), "changed")
					}
					return nil
				}
				if _, e := restoreNativeBackup(ctx, root, state, cache, r, guard, ops); e == nil || moves != 0 {
					t.Fatal("unsafe restore accepted", e, moves)
				}
			})
		}
	}
}

func TestNativePackageRestoreCopyFailures(t *testing.T) {
	for _, mode := range []string{"copy-error", "partial-copy", "corrupt-entry", "missing-file", "stage-replaced", "source-changed", "current-changed"} {
		t.Run(mode, func(t *testing.T) {
			root, state, cache, source, r := restoreFixture(t, "package")
			ops := nativeRestoreOps()
			moves := 0
			ops.switchOps.move = func(a, b string) error { moves++; return movePackageNoReplace(a, b) }
			ops.copy = func(ctx context.Context, a, b string) error {
				if mode == "copy-error" {
					return errors.New("injected")
				}
				if mode == "partial-copy" {
					writeFixture(t, filepath.Join(b, "package.json"), "partial")
					return errors.New("injected")
				}
				if e := copyRestorePackage(ctx, a, b); e != nil {
					return e
				}
				switch mode {
				case "corrupt-entry":
					writeFixture(t, filepath.Join(packageDir(filepath.Join(b, "node_modules")), "dist", "index.js"), "corrupted")
				case "missing-file":
					if e := os.Remove(filepath.Join(b, "package-lock.json")); e != nil {
						t.Fatal(e)
					}
				case "stage-replaced":
					if e := os.Rename(b, b+"-preserved"); e != nil {
						t.Fatal(e)
					}
					if e := os.Mkdir(b, 0700); e != nil {
						t.Fatal(e)
					}
				case "source-changed":
					writeFixture(t, filepath.Join(source, "package-lock.json"), "changed")
				case "current-changed":
					stageFixture(t, root, "0.3.6")
				}
				return nil
			}
			if _, e := restoreNativeBackup(context.Background(), root, state, cache, r, quarantineGuard, ops); e == nil || moves != 0 {
				t.Fatal("invalid copy activated", e, moves)
			}
			if mode != "current-changed" && stageVersion(root) != "0.3.5" {
				t.Fatal("installation changed on copy failure")
			}
			if stageVersion(source) != "0.3.4" {
				t.Fatal("source backup lost")
			}
		})
	}
}

func TestNativePackageRestoreMoveFailures(t *testing.T) {
	for boundary := 1; boundary <= 6; boundary++ {
		t.Run(fmt.Sprint(boundary), func(t *testing.T) {
			root, state, cache, source, r := restoreFixture(t, "package")
			ops := nativeRestoreOps()
			calls := 0
			ops.switchOps.move = func(a, b string) error {
				calls++
				if calls == boundary {
					return errors.New("injected")
				}
				return movePackageNoReplace(a, b)
			}
			if _, e := restoreNativeBackup(context.Background(), root, state, cache, r, quarantineGuard, ops); e == nil {
				t.Fatal("move error hidden")
			}
			if stageVersion(root) != "0.3.5" || stageVersion(source) != "0.3.4" {
				t.Fatal("rollback lost package")
			}
			stages, e := filepath.Glob(filepath.Join(state, "restore-stage-*"))
			if e != nil || len(stages) != 1 || switchStatus(t, stages[0]) != "switch-failed" {
				t.Fatal("rollback journal incorrect")
			}
		})
	}
}

func TestNativePackageRestoreRollbackFailures(t *testing.T) {
	for boundary := 7; boundary <= 12; boundary++ {
		t.Run(fmt.Sprint(boundary), func(t *testing.T) {
			root, state, cache, _, r := restoreFixture(t, "package")
			ops := nativeRestoreOps()
			calls := 0
			ops.switchOps.verify = func(string) (string, error) { return "", errors.New("injected") }
			ops.switchOps.move = func(a, b string) error {
				calls++
				if calls == boundary {
					return errors.New("injected")
				}
				return movePackageNoReplace(a, b)
			}
			previous, e := restoreNativeBackup(context.Background(), root, state, cache, r, quarantineGuard, ops)
			if e == nil || calls < boundary {
				t.Fatal("rollback error hidden")
			}
			if _, e := os.Stat(previous); e != nil {
				t.Fatal("previous package evidence lost")
			}
			stages, _ := filepath.Glob(filepath.Join(state, "restore-stage-*"))
			if len(stages) != 1 || switchStatus(t, stages[0]) != "rollback-incomplete" {
				t.Fatal("incomplete rollback not journaled")
			}
			row := inspectRecovery(context.Background(), state, cache, r.CacheID, time.Now())
			if row.Status != "ready" || row.Token != r.Expected {
				t.Fatal("original source backup changed")
			}
		})
	}
}

func TestNativeRestoreCancellationAfterMove(t *testing.T) {
	for _, kind := range []string{"package", "cache"} {
		t.Run(kind, func(t *testing.T) {
			root, state, cache, _, r := restoreFixture(t, kind)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ops := nativeRestoreOps()
			move := func(a, b string) error { cancel(); return movePackageNoReplace(a, b) }
			ops.move = move
			ops.switchOps.move = move
			target, e := restoreNativeBackup(ctx, root, state, cache, r, quarantineGuard, ops)
			if e != nil {
				t.Fatal("cancellation interrupted transaction", e)
			}
			if kind == "package" {
				if stageVersion(root) != "0.3.4" || stageVersion(target) != "0.3.5" {
					t.Fatal("package restore incomplete")
				}
			} else if stageVersion(target) != "0.3.4" {
				t.Fatal("cache restore incomplete")
			}
		})
	}
}

func TestNativeCacheRestoreFailures(t *testing.T) {
	for _, mode := range []string{"move-error", "after-move-error", "occupied-late", "metadata-late", "metadata-after-move", "post-mismatch", "backup-replaced"} {
		t.Run(mode, func(t *testing.T) {
			root, state, cache, source, r := restoreFixture(t, "cache")
			ops := nativeRestoreOps()
			calls := 0
			moves := 0
			guard := func(context.Context, string) error {
				calls++
				if calls != 2 {
					return nil
				}
				switch mode {
				case "occupied-late":
					if e := os.Mkdir(filepath.Join(cache, "_npx", "1234abcd"), 0700); e != nil {
						t.Fatal(e)
					}
				case "metadata-late":
					writeFixture(t, source+".json", "changed")
				case "backup-replaced":
					if e := os.Rename(source, source+"-preserved"); e != nil {
						t.Fatal(e)
					}
					backupFixture(t, state, r.CacheID)
				}
				return nil
			}
			ops.move = func(a, b string) error {
				moves++
				if mode == "move-error" {
					return errors.New("injected")
				}
				if e := movePackageNoReplace(a, b); e != nil {
					return e
				}
				switch mode {
				case "after-move-error":
					return errors.New("injected")
				case "metadata-after-move":
					writeFixture(t, source+".json", "changed")
				case "post-mismatch":
					writeFixture(t, filepath.Join(b, "package-lock.json"), "changed")
				}
				return nil
			}
			target, e := restoreNativeBackup(context.Background(), root, state, cache, r, guard, ops)
			if e == nil {
				t.Fatal("fault hidden")
			}
			if mode == "occupied-late" || mode == "metadata-late" || mode == "backup-replaced" {
				if moves != 0 {
					t.Fatal("stale cache moved")
				}
			}
			if mode == "after-move-error" || mode == "metadata-after-move" || mode == "post-mismatch" {
				if stageVersion(target) != "0.3.4" {
					t.Fatal("moved evidence lost")
				}
			} else if stageVersion(source) != "0.3.4" {
				t.Fatal("backup lost on failure")
			}
		})
	}
}

func TestNativePackageRestoreIntoAbsentInstall(t *testing.T) {
	root, state, cache, _, r := restoreFixture(t, "package")
	root = filepath.Join(filepath.Dir(root), "fresh-install")
	r.CurrentVersion = "absent"
	previous, e := restoreNativeBackup(context.Background(), root, state, cache, r, quarantineGuard, nativeRestoreOps())
	if e != nil || stageVersion(root) != "0.3.4" {
		t.Fatal("fresh restore failed", e)
	}
	entries, e := os.ReadDir(previous)
	if e != nil || len(entries) != 0 {
		t.Fatal("fresh previous backup should be empty")
	}
}

func TestNativePackageRestoreUnknownCurrent(t *testing.T) {
	root, state, cache, _, r := restoreFixture(t, "package")
	writeFixture(t, filepath.Join(packageDir(filepath.Join(root, "node_modules")), "package.json"), "{}")
	r.CurrentVersion = "unknown"
	previous, e := restoreNativeBackup(context.Background(), root, state, cache, r, quarantineGuard, nativeRestoreOps())
	if e != nil || stageVersion(root) != "0.3.4" {
		t.Fatal("unknown-current restore failed", e)
	}
	raw, e := os.ReadFile(filepath.Join(packageDir(filepath.Join(previous, "node_modules")), "package.json"))
	if e != nil || string(raw) != "{}" {
		t.Fatal("broken previous installation not preserved")
	}
}

func TestNativePackageRestoreJournalFailures(t *testing.T) {
	for _, mode := range []string{"switching", "completed"} {
		t.Run(mode, func(t *testing.T) {
			root, state, cache, _, r := restoreFixture(t, "package")
			ops := nativeRestoreOps()
			ops.switchOps.journal = func(stage string, j packageJournal) error {
				if j.State == mode {
					return errors.New("injected")
				}
				return writePackageJournal(stage, j)
			}
			if _, e := restoreNativeBackup(context.Background(), root, state, cache, r, quarantineGuard, ops); e == nil {
				t.Fatal("journal failure hidden")
			}
			if stageVersion(root) != "0.3.5" {
				t.Fatal("journal failure replaced installation")
			}
			row := inspectRecovery(context.Background(), state, cache, r.CacheID, time.Now())
			if row.Status != "ready" || row.Token != r.Expected {
				t.Fatal("journal failure changed source backup")
			}
		})
	}
}
