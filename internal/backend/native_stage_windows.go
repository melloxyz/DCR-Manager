//go:build windows

package backend

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Invoke the standard Node.js npm CLI directly; avoid constructing shell
// commands from paths. Alternate launchers are not yet supported.
func artifactNpm(ctx context.Context, stage, spec string) error {
	npm, e := runtimePath("npm.cmd")
	if e != nil {
		return e
	}
	node, e := runtimePath("node.exe")
	if e != nil {
		return e
	}
	if !strings.EqualFold(filepath.Dir(npm), filepath.Dir(node)) {
		return maintenanceError("UNSAFE")
	}
	cli := filepath.Join(filepath.Dir(npm), "node_modules", "npm", "bin", "npm-cli.js")
	for _, p := range []string{node, npm, cli} {
		i, e := plainInfo(p)
		if e != nil || !i.Mode().IsRegular() {
			return maintenanceError("UNSAFE")
		}
	}
	cmd := exec.CommandContext(ctx, node, cli, "install", "--prefix", stage, spec, "--cache", filepath.Join(stage, "npm-cache"), "--userconfig", filepath.Join(stage, "user.npmrc"), "--globalconfig", filepath.Join(stage, "global.npmrc"), "--save-exact", "--global=false", "--ignore-scripts", "--no-audit", "--no-fund", "--fetch-retries", "1", "--fetch-timeout", "30000")
	cmd.Dir = stage
	cmd.WaitDelay = time.Second
	configure(cmd)
	cmd.Env = []string{}
	for _, v := range os.Environ() {
		key := strings.ToLower(strings.SplitN(v, "=", 2)[0])
		if strings.HasPrefix(key, "npm_config_") || key == "node_options" || key == "node_path" {
			continue
		}
		cmd.Env = append(cmd.Env, v)
	}
	// Discard raw npm output; no credentials or private paths enter diagnostics.
	return cmd.Run()
}
