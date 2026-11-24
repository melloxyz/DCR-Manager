package backend

import (
	"dcr-manager/internal/app"
	"github.com/pelletier/go-toml/v2"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

func auditTOML(name, p string) app.Client {
	bad := app.Client{Client: name, Status: "unreadable-or-unsupported"}
	raw, e := readPlainFile(p, maxConfigBytes)
	if os.IsNotExist(e) {
		return app.Client{Client: name, Status: "not-found"}
	}
	if e != nil || !utf8.Valid(raw) {
		return bad
	}
	var cfg map[string]any
	if e = toml.Unmarshal(raw, &cfg); e != nil {
		return bad
	}
	r := app.Client{Client: name, Status: "checked"}
	data, exists := cfg["mcp_servers"]
	if !exists {
		return r
	}
	servers, ok := data.(map[string]any)
	if !ok {
		return bad
	}
	modes := map[string]bool{}
	for _, value := range servers {
		v, ok := value.(map[string]any)
		if !ok {
			continue
		}
		command := ""
		if item, exists := v["command"]; exists {
			var ok bool
			command, ok = item.(string)
			if !ok {
				continue
			}
		}
		var args []any
		if item, exists := v["args"]; exists {
			var ok bool
			args, ok = item.([]any)
			if !ok {
				continue
			}
		}
		match, floating := isDCRArgument(command), false
		for _, item := range args {
			arg, ok := item.(string)
			if !ok {
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
		mode := "other"
		switch strings.ToLower(filepath.Base(strings.ReplaceAll(command, "\\", "/"))) {
		case "npx", "npx.cmd", "npx.exe":
			mode = "npx"
		case "node", "node.exe":
			mode = "direct"
		}
		modes[mode] = true
	}
	var names []string
	for m := range modes {
		names = append(names, m)
	}
	sort.Strings(names)
	r.Modes = strings.Join(names, ", ")
	return r
}
func nativeClientAudits(home, roaming string) []app.Client {
	out := jsonClientAudits(home, roaming)
	if !filepath.IsAbs(home) {
		return append(out, app.Client{Client: "Codex", Status: "unreadable-or-unsupported"})
	}
	return append(out, auditTOML("Codex", filepath.Join(home, ".codex", "config.toml")))
}
