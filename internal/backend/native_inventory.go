package backend

import (
	"context"
	"crypto/sha256"
	"dcr-manager/internal/app"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const dcrPackage = "@wonderwhy-er/desktop-commander"

type installInventory struct {
	version               string
	installed, cacheKnown bool
	installations         []app.Installation
	caches                []app.CacheCandidate
}
type packageManifest struct {
	Name, Version string
	Engines       struct{ Node string }
}

func readManifest(path string) (packageManifest, error) {
	var m packageManifest
	raw, e := readPlainFile(path, maxConfigBytes)
	if e != nil {
		return m, e
	}
	if !utf8.Valid(raw) || json.Unmarshal([]byte(strings.TrimPrefix(string(raw), "\uFEFF")), &m) != nil || m.Name != dcrPackage || len(m.Version) > 128 || !versionPattern.MatchString(m.Version) {
		return m, errors.New("invalid manifest")
	}
	if len(m.Engines.Node) > 256 || strings.IndexFunc(m.Engines.Node, unicode.IsControl) >= 0 {
		m.Engines.Node = ""
	}
	return m, nil
}
func packageDir(nodeModules string) string {
	return filepath.Join(nodeModules, "@wonderwhy-er", "desktop-commander")
}
func inspectInstallation(kind, nodeModules string) (app.Installation, string, bool) {
	row := app.Installation{Kind: kind, Version: "unreadable"}
	if !filepath.IsAbs(nodeModules) {
		return row, "unknown", false
	}
	pkg := packageDir(nodeModules)
	m, e := readManifest(filepath.Join(pkg, "package.json"))
	if os.IsNotExist(e) {
		return row, "absent", false
	}
	if e != nil {
		return row, "unknown", true
	}
	row.Version = m.Version
	row.NodeRange = m.Engines.Node
	if info, e := plainInfo(filepath.Join(pkg, "dist", "index.js")); e == nil && info.Mode().IsRegular() {
		row.EntryExists = true
	}
	return row, m.Version, true
}

// Only exclusive DCR caches can become maintenance targets. Extra
// dependency/workspace declarations are conservatively rejected.
func exclusiveManifest(raw []byte) bool {
	var m struct {
		Dependencies                                      map[string]string
		DevDependencies, OptionalDependencies, Workspaces json.RawMessage
	}
	if !utf8.Valid(raw) || json.Unmarshal([]byte(strings.TrimPrefix(string(raw), "\uFEFF")), &m) != nil || len(m.Dependencies) != 1 || m.Dependencies[dcrPackage] == "" {
		return false
	}
	for _, extra := range []json.RawMessage{m.DevDependencies, m.OptionalDependencies, m.Workspaces} {
		switch strings.TrimSpace(string(extra)) {
		case "", "null":
		default:
			return false
		}
	}
	return true
}
func cacheCandidate(dir string, info os.FileInfo, version string, manifest []byte) (app.CacheCandidate, bool) {
	id := filepath.Base(dir)
	if !cacheIDPattern.MatchString(id) || !versionPattern.MatchString(version) {
		return app.CacheCandidate{}, false
	}
	ticks, ok := creationTicks(info)
	if !ok {
		return app.CacheCandidate{}, false
	}
	// Match the PowerShell T6 token while its mutation engine is still in use.
	tokenPath, ok := legacyTokenPath(dir, info)
	if !ok {
		return app.CacheCandidate{}, false
	}
	identity := fmt.Sprintf("%s|%d|%X|%s", tokenPath, ticks, sha256.Sum256(manifest), version)
	return app.CacheCandidate{ID: id, Version: version, Token: fmt.Sprintf("%x", sha256.Sum256([]byte(identity)))}, true
}
func scanInstallations(parent context.Context, local, global, cache string) installInventory {
	inv := installInventory{version: "unknown", installations: []app.Installation{}, caches: []app.CacheCandidate{}}
	if parent.Err() != nil {
		return inv
	}
	if filepath.IsAbs(local) {
		row, version, exists := inspectInstallation("dedicated", filepath.Join(local, "DesktopCommander", "node_modules"))
		inv.version = version
		inv.installed = exists && row.EntryExists && version != "unknown"
		if exists {
			inv.installations = append(inv.installations, row)
		}
	}
	if filepath.IsAbs(global) {
		if row, _, exists := inspectInstallation("global", global); exists {
			inv.installations = append(inv.installations, row)
		}
	}
	if !filepath.IsAbs(cache) {
		return inv
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	root := filepath.Join(cache, "_npx")
	// An absent cache is empty, not a failed read. A linked or inaccessible ancestor is unknown.
	info, e := plainInfo(root)
	if os.IsNotExist(e) {
		inv.cacheKnown = ctx.Err() == nil
		return inv
	}
	if e != nil || !info.IsDir() {
		return inv
	}
	f, e := os.Open(root)
	if e != nil {
		return inv
	}
	defer f.Close()
	opened, e := f.Stat()
	if e != nil || !os.SameFile(info, opened) {
		return inv
	}
	inv.cacheKnown = true
	count := 0
	for {
		if ctx.Err() != nil {
			inv.cacheKnown = false
			break
		}
		batch, e := f.Readdir(128)
		for _, d := range batch {
			count++
			if count > 10000 || ctx.Err() != nil {
				inv.cacheKnown = false
				break
			}
			if !d.IsDir() || !cacheIDPattern.MatchString(d.Name()) {
				continue
			}
			dir := filepath.Join(root, d.Name())
			current, err := plainInfo(dir)
			if err != nil || !current.IsDir() || !os.SameFile(d, current) {
				inv.cacheKnown = false
				continue
			}
			raw, err := readPlainFile(filepath.Join(dir, "package.json"), maxConfigBytes)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				inv.cacheKnown = false
				continue
			}
			if !exclusiveManifest(raw) {
				continue
			}
			row, v, exists := inspectInstallation("npx", filepath.Join(dir, "node_modules"))
			if !exists {
				continue
			}
			inv.installations = append(inv.installations, row)
			if v == "unknown" || v == "absent" {
				continue
			}
			if c, ok := cacheCandidate(dir, current, v, raw); ok {
				inv.caches = append(inv.caches, c)
			} else {
				inv.cacheKnown = false
			}
		}
		if !inv.cacheKnown && (count > 10000 || ctx.Err() != nil) {
			break
		}
		if e == io.EOF {
			break
		}
		if e != nil {
			inv.cacheKnown = false
			break
		}
	}
	sort.Slice(inv.caches, func(i, j int) bool { return inv.caches[i].ID < inv.caches[j].ID })
	sort.SliceStable(inv.installations, func(i, j int) bool {
		a, b := inv.installations[i], inv.installations[j]
		return a.Kind == "npx" && b.Kind == "npx" && a.Version < b.Version
	})
	if ctx.Err() != nil {
		inv.cacheKnown = false
	}
	return inv
}
func applyCacheParity(s *app.Snapshot, inv installInventory, sameRoot bool) {
	old := map[string]app.CacheCandidate{}
	for _, c := range s.Caches {
		old[c.ID] = c
	}
	known := sameRoot && s.Status.CacheKnown && inv.cacheKnown && len(inv.caches) == s.Status.CacheCandidates && len(old) == len(inv.caches)
	for _, c := range inv.caches {
		if old[c.ID] != c {
			known = false
		}
	}
	s.Status.CacheKnown = known
	s.Status.CacheCandidates = len(inv.caches)
	s.Caches = nil
	if known {
		s.Caches = inv.caches
	}
}
func npmDirectory(ctx context.Context, query string) (string, error) {
	if query != "cache" && query != "global" {
		return "", errors.New("unsupported npm query")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	p, e := runtimePath("npm.cmd")
	if e != nil {
		return "", e
	}
	cmd, e := npmDirectoryCommand(ctx, p, query)
	if e != nil {
		return "", e
	}
	// Inherit the same current directory as the legacy maintenance bridge.
	cmd.WaitDelay = time.Second
	var out boundedOutput
	cmd.Stdout = &out
	if e = cmd.Run(); e != nil || out.overflow || ctx.Err() != nil {
		return "", errors.New("npm directory query failed")
	}
	value := strings.TrimSpace(string(out.buf))
	if !utf8.ValidString(value) || !filepath.IsAbs(value) || strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return "", errors.New("invalid npm directory")
	}
	return filepath.Clean(value), nil
}
