package backend

import (
	"context"
	"dcr-manager/internal/app"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const pkgBackup = "package-backup-11111111111111111111111111111111"
const cacheBackup = "cache-backup-22222222222222222222222222222222"
const txnStage = "stage-33333333333333333333333333333333"

func writeFixture(t *testing.T, p, s string) {
	t.Helper()
	if e := os.WriteFile(p, []byte(s), 0600); e != nil {
		t.Fatal(e)
	}
}
func backupFixture(t *testing.T, state, name string) string {
	t.Helper()
	p := filepath.Join(state, name)
	packageFixture(t, p, "0.3.4", true)
	writeFixture(t, filepath.Join(p, "package-lock.json"), "{}")
	return p
}
func TestNativeRecoveryInventory(t *testing.T) {
	state := t.TempDir()
	cache := t.TempDir()
	npx := filepath.Join(cache, "_npx")
	if e := os.Mkdir(npx, 0700); e != nil {
		t.Fatal(e)
	}
	p := backupFixture(t, state, pkgBackup)
	q := backupFixture(t, state, cacheBackup)
	target := filepath.Join(npx, "1234abcd")
	meta, _ := json.Marshal(map[string]string{"originalPath": target, "backupPath": q})
	writeFixture(t, q+".json", string(meta))
	stage := filepath.Join(state, txnStage)
	if e := os.Mkdir(stage, 0700); e != nil {
		t.Fatal(e)
	}
	writeFixture(t, filepath.Join(stage, "transaction.json"), "{\"schema\":1,\"state\":\"rollback-incomplete\",\"secret\":\"fixture-private\"}")
	old := time.Now().Add(-31 * 24 * time.Hour)
	if e := os.Chtimes(p, old, old); e != nil {
		t.Fatal(e)
	}
	rows, known := nativeRecovery(context.Background(), state, cache)
	if !known || len(rows) != 3 {
		t.Fatal(rows, known)
	}
	by := map[string]app.Backup{}
	for _, r := range rows {
		by[r.Name] = r
	}
	if a := by[pkgBackup]; a.Status != "ready" || !a.Review || a.AgeDays < 30 || !a.SizeKnown || !cacheTokenPattern.MatchString(a.Token) {
		t.Fatal(a)
	}
	if a := by[cacheBackup]; a.Status != "ready" || !cacheTokenPattern.MatchString(a.Token) {
		t.Fatal(a)
	}
	if a := by[txnStage]; a.Status != "rollback-incomplete" || a.Token != "" {
		t.Fatal(a)
	}
	raw, _ := json.Marshal(rows)
	if strings.Contains(string(raw), "fixture-private") || strings.Contains(string(raw), "originalPath") {
		t.Fatal("private data leaked")
	}
	before := by[pkgBackup].Token
	writeFixture(t, filepath.Join(p, "package-lock.json"), "{\"changed\":true}")
	rows, _ = nativeRecovery(context.Background(), state, cache)
	for _, r := range rows {
		if r.Name == pkgBackup && r.Token == before {
			t.Fatal("stale token survived")
		}
	}
	if e := os.Mkdir(target, 0700); e != nil {
		t.Fatal(e)
	}
	rows, _ = nativeRecovery(context.Background(), state, cache)
	for _, r := range rows {
		if r.Name == cacheBackup && (r.Status != "destination-blocked" || r.Token != "") {
			t.Fatal(r)
		}
	}
}
func TestRecoveryParityFailClosed(t *testing.T) {
	b := app.Backup{Name: pkgBackup, Kind: "package", Status: "ready", Token: strings.Repeat("a", 64), Version: "0.3.4", SizeKnown: true, Bytes: 42}
	s := app.Snapshot{RecoveryKnown: true, Backups: []app.Backup{b}}
	applyRecoveryParity(&s, []app.Backup{b}, true, true)
	if !s.RecoveryKnown || s.Backups[0].Token != b.Token {
		t.Fatal(s)
	}
	mismatch := b
	mismatch.Token = strings.Repeat("b", 64)
	applyRecoveryParity(&s, []app.Backup{mismatch}, true, true)
	if s.Backups[0].Status == "ready" || s.Backups[0].Token != "" {
		t.Fatal("mismatched token authorized")
	}
}
func TestRecoveryInvalidAndCancelled(t *testing.T) {
	state := t.TempDir()
	if rows, known := nativeRecovery(context.Background(), filepath.Join(state, "missing"), ""); !known || len(rows) != 0 {
		t.Fatal(rows, known)
	}
	if _, known := nativeRecovery(context.Background(), "relative", ""); known {
		t.Fatal("relative accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, known := nativeRecovery(ctx, state, ""); known {
		t.Fatal("cancelled accepted")
	}
	p := backupFixture(t, state, pkgBackup)
	if e := os.Remove(filepath.Join(p, "package-lock.json")); e != nil {
		t.Fatal(e)
	}
	rows, known := nativeRecovery(context.Background(), state, "")
	if !known || len(rows) != 1 || rows[0].Status != "incomplete" || rows[0].Token != "" {
		t.Fatal(rows, known)
	}
}
