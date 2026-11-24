package backend

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var packageArtifacts = []string{"node_modules", "package.json", "package-lock.json"}

var (
	errPackageJournal = errors.New("package transaction journal failed; preserve stage and backup for review")
	errPackageSwitch  = errors.New("package switch failed; original package restored")
)

type packageJournal struct {
	Schema  int    `json:"schema"`
	State   string `json:"state"`
	Backup  string `json:"backup"`
	Version string `json:"version"`
}

type packageSwitchOps struct {
	move    func(string, string) error
	journal func(string, packageJournal) error
	verify  func(string) (string, error)
}

func nativeSwitchOps() packageSwitchOps {
	return packageSwitchOps{move: movePackageNoReplace, journal: writePackageJournal, verify: validatePreparedPackage}
}

// Track only package artifacts; Remote credentials and unrelated files never enter the plan.
func packageTree(root string) (map[string]os.FileInfo, error) {
	tree := map[string]os.FileInfo{}
	for _, name := range packageArtifacts {
		start := filepath.Join(root, name)
		if _, e := plainInfo(start); os.IsNotExist(e) {
			continue
		} else if e != nil {
			return nil, e
		}
		e := filepath.WalkDir(start, func(p string, _ fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			i, e := plainInfo(p)
			if e != nil {
				return e
			}
			if !i.IsDir() && !i.Mode().IsRegular() {
				return maintenanceError("UNSAFE")
			}
			if !os.SameFile(i, i) {
				return maintenanceError("UNSAFE")
			}
			if len(tree) >= 100000 {
				return errRecoveryLimit
			}
			rel, e := filepath.Rel(root, p)
			if e != nil {
				return e
			}
			tree[rel] = i
			return nil
		})
		if e != nil {
			return nil, e
		}
	}
	return tree, nil
}

func samePackageTree(root string, want map[string]os.FileInfo) bool {
	got, e := packageTree(root)
	if e != nil || len(got) != len(want) {
		return false
	}
	for p, i := range want {
		n, ok := got[p]
		if !ok || !os.SameFile(i, n) || i.Size() != n.Size() || !i.ModTime().Equal(n.ModTime()) {
			return false
		}
	}
	return true
}

func writePackageJournal(stage string, j packageJournal) error {
	path := filepath.Join(stage, "transaction.json")
	var previous os.FileInfo
	if j.State != "switching" {
		var e error
		previous, e = plainInfo(path)
		if e != nil || !previous.Mode().IsRegular() {
			return maintenanceError("UNSAFE")
		}
		if !os.SameFile(previous, previous) {
			return maintenanceError("UNSAFE")
		}
		raw, e := readPlainFile(path, maxConfigBytes)
		if e != nil {
			return e
		}
		var old packageJournal
		if json.Unmarshal(raw, &old) != nil || old.Schema != 1 || old.Backup != j.Backup || old.Version != j.Version || old.State != "switching" {
			return maintenanceError("UNSAFE")
		}
	} else if _, e := os.Lstat(path); !os.IsNotExist(e) {
		return maintenanceError("UNSAFE")
	}
	raw, e := json.Marshal(j)
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(stage, "transaction-*.tmp")
	if e != nil {
		return e
	}
	// Failed writes remain as evidence. Never truncate a pre-existing journal.
	if _, e = f.Write(raw); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	if previous != nil {
		now, e := plainInfo(path)
		if e != nil || !os.SameFile(previous, now) {
			return maintenanceError("UNSAFE")
		}
	}
	return movePackageJournal(f.Name(), path, previous != nil)
}

// Isolated primitive only: caller must eventually hold the shared Windows mutex.
// No context cancellation after the first move: finish or independently roll back.
func switchNativePackage(ctx context.Context, root, state string, p packageStage, expected string, guard stageGuard, ops packageSwitchOps) (string, error) {
	if e := ctx.Err(); e != nil {
		return "", e
	}
	if guard == nil || ops.move == nil || ops.journal == nil || ops.verify == nil {
		return "", maintenanceError("UNSAFE")
	}
	check := func() error {
		if e := stageRoots(root, state); e != nil {
			return e
		}
		if filepath.Dir(p.Stage) != state || !strings.HasPrefix(filepath.Base(p.Stage), "stage-") || !recoveryName.MatchString(filepath.Base(p.Stage)) {
			return maintenanceError("UNSAFE")
		}
		i, e := plainInfo(p.Stage)
		if e != nil || p.identity == nil || !os.SameFile(i, p.identity) {
			return maintenanceError("UNSAFE")
		}
		if e = stageOwnership(root); e != nil {
			return e
		}
		if stageVersion(root) != expected {
			return maintenanceError("STALE")
		}
		v, e := validatePreparedPackage(p.Stage)
		if e != nil {
			return e
		}
		if v != p.Version {
			return maintenanceError("STALE")
		}
		if _, e := measureRecovery(ctx, p.Stage); e != nil {
			return maintenanceError("UNSAFE")
		}
		if _, e = os.Lstat(filepath.Join(p.Stage, "transaction.json")); !os.IsNotExist(e) {
			return maintenanceError("UNSAFE")
		}
		return nil
	}
	if e := check(); e != nil {
		return "", e
	}
	before, e := packageTree(root)
	if e != nil {
		return "", maintenanceError("UNSAFE")
	}
	prepared, e := packageTree(p.Stage)
	if e != nil {
		return "", maintenanceError("UNSAFE")
	}
	if e = guard(ctx, root); e != nil {
		return "", e
	}
	if e = ctx.Err(); e != nil {
		return "", e
	}
	if e = check(); e != nil {
		return "", e
	}
	if !samePackageTree(root, before) || !samePackageTree(p.Stage, prepared) {
		return "", maintenanceError("STALE")
	}
	var id [16]byte
	if _, e = rand.Read(id[:]); e != nil {
		return "", e
	}
	backup := filepath.Join(state, "package-backup-"+hex.EncodeToString(id[:]))
	if e = os.Mkdir(backup, 0700); e != nil {
		return "", e
	}
	if _, e = plainInfo(root); os.IsNotExist(e) {
		if e = os.Mkdir(root, 0700); e != nil {
			return backup, e
		}
	} else if e != nil {
		return backup, e
	}
	journal := packageJournal{Schema: 1, State: "switching", Backup: filepath.Base(backup), Version: p.Version}
	if e = ops.journal(p.Stage, journal); e != nil {
		return backup, errPackageJournal
	}
	type movedFile struct {
		from, to string
		identity os.FileInfo
	}
	var moves []movedFile
	move := func(from, to string) error {
		identity, e := plainInfo(from)
		if e != nil {
			return e
		}
		// Windows FileInfo loads its file ID lazily; capture it before the path moves.
		if !os.SameFile(identity, identity) {
			return maintenanceError("UNSAFE")
		}
		if _, e := os.Lstat(to); !os.IsNotExist(e) {
			return maintenanceError("UNSAFE")
		}
		if parent, e := plainInfo(filepath.Dir(to)); e != nil || !parent.IsDir() {
			return maintenanceError("UNSAFE")
		}
		if e := ops.move(from, to); e != nil {
			return e
		}
		moves = append(moves, movedFile{from, to, identity})
		return nil
	}
	failure := func() error {
		if e := ctx.Err(); e != nil {
			return e
		}
		if e := stageRoots(root, state); e != nil {
			return e
		}
		if stageVersion(root) != expected || !samePackageTree(root, before) || !samePackageTree(p.Stage, prepared) {
			return maintenanceError("STALE")
		}
		for _, name := range packageArtifacts {
			dest := filepath.Join(root, name)
			if _, ok := before[name]; ok {
				if e := move(dest, filepath.Join(backup, name)); e != nil {
					return e
				}
			}
			if e := move(filepath.Join(p.Stage, name), dest); e != nil {
				return e
			}
		}
		v, e := ops.verify(root)
		if e != nil || v != p.Version || !samePackageTree(root, prepared) || !samePackageTree(backup, before) {
			return maintenanceError("VERIFY")
		}
		journal.State = "completed"
		return ops.journal(p.Stage, journal)
	}()
	if failure == nil {
		return backup, nil
	}
	var rollbackErrors []error
	for i := len(moves) - 1; i >= 0; i-- {
		m := moves[i]
		now, e := plainInfo(m.to)
		if e != nil || !os.SameFile(m.identity, now) {
			rollbackErrors = append(rollbackErrors, maintenanceError("UNSAFE"))
			continue
		}
		if _, e := os.Lstat(m.from); !os.IsNotExist(e) {
			rollbackErrors = append(rollbackErrors, maintenanceError("UNSAFE"))
			continue
		}
		if parent, e := plainInfo(filepath.Dir(m.from)); e != nil || !parent.IsDir() {
			rollbackErrors = append(rollbackErrors, maintenanceError("UNSAFE"))
			continue
		}
		if e := ops.move(m.to, m.from); e != nil {
			rollbackErrors = append(rollbackErrors, e)
		}
	}
	if !samePackageTree(root, before) || !samePackageTree(p.Stage, prepared) {
		rollbackErrors = append(rollbackErrors, maintenanceError("VERIFY"))
	}
	journal.State = "switch-failed"
	if len(rollbackErrors) > 0 {
		journal.State = "rollback-incomplete"
	}
	journalError := ops.journal(p.Stage, journal)
	// Return fixed codes: filesystem errors can contain private paths.
	if len(rollbackErrors) > 0 {
		if journalError != nil {
			return backup, errors.Join(maintenanceError("ROLLBACK"), errPackageJournal)
		}
		return backup, maintenanceError("ROLLBACK")
	}
	if journalError != nil {
		return backup, errPackageJournal
	}
	return backup, errPackageSwitch
}
