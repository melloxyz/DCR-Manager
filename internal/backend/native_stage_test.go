package backend

import (
	"context"
	"dcr-manager/internal/app"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func stageFixture(t *testing.T, p, v string) {
	t.Helper()
	packageFixture(t, p, v, true)
	lock := "{\"lockfileVersion\":3,\"packages\":{\"\":{\"dependencies\":{\"" + dcrPackage + "\":\"" + v + "\"}},\"node_modules/" + dcrPackage + "\":{\"version\":\"" + v + "\"}}}"
	if e := os.WriteFile(filepath.Join(p, "package-lock.json"), []byte(lock), 0600); e != nil {
		t.Fatal(e)
	}
}
func TestPrepareNativePackage(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "install")
	state := filepath.Join(base, "state")
	stageFixture(t, root, "0.3.4")
	remote := filepath.Join(root, "Remote")
	if e := os.Mkdir(remote, 0700); e != nil {
		t.Fatal(e)
	}
	sentinel := filepath.Join(remote, "fixture")
	if e := os.WriteFile(sentinel, []byte("preserve"), 0600); e != nil {
		t.Fatal(e)
	}
	calls := 0
	guard := func(context.Context, string) error { calls++; return nil }
	runner := func(_ context.Context, stage, spec string) error {
		if spec != dcrPackage+"@0.3.4" {
			t.Fatal(spec)
		}
		stageFixture(t, stage, "0.3.4")
		return nil
	}
	p, e := prepareNativePackage(context.Background(), root, state, app.RepairPackage, "0.3.4", guard, runner)
	if e != nil || p.Version != "0.3.4" || !recoveryName.MatchString(filepath.Base(p.Stage)) || calls != 2 {
		t.Fatal(p, e, calls)
	}
	raw, e := os.ReadFile(sentinel)
	if e != nil || string(raw) != "preserve" {
		t.Fatal("credential fixture changed")
	}
	if _, e = os.Stat(filepath.Join(root, "package-lock.json")); e != nil {
		t.Fatal(e)
	}
}
func TestPrepareNativePackageRejectsInvalidStage(t *testing.T) {
	for _, mode := range []string{"wrong-version", "missing-lock", "mixed", "stale", "guard", "failed-runner", "invalid-lock", "lock-mismatch"} {
		t.Run(mode, func(t *testing.T) {
			base := t.TempDir()
			root := filepath.Join(base, "install")
			state := filepath.Join(base, "state")
			stageFixture(t, root, "0.3.4")
			calls := 0
			guard := func(context.Context, string) error {
				calls++
				if mode == "guard" && calls == 2 {
					return errors.New("ACTIVE")
				}
				return nil
			}
			runner := func(_ context.Context, p, spec string) error {
				stageFixture(t, p, "0.3.4")
				switch mode {
				case "wrong-version":
					stageFixture(t, p, "0.3.5")
				case "missing-lock":
					if e := os.Remove(filepath.Join(p, "package-lock.json")); e != nil {
						t.Fatal(e)
					}
				case "invalid-lock":
					if e := os.WriteFile(filepath.Join(p, "package-lock.json"), []byte("{}"), 0600); e != nil {
						t.Fatal(e)
					}
				case "lock-mismatch":
					lock := "{\"lockfileVersion\":3,\"packages\":{\"\":{\"dependencies\":{\"" + dcrPackage + "\":\"0.3.4\"}},\"node_modules/" + dcrPackage + "\":{\"version\":\"0.3.5\"}}}"
					if e := os.WriteFile(filepath.Join(p, "package-lock.json"), []byte(lock), 0600); e != nil {
						t.Fatal(e)
					}
				case "mixed":
					packageFixture(t, p, "0.3.4", false)
				case "stale":
					stageFixture(t, root, "0.3.5")
				case "failed-runner":
					return errors.New("fixture secret")
				}
				return nil
			}
			if p, e := prepareNativePackage(context.Background(), root, state, app.RepairPackage, "0.3.4", guard, runner); e == nil || p.Stage != "" {
				t.Fatal("unsafe stage accepted", p)
			}
			if _, e := os.Stat(filepath.Join(root, "node_modules")); e != nil {
				t.Fatal("original missing")
			}
		})
	}
}
func TestPrepareNativePackageInstallAndUpdate(t *testing.T) {
	for _, c := range []struct {
		action   app.MaintenanceAction
		expected string
	}{{app.RepairPackage, "absent"}, {app.UpdatePackage, "0.3.4"}} {
		base := t.TempDir()
		root := filepath.Join(base, "install")
		state := filepath.Join(base, "state")
		if c.expected != "absent" {
			stageFixture(t, root, c.expected)
		}
		p, e := prepareNativePackage(context.Background(), root, state, c.action, c.expected, func(context.Context, string) error { return nil }, func(_ context.Context, p, spec string) error {
			if spec != dcrPackage+"@latest" {
				t.Fatal(spec)
			}
			stageFixture(t, p, "0.3.5")
			return nil
		})
		if e != nil || p.Version != "0.3.5" {
			t.Fatal(p, e)
		}
		_, v, _ := inspectInstallation("dedicated", filepath.Join(root, "node_modules"))
		if v != c.expected {
			t.Fatal("installation changed", v)
		}
	}
}
func TestNativeStageOwnershipAndGuard(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "install")
	state := filepath.Join(base, "state")
	packageFixture(t, root, "0.3.4", false)
	run := func(context.Context, string, string) error { t.Fatal("runner called"); return nil }
	guard := func(context.Context, string) error { return nil }
	if _, e := prepareNativePackage(context.Background(), root, state, app.RepairPackage, "0.3.4", guard, run); e == nil {
		t.Fatal("mixed install allowed")
	}
	if _, e := prepareNativePackage(context.Background(), base, filepath.Join(base, "state"), app.RepairPackage, "absent", guard, run); e == nil {
		t.Fatal("overlapping roots allowed")
	}
}

func TestNativeStageCancelled(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "install")
	state := filepath.Join(base, "state")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p, e := prepareNativePackage(ctx, root, state, app.RepairPackage, "absent", func(context.Context, string) error { t.Fatal("guard invoked"); return nil }, func(context.Context, string, string) error { t.Fatal("runner invoked"); return nil })
	if !errors.Is(e, context.Canceled) || p.Stage != "" {
		t.Fatal(p, e)
	}
	if _, e := os.Stat(state); !os.IsNotExist(e) {
		t.Fatal("cancelled stage created state")
	}
}
