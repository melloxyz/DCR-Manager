//go:build windows

package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func switchStatus(t *testing.T, stage string) string {
	t.Helper()
	raw, e := os.ReadFile(filepath.Join(stage, "transaction.json"))
	if e != nil {
		t.Fatal(e)
	}
	var j packageJournal
	if e = json.Unmarshal(raw, &j); e != nil {
		t.Fatal(e)
	}
	return j.State
}

func TestNativePackageSwitchMoveFailures(t *testing.T) {
	for boundary := 1; boundary <= 6; boundary++ {
		t.Run(fmt.Sprint(boundary), func(t *testing.T) {
			root, state, p := switchFixture(t)
			before, _ := packageTree(root)
			prepared, _ := packageTree(p.Stage)
			ops := nativeSwitchOps()
			calls := 0
			ops.move = func(a, b string) error {
				calls++
				if calls == boundary {
					return errors.New("injected")
				}
				return movePackageNoReplace(a, b)
			}
			_, e := switchNativePackage(context.Background(), root, state, p, "0.3.4", func(context.Context, string) error { return nil }, ops)
			if e == nil || !samePackageTree(root, before) || !samePackageTree(p.Stage, prepared) || switchStatus(t, p.Stage) != "switch-failed" {
				t.Fatal("rollback did not restore fixture", e)
			}
		})
	}
}

func TestNativePackageSwitchRollbackAndJournalFailures(t *testing.T) {
	for _, mode := range []string{"rollback", "verify", "initial-journal", "final-journal", "rollback-journal", "occupied-target", "replaced-target"} {
		t.Run(mode, func(t *testing.T) {
			root, state, p := switchFixture(t)
			before, _ := packageTree(root)
			prepared, _ := packageTree(p.Stage)
			ops := nativeSwitchOps()
			calls := 0
			ops.move = func(a, b string) error {
				calls++
				if mode == "rollback" && calls >= 3 {
					return errors.New("injected")
				}
				if mode == "occupied-target" && calls == 1 {
					if e := movePackageNoReplace(a, b); e != nil {
						return e
					}
					if e := os.Mkdir(a, 0700); e != nil {
						t.Fatal(e)
					}
					if e := os.WriteFile(filepath.Join(a, "occupant"), []byte("preserve"), 0600); e != nil {
						t.Fatal(e)
					}
					return nil
				}
				return movePackageNoReplace(a, b)
			}
			if mode == "verify" || mode == "rollback-journal" {
				ops.verify = func(string) (string, error) { return "", errors.New("injected") }
			}
			if mode == "replaced-target" {
				ops.verify = func(string) (string, error) {
					path := filepath.Join(root, "package.json")
					if e := os.Rename(path, path+".original"); e != nil {
						t.Fatal(e)
					}
					if e := os.WriteFile(path, []byte("unrelated"), 0600); e != nil {
						t.Fatal(e)
					}
					return "", errors.New("injected")
				}
			}
			ops.journal = func(s string, j packageJournal) error {
				if mode == "initial-journal" || mode == "final-journal" && j.State == "completed" || mode == "rollback-journal" && j.State == "switch-failed" {
					return errors.New("injected")
				}
				return writePackageJournal(s, j)
			}
			_, e := switchNativePackage(context.Background(), root, state, p, "0.3.4", func(context.Context, string) error { return nil }, ops)
			if e == nil {
				t.Fatal("injected failure accepted")
			}
			if mode == "rollback" || mode == "replaced-target" || mode == "occupied-target" {
				if switchStatus(t, p.Stage) != "rollback-incomplete" {
					t.Fatal("missing incomplete state")
				}
				if mode == "replaced-target" {
					raw, _ := os.ReadFile(filepath.Join(root, "package.json"))
					if string(raw) != "unrelated" {
						t.Fatal("unrelated target moved")
					}
				}
				if mode == "occupied-target" {
					raw, _ := os.ReadFile(filepath.Join(root, "node_modules", "occupant"))
					if string(raw) != "preserve" {
						t.Fatal("occupant changed")
					}
				}
			} else if !samePackageTree(root, before) || !samePackageTree(p.Stage, prepared) {
				t.Fatal("failed transaction changed packages")
			}
		})
	}
}

func TestNativePackageSwitchPreconditions(t *testing.T) {
	for _, mode := range []string{"stale", "version", "identity", "journal", "linked-artifact", "linked-stage-extra", "canceled", "guard-canceled", "guard-stale", "nested-state", "relative"} {
		t.Run(mode, func(t *testing.T) {
			root, state, p := switchFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			expected := "0.3.4"
			switch mode {
			case "stale":
				expected = "0.3.3"
			case "version":
				p.Version = "0.3.6"
			case "identity":
				p.identity = nil
			case "journal":
				if e := os.WriteFile(filepath.Join(p.Stage, "transaction.json"), []byte("existing"), 0600); e != nil {
					t.Fatal(e)
				}
			case "linked-artifact", "linked-stage-extra":
				dir := p.Stage
				if mode == "linked-artifact" {
					dir = filepath.Join(p.Stage, "node_modules")
				}
				if e := os.Symlink(t.TempDir(), filepath.Join(dir, "linked")); e != nil {
					t.Skip("symlink unavailable")
				}
			case "canceled":
				cancel()
			case "nested-state":
				state = root
			case "relative":
				root = "relative"
			}
			guard := func(context.Context, string) error {
				if mode == "guard-canceled" {
					cancel()
				}
				if mode == "guard-stale" {
					stageFixture(t, root, "0.3.6")
				}
				return nil
			}
			if _, e := switchNativePackage(ctx, root, state, p, expected, guard, nativeSwitchOps()); e == nil {
				t.Fatal("unsafe precondition accepted")
			}
		})
	}
}

func TestNativePackageSwitchFreshAndRepair(t *testing.T) {
	for _, fresh := range []bool{true, false} {
		t.Run(fmt.Sprint(fresh), func(t *testing.T) {
			root, state, p := switchFixture(t)
			expected := "0.3.4"
			if fresh {
				root = filepath.Join(filepath.Dir(root), "fresh")
				expected = "absent"
			} else {
				stageFixture(t, p.Stage, "0.3.4")
				p.Version = "0.3.4"
			}
			backup, e := switchNativePackage(context.Background(), root, state, p, expected, func(context.Context, string) error { return nil }, nativeSwitchOps())
			if e != nil {
				t.Fatal(e)
			}
			if stageVersion(root) != p.Version {
				t.Fatal("incorrect result")
			}
			if fresh {
				entries, e := os.ReadDir(backup)
				if e != nil || len(entries) != 0 {
					t.Fatal("fresh backup is not empty")
				}
			}
		})
	}
}

func TestNativePackageMoveNeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	for _, p := range []string{a, b} {
		if e := os.WriteFile(p, []byte(filepath.Base(p)), 0600); e != nil {
			t.Fatal(e)
		}
	}
	if e := movePackageNoReplace(a, b); e == nil {
		t.Fatal("occupied destination accepted")
	}
	raw, _ := os.ReadFile(b)
	if string(raw) != "b" {
		t.Fatal("destination overwritten")
	}
}

func TestNativePackageJournalRefusesExistingEvidence(t *testing.T) {
	for _, mode := range []string{"initial-existing", "initial-directory", "terminal-unrelated"} {
		t.Run(mode, func(t *testing.T) {
			stage := t.TempDir()
			path := filepath.Join(stage, "transaction.json")
			if mode == "initial-directory" {
				if e := os.Mkdir(path, 0700); e != nil {
					t.Fatal(e)
				}
			} else {
				if e := os.WriteFile(path, []byte("evidence"), 0600); e != nil {
					t.Fatal(e)
				}
			}
			state := "switching"
			if mode == "terminal-unrelated" {
				state = "completed"
			}
			if e := writePackageJournal(stage, packageJournal{Schema: 1, State: state, Backup: "package-backup-fixture", Version: "0.3.5"}); e == nil {
				t.Fatal("existing evidence accepted")
			}
			if mode != "initial-directory" {
				raw, _ := os.ReadFile(path)
				if string(raw) != "evidence" {
					t.Fatal("evidence overwritten")
				}
			}
		})
	}
}

func TestNativePackageSwitchCancellationBoundary(t *testing.T) {
	for _, during := range []bool{false, true} {
		t.Run(fmt.Sprint(during), func(t *testing.T) {
			root, state, p := switchFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ops := nativeSwitchOps()
			if during {
				ops.move = func(a, b string) error { cancel(); return movePackageNoReplace(a, b) }
			} else {
				ops.journal = func(s string, j packageJournal) error { e := writePackageJournal(s, j); cancel(); return e }
			}
			_, e := switchNativePackage(ctx, root, state, p, "0.3.4", func(context.Context, string) error { return nil }, ops)
			if during {
				if e != nil || stageVersion(root) != "0.3.5" {
					t.Fatal("cancellation interrupted transaction", e)
				}
			} else if e == nil || stageVersion(root) != "0.3.4" || switchStatus(t, p.Stage) != "switch-failed" {
				t.Fatal("canceled preflight modified installation", e)
			}
		})
	}
}

func TestNativePackageSwitchLateStaleInput(t *testing.T) {
	root, state, p := switchFixture(t)
	ops := nativeSwitchOps()
	moves := 0
	ops.journal = func(s string, j packageJournal) error {
		e := writePackageJournal(s, j)
		if j.State == "switching" {
			stageFixture(t, root, "0.3.6")
		}
		return e
	}
	ops.move = func(a, b string) error { moves++; return movePackageNoReplace(a, b) }
	if _, e := switchNativePackage(context.Background(), root, state, p, "0.3.4", func(context.Context, string) error { return nil }, ops); e == nil || moves != 0 || stageVersion(root) != "0.3.6" {
		t.Fatal("late stale input moved", e, moves)
	}
}

func TestNativePackageSwitchRollbackMoveBoundaries(t *testing.T) {
	for boundary := 7; boundary <= 12; boundary++ {
		t.Run(fmt.Sprint(boundary), func(t *testing.T) {
			root, state, p := switchFixture(t)
			ops := nativeSwitchOps()
			calls := 0
			ops.verify = func(string) (string, error) { return "", errors.New("injected post-verification failure") }
			ops.move = func(a, b string) error {
				calls++
				if calls == boundary {
					return errors.New("injected rollback failure")
				}
				return movePackageNoReplace(a, b)
			}
			backup, e := switchNativePackage(context.Background(), root, state, p, "0.3.4", func(context.Context, string) error { return nil }, ops)
			if e == nil || calls < boundary || switchStatus(t, p.Stage) != "rollback-incomplete" {
				t.Fatal("rollback failure not retained", e, calls)
			}
			if _, e = os.Stat(backup); e != nil {
				t.Fatal("backup evidence missing")
			}
		})
	}
}

func TestNativePackageSwitchActiveGuard(t *testing.T) {
	root, state, p := switchFixture(t)
	ops := nativeSwitchOps()
	moves := 0
	ops.move = func(a, b string) error { moves++; return movePackageNoReplace(a, b) }
	_, e := switchNativePackage(context.Background(), root, state, p, "0.3.4", func(context.Context, string) error { return maintenanceError("ACTIVE") }, ops)
	if e == nil || moves != 0 || stageVersion(root) != "0.3.4" {
		t.Fatal("active guard bypassed")
	}
	entries, e := os.ReadDir(state)
	if e != nil || len(entries) != 1 {
		t.Fatal("guard refusal created transaction evidence")
	}
}

func switchFixture(t *testing.T) (string, string, packageStage) {
	t.Helper()
	base := t.TempDir()
	root, state := filepath.Join(base, "install"), filepath.Join(base, "state")
	stageFixture(t, root, "0.3.4")
	for _, name := range []string{filepath.Join("Remote", "credentials"), "unrelated-shim.cmd"} {
		path := filepath.Join(root, name)
		if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(path, []byte("fixture-secret"), 0600); e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() {
			raw, e := os.ReadFile(path)
			if e != nil || string(raw) != "fixture-secret" {
				t.Error("untracked fixture changed")
			}
		})
	}
	p, e := prepareNativePackage(context.Background(), root, state, "update", "0.3.4", func(context.Context, string) error { return nil }, func(_ context.Context, s, _ string) error { stageFixture(t, s, "0.3.5"); return nil })
	if e != nil {
		t.Fatal(e)
	}
	return root, state, p
}

func TestNativePackageSwitch(t *testing.T) {
	root, state, p := switchFixture(t)
	sentinel := filepath.Join(root, "Remote", "credentials")
	backup, e := switchNativePackage(context.Background(), root, state, p, "0.3.4", func(context.Context, string) error { return nil }, nativeSwitchOps())
	if e != nil {
		t.Fatal(e)
	}
	if stageVersion(root) != "0.3.5" || stageVersion(backup) != "0.3.4" {
		t.Fatal("incorrect package versions")
	}
	raw, e := os.ReadFile(sentinel)
	if e != nil || string(raw) != "fixture-secret" {
		t.Fatal("credentials changed")
	}
	raw, e = os.ReadFile(filepath.Join(p.Stage, "transaction.json"))
	if e != nil {
		t.Fatal(e)
	}
	var j struct {
		Schema                 int
		State, Backup, Version string
	}
	if json.Unmarshal(raw, &j) != nil || j.Schema != 1 || j.State != "completed" || j.Backup != filepath.Base(backup) || j.Version != "0.3.5" {
		t.Fatal(string(raw))
	}
}
