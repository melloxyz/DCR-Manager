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
	"regexp"
	"sort"
	"strings"
	"time"
)

var recoveryName = regexp.MustCompile("^(package-backup|cache-backup|stage|restore-stage)-[a-f0-9]{32}$")
var errRecoveryLimit = errors.New("recovery scan limit")

type recoveryMeasure struct{ bytes, count int64 }

// Read directory batches to keep the entry cap meaningful even for wide trees.
func measureRecovery(ctx context.Context, root string) (recoveryMeasure, error) {
	var m recoveryMeasure
	stack := []string{root}
	for len(stack) > 0 {
		if ctx.Err() != nil {
			return m, errRecoveryLimit
		}
		path := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		info, e := plainInfo(path)
		if e != nil || !info.IsDir() {
			return m, errors.New("unsafe recovery directory")
		}
		f, e := os.Open(path)
		if e != nil {
			return m, e
		}
		opened, e := f.Stat()
		if e != nil || !os.SameFile(info, opened) {
			f.Close()
			return m, errors.New("directory changed")
		}
		for {
			if ctx.Err() != nil {
				f.Close()
				return m, errRecoveryLimit
			}
			batch, e := f.Readdir(128)
			for _, d := range batch {
				m.count++
				if m.count > 100000 || ctx.Err() != nil {
					f.Close()
					return m, errRecoveryLimit
				}
				if isReparse(d) || d.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
					f.Close()
					return m, errors.New("unsafe recovery entry")
				}
				if d.IsDir() {
					stack = append(stack, filepath.Join(path, d.Name()))
				} else if d.Mode().IsRegular() && d.Size() >= 0 && m.bytes <= 1<<62-d.Size() {
					m.bytes += d.Size()
				} else {
					f.Close()
					return m, errors.New("unsafe recovery file")
				}
			}
			if e == io.EOF {
				break
			}
			if e != nil {
				f.Close()
				return m, e
			}
		}
		f.Close()
	}
	return m, nil
}
func recoveryToken(path string, info os.FileInfo, m recoveryMeasure, version string, metadata []byte) (string, error) {
	created, ok := creationTicks(info)
	if !ok {
		return "", errors.New("creation time unavailable")
	}
	modified := info.ModTime().Unix()*10000000 + int64(info.ModTime().Nanosecond()/100) + 621355968000000000
	tokenPath, ok := legacyTokenPath(path, info)
	if !ok {
		return "", errors.New("canonical recovery path unavailable")
	}
	parts := []string{tokenPath, fmt.Sprint(created), fmt.Sprint(modified), fmt.Sprint(m.bytes), fmt.Sprint(m.count), version}
	for _, name := range []string{"package.json", "package-lock.json", filepath.Join("node_modules", "@wonderwhy-er", "desktop-commander", "package.json")} {
		raw, e := readPlainFile(filepath.Join(path, name), maxConfigBytes)
		if os.IsNotExist(e) {
			continue
		}
		if e != nil {
			return "", e
		}
		parts = append(parts, fmt.Sprintf("%X", sha256.Sum256(raw)))
	}
	if metadata != nil {
		parts = append(parts, fmt.Sprintf("%X", sha256.Sum256(metadata)))
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(parts, "|")))), nil
}
func cacheRestoreMetadata(path, cache string) ([]byte, error) {
	if !filepath.IsAbs(cache) {
		return nil, errors.New("cache unavailable")
	}
	parent := filepath.Join(cache, "_npx")
	info, e := plainInfo(parent)
	if e != nil || !info.IsDir() {
		return nil, errors.New("cache parent unavailable")
	}
	raw, e := readPlainFile(path+".json", maxConfigBytes)
	if e != nil {
		return nil, e
	}
	var meta struct{ OriginalPath, BackupPath string }
	if json.Unmarshal([]byte(strings.TrimPrefix(string(raw), "\uFEFF")), &meta) != nil || !strings.EqualFold(meta.BackupPath, path) || !filepath.IsAbs(meta.OriginalPath) {
		return nil, errors.New("invalid metadata")
	}
	target := filepath.Clean(meta.OriginalPath)
	if !strings.EqualFold(filepath.Dir(target), parent) || !cacheIDPattern.MatchString(filepath.Base(target)) || !strings.EqualFold(filepath.VolumeName(target), filepath.VolumeName(path)) {
		return nil, errors.New("invalid target")
	}
	if _, e = plainInfo(target); !os.IsNotExist(e) {
		return nil, errors.New("destination not empty or unsafe")
	}
	return raw, nil
}
func inspectRecovery(ctx context.Context, state, cache, name string, now time.Time) app.Backup {
	r := app.Backup{Name: name, Status: "unsafe", Kind: "stage"}
	if !recoveryName.MatchString(name) {
		return r
	}
	if strings.HasPrefix(name, "package-backup-") {
		r.Kind = "package"
	} else if strings.HasPrefix(name, "cache-backup-") {
		r.Kind = "cache"
	}
	if ctx.Err() != nil {
		r.Status = "not-assessed"
		return r
	}
	path := filepath.Join(state, name)
	info, e := plainInfo(path)
	if e != nil || !info.IsDir() {
		return r
	}
	r.AgeDays = int(max(0, now.Sub(info.ModTime()).Hours()/24))
	r.Review = r.AgeDays >= 30
	m, e := measureRecovery(ctx, path)
	if errors.Is(e, errRecoveryLimit) {
		r.Status = "not-assessed"
		return r
	}
	if e != nil {
		return r
	}
	r.Bytes = m.bytes
	r.SizeKnown = true
	if r.Kind == "stage" {
		r.Status = "unclassified-stage"
		raw, e := readPlainFile(filepath.Join(path, "transaction.json"), maxConfigBytes)
		if os.IsNotExist(e) {
			return r
		}
		var journal struct {
			Schema int
			State  string
		}
		if e != nil || json.Unmarshal([]byte(strings.TrimPrefix(string(raw), "\uFEFF")), &journal) != nil || journal.Schema != 1 {
			r.Status = "unsafe"
			return r
		}
		switch journal.State {
		case "switching", "completed", "rollback-incomplete", "switch-failed":
			r.Status = journal.State
		}
		return r
	}
	top, e := readPlainFile(filepath.Join(path, "package.json"), maxConfigBytes)
	if os.IsNotExist(e) {
		// A partial package without its ownership manifest is unsafe in the legacy inventory.
		if stageOwnership(path) != nil {
			return r
		}
		r.Status = "incomplete"
		return r
	}
	if e != nil || !exclusiveManifest(top) {
		return r
	}
	pkg := packageDir(filepath.Join(path, "node_modules"))
	manifest, e := readManifest(filepath.Join(pkg, "package.json"))
	if os.IsNotExist(e) {
		r.Status = "incomplete"
		return r
	}
	if e != nil {
		return r
	}
	r.Version = manifest.Version
	r.Status = "incomplete"
	entry, e := plainInfo(filepath.Join(pkg, "dist", "index.js"))
	if e != nil || !entry.Mode().IsRegular() {
		return r
	}
	var metadata []byte
	if r.Kind == "package" {
		var deps struct{ Dependencies map[string]string }
		if json.Unmarshal([]byte(strings.TrimPrefix(string(top), "\uFEFF")), &deps) != nil || deps.Dependencies[dcrPackage] != r.Version {
			return r
		}
		if lock, e := plainInfo(filepath.Join(path, "package-lock.json")); e != nil || !lock.Mode().IsRegular() {
			return r
		}
	} else {
		metadata, e = cacheRestoreMetadata(path, cache)
		if e != nil {
			r.Status = "destination-blocked"
			return r
		}
	}
	if ctx.Err() != nil {
		r.Status = "not-assessed"
		return r
	}
	token, e := recoveryToken(path, info, m, r.Version, metadata)
	if e != nil {
		r.Status = "unsafe"
		return r
	}
	if ctx.Err() != nil {
		r.Status = "not-assessed"
		return r
	}
	r.Status = "ready"
	r.Token = token
	return r
}
func nativeRecovery(parent context.Context, state, cache string) ([]app.Backup, bool) {
	rows := []app.Backup{}
	if !filepath.IsAbs(state) || parent.Err() != nil {
		return rows, false
	}
	ctx, cancel := context.WithTimeout(parent, 8*time.Second)
	defer cancel()
	info, e := plainInfo(state)
	if os.IsNotExist(e) {
		return rows, true
	}
	if e != nil || !info.IsDir() {
		return rows, false
	}
	f, e := os.Open(state)
	if e != nil {
		return rows, false
	}
	defer f.Close()
	opened, e := f.Stat()
	if e != nil || !os.SameFile(info, opened) {
		return rows, false
	}
	var entries []os.FileInfo
	count := 0
	for {
		if ctx.Err() != nil {
			return rows, false
		}
		batch, e := f.Readdir(128)
		for _, d := range batch {
			count++
			if count > 10000 {
				return rows, false
			}
			if recoveryName.MatchString(d.Name()) && (d.IsDir() || isReparse(d)) {
				entries = append(entries, d)
			}
		}
		if e == io.EOF {
			break
		}
		if e != nil {
			return rows, false
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].ModTime().Equal(entries[j].ModTime()) {
			return entries[i].Name() < entries[j].Name()
		}
		return entries[i].ModTime().After(entries[j].ModTime())
	})
	now := time.Now()
	for _, entry := range entries {
		rows = append(rows, inspectRecovery(ctx, state, cache, entry.Name(), now))
	}
	return rows, parent.Err() == nil
}

// Keep the independent legacy mutation engine authoritative while migration is
// ongoing. An unverified ready row must never carry a usable restore token.
func applyRecoveryParity(s *app.Snapshot, rows []app.Backup, known, sameCache bool) {
	old := map[string]app.Backup{}
	for _, b := range s.Backups {
		old[b.Name] = b
	}
	s.RecoveryKnown = s.RecoveryKnown && known && len(old) == len(rows)
	for i := range rows {
		b := &rows[i]
		prior, ok := old[b.Name]
		if !ok {
			s.RecoveryKnown = false
		}
		if b.Status != "ready" {
			b.Token = ""
			continue
		}
		if b.Kind == "cache" && !sameCache {
			b.Status = "destination-blocked"
			b.Token = ""
			continue
		}
		if !s.RecoveryKnown || !ok || prior.Kind != b.Kind || prior.Status != "ready" || prior.Token != b.Token || prior.Version != b.Version || prior.Bytes != b.Bytes || !prior.SizeKnown || !b.SizeKnown {
			b.Status = "not-assessed"
			b.Token = ""
		}
	}
	if !s.RecoveryKnown {
		for i := range rows {
			if rows[i].Status == "ready" {
				rows[i].Status = "not-assessed"
				rows[i].Token = ""
			}
		}
	}
	s.Backups = rows
}
