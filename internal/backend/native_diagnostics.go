package backend

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"dcr-manager/internal/app"
)

const maxLogBytes = 2 * 1024 * 1024
const maxConfigBytes = 5 * 1024 * 1024
const maxLogDirectoryEntries = 10000

// Inspect ancestors before descendants; never intentionally follow a reparse point.
// This is a read-only best-effort check, not a sandbox against external filesystem races.
func plainInfo(p string) (os.FileInfo, error) {
	if !filepath.IsAbs(p) {
		return nil, errors.New("absolute path required")
	}
	p = filepath.Clean(p)
	if parent := filepath.Dir(p); parent != p {
		info, e := plainInfo(parent)
		if e != nil {
			return nil, e
		}
		if !info.IsDir() {
			return nil, errors.New("invalid parent")
		}
	}
	info, e := os.Lstat(p)
	if e != nil {
		return nil, e
	}
	if isReparse(info) || info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
		return nil, errors.New("unsupported path")
	}
	return info, nil
}
func readPlainFile(p string, limit int64) ([]byte, error) {
	info, e := plainInfo(p)
	if e != nil {
		return nil, e
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, errors.New("unsupported file")
	}
	f, e := os.Open(p)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	opened, e := f.Stat()
	if e != nil {
		return nil, e
	}
	if !opened.Mode().IsRegular() || !os.SameFile(info, opened) || opened.Size() > limit {
		return nil, errors.New("file changed")
	}
	raw, e := io.ReadAll(io.LimitReader(f, limit+1))
	if e != nil {
		return nil, e
	}
	if int64(len(raw)) > limit {
		return nil, errors.New("file exceeded limit")
	}
	return raw, nil
}

var logName = regexp.MustCompile("^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}_[0-9]{2}_[0-9]{2}_[0-9]{3}Z-debug-[0-9]+\\.log$")
var logDCR = regexp.MustCompile("(?i)@wonderwhy-er[/\\\\]desktop-commander")
var logCode = regexp.MustCompile("\\b(EBUSY|EPERM|EACCES|ENOTFOUND|ETIMEDOUT|ECONNRESET)\\b")

func nativeLogs(parent context.Context, root string) ([]app.Log, bool) {
	rows := []app.Log{}
	if parent.Err() != nil || !filepath.IsAbs(root) {
		return rows, false
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	dir := filepath.Join(root, "_logs")
	info, e := plainInfo(dir)
	if os.IsNotExist(e) {
		return rows, true
	}
	if e != nil || !info.IsDir() {
		return rows, false
	}
	f, e := os.Open(dir)
	if e != nil {
		return rows, false
	}
	defer f.Close()
	opened, e := f.Stat()
	if e != nil || !os.SameFile(info, opened) {
		return rows, false
	}
	known := true
	var files []os.FileInfo
	count := 0
	for {
		if ctx.Err() != nil {
			known = false
			break
		}
		batch, e := f.Readdir(256)
		for _, entry := range batch {
			count++
			if count > maxLogDirectoryEntries {
				known = false
				break
			}
			if logName.MatchString(entry.Name()) {
				files = append(files, entry)
			}
		}
		if count > maxLogDirectoryEntries {
			break
		}
		if e == io.EOF {
			break
		}
		if e != nil {
			known = false
			break
		}
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].ModTime().Equal(files[j].ModTime()) {
			return files[i].Name() > files[j].Name()
		}
		return files[i].ModTime().After(files[j].ModTime())
	})
	if len(files) > 100 {
		files = files[:100]
	}
	for _, entry := range files {
		if ctx.Err() != nil {
			return rows, false
		}
		raw, e := readPlainFile(filepath.Join(dir, entry.Name()), maxLogBytes)
		if e != nil {
			known = false
			continue
		}
		if !logDCR.Match(raw) {
			continue
		}
		set := map[string]bool{}
		for _, code := range logCode.FindAll(raw, -1) {
			set[string(code)] = true
		}
		codes := make([]string, 0, len(set))
		for code := range set {
			codes = append(codes, code)
		}
		sort.Strings(codes)
		rows = append(rows, app.Log{Name: entry.Name(), Codes: codes, ModifiedUTC: entry.ModTime().UTC().Format("2006-01-02T15:04:05Z")})
		if len(rows) == 20 {
			break
		}
	}
	return rows, known && ctx.Err() == nil
}

var dcrArgument = regexp.MustCompile("(?i)^@wonderwhy-er/desktop-commander(?:@[^@\\s]+)?$")
var dcrPath = regexp.MustCompile("(?i)[/\\\\]+@wonderwhy-er[/\\\\]+desktop-commander[/\\\\]+")
var floatingArgument = regexp.MustCompile("(?i)^@wonderwhy-er/desktop-commander(?:@latest)?$")

func isDCRArgument(v string) bool { return dcrArgument.MatchString(v) || dcrPath.MatchString(v) }
func auditJSON(name, p string) app.Client {
	bad := app.Client{Client: name, Status: "unreadable-or-unsupported"}
	raw, e := readPlainFile(p, maxConfigBytes)
	if os.IsNotExist(e) {
		return app.Client{Client: name, Status: "not-found"}
	}
	if e != nil {
		return bad
	}
	raw = []byte(strings.TrimPrefix(string(raw), "\uFEFF"))
	if !utf8.Valid(raw) {
		return bad
	}
	var cfg map[string]json.RawMessage
	if json.Unmarshal(raw, &cfg) != nil || cfg == nil {
		return bad
	}
	r := app.Client{Client: name, Status: "checked"}
	modes := map[string]bool{}
	for _, key := range []string{"mcpServers", "servers"} {
		data, ok := cfg[key]
		if !ok {
			continue
		}
		var servers map[string]json.RawMessage
		if json.Unmarshal(data, &servers) != nil || servers == nil {
			return bad
		}
		for _, data := range servers {
			var v struct {
				Command string
				Args    []json.RawMessage
			}
			if json.Unmarshal(data, &v) != nil {
				continue
			}
			match, floating := isDCRArgument(v.Command), false
			for _, item := range v.Args {
				var arg string
				if json.Unmarshal(item, &arg) != nil {
					continue
				}
				match = match || isDCRArgument(arg)
				floating = floating || floatingArgument.MatchString(arg)
			}
			if !match {
				continue
			}
			r.Entries++
			if floating {
				r.Floating++
			}
			command := strings.ToLower(filepath.Base(strings.ReplaceAll(v.Command, "\\", "/")))
			mode := "other"
			switch command {
			case "npx", "npx.cmd", "npx.exe":
				mode = "npx"
			case "node", "node.exe":
				mode = "direct"
			}
			modes[mode] = true
		}
	}
	var names []string
	for m := range modes {
		names = append(names, m)
	}
	sort.Strings(names)
	r.Modes = strings.Join(names, ", ")
	return r
}
func jsonClientAudits(home, roaming string) []app.Client {
	locations := []struct{ name, root, rel string }{
		{"Claude Desktop", roaming, "Claude/claude_desktop_config.json"},
		{"Claude Code", home, ".claude.json"},
		{"Cursor", home, ".cursor/mcp.json"},
		{"Windsurf", home, ".codeium/windsurf/mcp_config.json"},
		{"Antigravity", home, ".gemini/antigravity/mcp_config.json"},
		{"VS Code", roaming, "Code/User/mcp.json"},
	}
	out := make([]app.Client, 0, len(locations))
	for _, c := range locations {
		if !filepath.IsAbs(c.root) {
			out = append(out, app.Client{Client: c.name, Status: "unreadable-or-unsupported"})
			continue
		}
		out = append(out, auditJSON(c.name, filepath.Join(c.root, c.rel)))
	}
	return out
}
