package backend

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

type cacheSidecar struct {
	OriginalPath string `json:"originalPath"`
	BackupPath   string `json:"backupPath"`
}

type cacheQuarantineOps struct {
	move    func(string, string) error
	sidecar func(string, cacheSidecar) error
}

func nativeQuarantineOps() cacheQuarantineOps {
	return cacheQuarantineOps{move: movePackageNoReplace, sidecar: writeCacheSidecar}
}

var (
	errCacheSidecar = errors.New("cache sidecar write failed; preserve transaction evidence")
	errCacheMove    = errors.New("cache quarantine failed; preserve source and sidecar for review")
	errCacheVerify  = errors.New("cache quarantine not verified; preserve backup and sidecar for recovery")
)

func writeCacheSidecar(backup string, meta cacheSidecar) error {
	parent, e := plainInfo(filepath.Dir(backup))
	if e != nil || !parent.IsDir() {
		return maintenanceError("UNSAFE")
	}
	for _, p := range []string{backup, backup + ".json"} {
		if _, e := os.Lstat(p); !os.IsNotExist(e) {
			return maintenanceError("UNSAFE")
		}
	}
	raw, e := json.Marshal(meta)
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(backup), "cache-sidecar-*.tmp")
	if e != nil {
		return e
	}
	// Keep failed writes as evidence; publish only a fully flushed sidecar, without replacement.
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
	return movePackageNoReplace(f.Name(), backup+".json")
}

func quarantineCandidate(ctx context.Context, cache, id, expected string) (string, os.FileInfo, recoveryMeasure, error) {
	var zero recoveryMeasure
	source := filepath.Join(cache, "_npx", id)
	info, e := plainInfo(source)
	if e != nil || !info.IsDir() || !os.SameFile(info, info) {
		return source, nil, zero, maintenanceError("UNSAFE")
	}
	if e = stageOwnership(source); e != nil {
		return source, nil, zero, e
	}
	raw, e := readPlainFile(filepath.Join(source, "package.json"), maxConfigBytes)
	if e != nil || !exclusiveManifest(raw) {
		return source, nil, zero, maintenanceError("UNSAFE")
	}
	_, version, _ := inspectInstallation("npx", filepath.Join(source, "node_modules"))
	candidate, ok := cacheCandidate(source, info, version, raw)
	if !ok || candidate.Token != expected {
		return source, nil, zero, maintenanceError("STALE")
	}
	measure, e := measureRecovery(ctx, source)
	if e != nil {
		return source, nil, zero, maintenanceError("UNSAFE")
	}
	return source, info, measure, nil
}

// Isolated single-directory transaction. Keep Maintain on PowerShell until shared
// mutex, in-mutex all-process checks and native restore acceptance are complete.
func quarantineNativeCache(ctx context.Context, cache, state, id, expected string, guard stageGuard, ops cacheQuarantineOps) (string, error) {
	if e := ctx.Err(); e != nil {
		return "", e
	}
	if guard == nil || ops.move == nil || ops.sidecar == nil || !cacheIDPattern.MatchString(id) || !cacheTokenPattern.MatchString(expected) {
		return "", maintenanceError("UNSAFE")
	}
	if e := stageRoots(cache, state); e != nil {
		return "", e
	}
	source, identity, measure, e := quarantineCandidate(ctx, cache, id, expected)
	if e != nil {
		return "", e
	}
	if e = guard(ctx, source); e != nil {
		return "", e
	}
	if e = ctx.Err(); e != nil {
		return "", e
	}
	if e = stageRoots(cache, state); e != nil {
		return "", e
	}
	_, now, current, e := quarantineCandidate(ctx, cache, id, expected)
	if e != nil {
		return "", e
	}
	if !os.SameFile(identity, now) || current != measure {
		return "", maintenanceError("STALE")
	}
	if _, e = plainInfo(state); os.IsNotExist(e) {
		if e = os.Mkdir(state, 0700); e != nil {
			return "", maintenanceError("UNSAFE")
		}
	} else if e != nil {
		return "", maintenanceError("UNSAFE")
	}
	var random [16]byte
	if _, e = rand.Read(random[:]); e != nil {
		return "", maintenanceError("UNSAFE")
	}
	backup := filepath.Join(state, "cache-backup-"+hex.EncodeToString(random[:]))
	meta := cacheSidecar{OriginalPath: source, BackupPath: backup}
	if e = ops.sidecar(backup, meta); e != nil {
		return backup, errCacheSidecar
	}
	// A second guard runs after the sidecar is durable, immediately before final validation.
	if e = guard(ctx, source); e != nil {
		return backup, e
	}
	if e = ctx.Err(); e != nil {
		return backup, e
	}
	if e = stageRoots(cache, state); e != nil {
		return backup, e
	}
	_, now, current, e = quarantineCandidate(ctx, cache, id, expected)
	if e != nil {
		return backup, e
	}
	if !os.SameFile(identity, now) || current != measure {
		return backup, maintenanceError("STALE")
	}
	if _, e = os.Lstat(backup); !os.IsNotExist(e) {
		return backup, maintenanceError("UNSAFE")
	}
	raw, e := readPlainFile(backup+".json", maxConfigBytes)
	var written cacheSidecar
	if e != nil || json.Unmarshal(raw, &written) != nil || written != meta {
		return backup, errCacheSidecar
	}
	manifest, e := readPlainFile(filepath.Join(source, "package.json"), maxConfigBytes)
	if e != nil {
		return backup, maintenanceError("UNSAFE")
	}
	version := stageVersion(source)
	if e = ctx.Err(); e != nil {
		return backup, e
	}
	if e = ops.move(source, backup); e != nil {
		return backup, errCacheMove
	}
	// Cancellation after the rename must not interrupt evidence verification.
	verifyCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	moved, e := plainInfo(backup)
	if e != nil || !os.SameFile(identity, moved) {
		return backup, errCacheVerify
	}
	after, e := measureRecovery(verifyCtx, backup)
	if e != nil || after != measure {
		return backup, errCacheVerify
	}
	if _, e = os.Lstat(source); !os.IsNotExist(e) {
		return backup, errCacheVerify
	}
	if _, e = cacheRestoreMetadata(backup, cache); e != nil {
		return backup, errCacheVerify
	}
	installed, e := readPlainFile(filepath.Join(backup, "package.json"), maxConfigBytes)
	if e != nil || !bytes.Equal(manifest, installed) || stageVersion(backup) != version {
		return backup, errCacheVerify
	}
	metadata, e := readPlainFile(backup+".json", maxConfigBytes)
	if e != nil || !bytes.Equal(raw, metadata) {
		return backup, errCacheVerify
	}
	return backup, nil
}
