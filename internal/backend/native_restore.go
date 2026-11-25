package backend

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"dcr-manager/internal/app"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type restoreOps struct {
	copy      func(context.Context, string, string) error
	move      func(string, string) error
	switchOps packageSwitchOps
}

func nativeRestoreOps() restoreOps {
	ops := nativeSwitchOps()
	ops.verify = validatePackageFiles
	return restoreOps{copy: copyRestorePackage, move: movePackageNoReplace, switchOps: ops}
}

var errRestoreCopy = errors.New("backup staging failed; preserve source and restore stage for review")
var errRestoreVerify = errors.New("backup restore not verified; preserve recovery evidence")

// Read an unchanged plain file, with cancellation and an exact size bound.
func readRestoreFile(ctx context.Context, path string, out io.Writer) error {
	info, e := plainInfo(path)
	if e != nil || !info.Mode().IsRegular() {
		return maintenanceError("UNSAFE")
	}
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	opened, e := f.Stat()
	if e != nil || !os.SameFile(info, opened) {
		return maintenanceError("UNSAFE")
	}
	left := info.Size()
	for left > 0 {
		if e = ctx.Err(); e != nil {
			return e
		}
		n, e := io.CopyN(out, f, min(left, 64*1024))
		if e != nil {
			return e
		}
		left -= n
	}
	var extra [1]byte
	if n, e := f.Read(extra[:]); n != 0 || e != io.EOF {
		return maintenanceError("STALE")
	}
	after, e := f.Stat()
	if e != nil || after.Size() != info.Size() || !after.ModTime().Equal(info.ModTime()) {
		return maintenanceError("STALE")
	}
	return ctx.Err()
}

func copyRestorePackage(ctx context.Context, source, stage string) error {
	tree, e := packageTree(source)
	if e != nil {
		return e
	}
	names := make([]string, 0, len(tree))
	for name := range tree {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if e = ctx.Err(); e != nil {
			return e
		}
		target := filepath.Join(stage, name)
		if parent, e := plainInfo(filepath.Dir(target)); e != nil || !parent.IsDir() {
			return maintenanceError("UNSAFE")
		}
		if tree[name].IsDir() {
			if e = os.Mkdir(target, 0700); e != nil {
				return e
			}
			continue
		}
		f, e := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			return e
		}
		e = readRestoreFile(ctx, filepath.Join(source, name), f)
		if e == nil {
			e = f.Sync()
		}
		closed := f.Close()
		if e != nil {
			return e
		}
		if closed != nil {
			return closed
		}
	}
	if !samePackageTree(source, tree) {
		return maintenanceError("STALE")
	}
	return nil
}

func equalRestoreContents(ctx context.Context, source, stage string) bool {
	before, e := packageTree(source)
	if e != nil {
		return false
	}
	after, e := packageTree(stage)
	if e != nil || len(before) != len(after) {
		return false
	}
	for name, info := range before {
		other, ok := after[name]
		if !ok || info.IsDir() != other.IsDir() || (!info.IsDir() && info.Size() != other.Size()) {
			return false
		}
		if info.IsDir() {
			continue
		}
		a, b := sha256.New(), sha256.New()
		if readRestoreFile(ctx, filepath.Join(source, name), a) != nil || readRestoreFile(ctx, filepath.Join(stage, name), b) != nil || string(a.Sum(nil)) != string(b.Sum(nil)) {
			return false
		}
	}
	return ctx.Err() == nil
}

func checkedRestore(ctx context.Context, state, cache string, r app.MaintenanceRequest) (app.Backup, error) {
	row := inspectRecovery(ctx, state, cache, r.CacheID, time.Now())
	if row.Status != "ready" || row.Token != r.Expected {
		return app.Backup{}, maintenanceError("STALE")
	}
	return row, nil
}

// Internal only. Production maintenance remains on PowerShell until D.4/D.5.
func restoreNativeBackup(ctx context.Context, root, state, cache string, r app.MaintenanceRequest, guard stageGuard, ops restoreOps) (string, error) {
	if e := ctx.Err(); e != nil {
		return "", e
	}
	if _, e := maintenanceArgs(r); e != nil || r.Action != app.RestoreBackup || guard == nil || ops.copy == nil || ops.move == nil {
		return "", maintenanceError("UNSAFE")
	}
	destination := root
	if strings.HasPrefix(r.CacheID, "cache-") {
		destination = cache
	}
	if e := stageRoots(destination, state); e != nil {
		return "", e
	}
	row, e := checkedRestore(ctx, state, cache, r)
	if e != nil {
		return "", e
	}
	source := filepath.Join(state, r.CacheID)
	identity, e := plainInfo(source)
	if e != nil || !os.SameFile(identity, identity) {
		return "", maintenanceError("UNSAFE")
	}
	recheck := func() error {
		if e := ctx.Err(); e != nil {
			return e
		}
		if e := stageRoots(destination, state); e != nil {
			return e
		}
		now, e := plainInfo(source)
		if e != nil || !os.SameFile(identity, now) {
			return maintenanceError("STALE")
		}
		_, e = checkedRestore(ctx, state, cache, r)
		return e
	}
	if e = guard(ctx, source); e != nil {
		return "", e
	}
	if e = recheck(); e != nil {
		return "", e
	}
	if row.Kind == "cache" {
		return restoreCacheBackup(ctx, cache, source, identity, guard, ops, recheck)
	}
	if e = stageOwnership(root); e != nil {
		return "", e
	}
	if stageVersion(root) != r.CurrentVersion {
		return "", maintenanceError("STALE")
	}
	var id [16]byte
	if _, e = rand.Read(id[:]); e != nil {
		return "", maintenanceError("UNSAFE")
	}
	stage := filepath.Join(state, "restore-stage-"+hex.EncodeToString(id[:]))
	if e = os.Mkdir(stage, 0700); e != nil {
		return "", maintenanceError("UNSAFE")
	}
	stageIdentity, e := plainInfo(stage)
	if e != nil || !os.SameFile(stageIdentity, stageIdentity) {
		return "", maintenanceError("UNSAFE")
	}
	if e = ops.copy(ctx, source, stage); e != nil {
		return "", errRestoreCopy
	}
	now, e := plainInfo(stage)
	if e != nil || !os.SameFile(stageIdentity, now) {
		return "", maintenanceError("UNSAFE")
	}
	if version, e := validatePackageFiles(stage); e != nil || version != row.Version || !equalRestoreContents(ctx, source, stage) {
		return "", errRestoreCopy
	}
	if e = recheck(); e != nil {
		return "", e
	}
	switchGuard := func(c context.Context, p string) error {
		if e := guard(c, p); e != nil {
			return e
		}
		if e := recheck(); e != nil {
			return e
		}
		if !equalRestoreContents(c, source, stage) {
			return errRestoreCopy
		}
		return nil
	}
	return switchNativePackage(ctx, root, state, packageStage{Stage: stage, Version: row.Version, identity: stageIdentity}, r.CurrentVersion, switchGuard, ops.switchOps)
}

func restoreCacheBackup(ctx context.Context, cache, source string, identity os.FileInfo, guard stageGuard, ops restoreOps, recheck func() error) (string, error) {
	raw, e := cacheRestoreMetadata(source, cache)
	if e != nil {
		return "", maintenanceError("UNSAFE")
	}
	var meta cacheSidecar
	if json.Unmarshal([]byte(strings.TrimPrefix(string(raw), "\uFEFF")), &meta) != nil {
		return "", maintenanceError("UNSAFE")
	}
	target := filepath.Clean(meta.OriginalPath)
	version := stageVersion(source)
	before, e := packageTree(source)
	if e != nil {
		return "", maintenanceError("UNSAFE")
	}
	measure, e := measureRecovery(ctx, source)
	if e != nil {
		return "", maintenanceError("UNSAFE")
	}
	if e = guard(ctx, target); e != nil {
		return "", e
	}
	if e = recheck(); e != nil {
		return "", e
	}
	if _, e = plainInfo(target); !os.IsNotExist(e) {
		return "", maintenanceError("UNSAFE")
	}
	if e = ctx.Err(); e != nil {
		return "", e
	}
	if e = ops.move(source, target); e != nil {
		return target, errRestoreVerify
	}
	post, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	now, e := plainInfo(target)
	if e != nil || !os.SameFile(identity, now) {
		return target, errRestoreVerify
	}
	after, e := measureRecovery(post, target)
	if e != nil || after != measure || !samePackageTree(target, before) || stageVersion(target) != version {
		return target, errRestoreVerify
	}
	if _, e = os.Lstat(source); !os.IsNotExist(e) {
		return target, errRestoreVerify
	}
	metadata, e := readPlainFile(source+".json", maxConfigBytes)
	if e != nil || !bytes.Equal(raw, metadata) {
		return target, errRestoreVerify
	}
	return target, nil
}
