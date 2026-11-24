package backend

import (
	"context"
	"crypto/rand"
	"dcr-manager/internal/app"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type packageStage struct {
	Stage, Version string
	identity       os.FileInfo
}
type stageGuard func(context.Context, string) error
type stageRunner func(context.Context, string, string) error

func stageOwnership(root string) error {
	info, e := plainInfo(root)
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil || !info.IsDir() {
		return maintenanceError("UNSAFE")
	}
	manifest := filepath.Join(root, "package.json")
	raw, e := readPlainFile(manifest, maxConfigBytes)
	missing := os.IsNotExist(e)
	if !missing && (e != nil || !exclusiveManifest(raw)) {
		return maintenanceError("UNSAFE")
	}
	for _, name := range []string{"node_modules", "package-lock.json"} {
		i, e := plainInfo(filepath.Join(root, name))
		if os.IsNotExist(e) {
			continue
		}
		if e != nil || missing || (name == "node_modules" && !i.IsDir()) || (name != "node_modules" && !i.Mode().IsRegular()) {
			return maintenanceError("UNSAFE")
		}
	}
	return nil
}
func stageVersion(root string) string {
	_, v, _ := inspectInstallation("dedicated", filepath.Join(root, "node_modules"))
	return v
}
func validatePreparedPackage(stage string) (string, error) {
	if e := stageOwnership(stage); e != nil {
		return "", e
	}
	row, v, _ := inspectInstallation("dedicated", filepath.Join(stage, "node_modules"))
	if !versionPattern.MatchString(v) || !row.EntryExists {
		return "", maintenanceError("NPM")
	}
	raw, e := readPlainFile(filepath.Join(stage, "package.json"), maxConfigBytes)
	var top struct{ Dependencies map[string]string }
	if e != nil || json.Unmarshal([]byte(strings.TrimPrefix(string(raw), "\uFEFF")), &top) != nil || top.Dependencies[dcrPackage] != v {
		return "", maintenanceError("NPM")
	}
	// npm v7+ lockfile structure; require pinned root and installed package version.
	raw, e = readPlainFile(filepath.Join(stage, "package-lock.json"), 16*1024*1024)
	var lock struct {
		LockfileVersion int
		Packages        map[string]struct {
			Version      string
			Dependencies map[string]string
		}
	}
	if e != nil || json.Unmarshal([]byte(strings.TrimPrefix(string(raw), "\uFEFF")), &lock) != nil || lock.LockfileVersion < 2 || lock.Packages[""].Dependencies[dcrPackage] != v || lock.Packages["node_modules/"+dcrPackage].Version != v {
		return "", maintenanceError("NPM")
	}
	return v, nil
}
func canonicalStageRoot(p string) (string, error) {
	if !filepath.IsAbs(p) || filepath.Clean(p) != p || filepath.Dir(p) == p {
		return "", maintenanceError("UNSAFE")
	}
	info, e := plainInfo(p)
	if os.IsNotExist(e) {
		// Require an existing, validated parent. Do not recursively create arbitrary roots.
		parent := filepath.Dir(p)
		i, e := plainInfo(parent)
		if e != nil || !i.IsDir() {
			return "", maintenanceError("UNSAFE")
		}
		full, ok := legacyTokenPath(parent, i)
		if !ok {
			return "", maintenanceError("UNSAFE")
		}
		return filepath.Join(full, strings.ToLower(filepath.Base(p))), nil
	}
	if e != nil || !info.IsDir() {
		return "", maintenanceError("UNSAFE")
	}
	full, ok := legacyTokenPath(p, info)
	if !ok {
		return "", maintenanceError("UNSAFE")
	}
	return full, nil
}
func stageRoots(root, state string) error {
	a, e := canonicalStageRoot(root)
	if e != nil {
		return e
	}
	b, e := canonicalStageRoot(state)
	if e != nil {
		return e
	}
	sep := string(filepath.Separator)
	if a == b || strings.HasPrefix(a, b+sep) || strings.HasPrefix(b, a+sep) || !strings.EqualFold(filepath.VolumeName(a), filepath.VolumeName(b)) {
		return maintenanceError("UNSAFE")
	}
	return nil
}

// Prepare only. No package switch, credentials, cache quarantine or active install
// writes are performed here. The caller must eventually hold the shared mutex and
// repeat all checks before a separate journaled transaction. Not wired to Maintain.
func prepareNativePackage(ctx context.Context, root, state string, action app.MaintenanceAction, expected string, guard stageGuard, run stageRunner) (packageStage, error) {
	var zero packageStage
	if ctx.Err() != nil {
		return zero, ctx.Err()
	}
	if guard == nil || run == nil {
		return zero, maintenanceError("UNSAFE")
	}
	if _, e := maintenanceArgs(app.MaintenanceRequest{Action: action, Expected: expected}); e != nil {
		return zero, e
	}
	if action != app.RepairPackage && action != app.UpdatePackage {
		return zero, maintenanceError("UNSAFE")
	}
	if e := stageRoots(root, state); e != nil {
		return zero, e
	}
	if e := stageOwnership(root); e != nil {
		return zero, e
	}
	if stageVersion(root) != expected {
		return zero, maintenanceError("STALE")
	}
	if e := guard(ctx, root); e != nil {
		return zero, e
	}
	spec := dcrPackage + "@" + expected
	if action == app.UpdatePackage || expected == "absent" {
		spec = dcrPackage + "@latest"
	}
	if _, e := plainInfo(state); os.IsNotExist(e) {
		if e = os.Mkdir(state, 0700); e != nil {
			return zero, maintenanceError("UNSAFE")
		}
	}
	if e := stageRoots(root, state); e != nil {
		return zero, e
	}
	var id [16]byte
	if _, e := rand.Read(id[:]); e != nil {
		return zero, maintenanceError("UNSAFE")
	}
	stage := filepath.Join(state, "stage-"+hex.EncodeToString(id[:]))
	if e := os.Mkdir(stage, 0700); e != nil {
		return zero, maintenanceError("UNSAFE")
	}
	identity, e := plainInfo(stage)
	if e != nil {
		return zero, maintenanceError("UNSAFE")
	}
	// Preserve failed stages for explicit review instead of deleting uncertain files.
	if e = run(ctx, stage, spec); e != nil {
		return zero, maintenanceError("NPM")
	}
	if e = ctx.Err(); e != nil {
		return zero, e
	}
	now, e := plainInfo(stage)
	if e != nil || !os.SameFile(identity, now) {
		return zero, maintenanceError("UNSAFE")
	}
	v, e := validatePreparedPackage(stage)
	if e != nil {
		return zero, e
	}
	if action == app.RepairPackage && expected != "absent" && v != expected {
		return zero, maintenanceError("NPM")
	}
	if e = guard(ctx, root); e != nil {
		return zero, e
	}
	if e = ctx.Err(); e != nil {
		return zero, e
	}
	if e = stageRoots(root, state); e != nil {
		return zero, e
	}
	if e = stageOwnership(root); e != nil {
		return zero, e
	}
	if stageVersion(root) != expected {
		return zero, maintenanceError("STALE")
	}
	return packageStage{Stage: stage, Version: v, identity: identity}, nil
}

// This runner intentionally suppresses npm lifecycle scripts. It is for
// artifact acceptance only, not a fully functional upstream installation.
func runNativeArtifactStage(parent context.Context, stage, spec string) error {
	if !strings.HasPrefix(spec, dcrPackage+"@") {
		return maintenanceError("UNSAFE")
	}
	version := strings.TrimPrefix(spec, dcrPackage+"@")
	if version != "latest" && !versionPattern.MatchString(version) {
		return maintenanceError("UNSAFE")
	}
	if i, e := plainInfo(stage); e != nil || !i.IsDir() || !strings.HasPrefix(filepath.Base(stage), "stage-") || !recoveryName.MatchString(filepath.Base(stage)) {
		return maintenanceError("UNSAFE")
	}
	ctx, cancel := context.WithTimeout(parent, 3*time.Minute)
	defer cancel()
	if e := artifactNpm(ctx, stage, spec); e != nil {
		return maintenanceError("NPM")
	}
	if ctx.Err() != nil {
		return errors.New("npm staging timed out")
	}
	return nil
}
