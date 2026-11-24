package backend

import (
	"context"
	"dcr-manager/internal/app"
	"errors"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var errRuntimeMissing = errors.New("runtime missing")
var runtimeToken = regexp.MustCompile("^v?[0-9]+\\.[0-9]+\\.[0-9]+(?:[-+][0-9A-Za-z.+-]+)?$")
var exactNode = regexp.MustCompile("^v?([0-9]+)\\.([0-9]+)\\.([0-9]+)$")
var minimumNode = regexp.MustCompile("^>=\\s*([0-9]+)(?:\\.([0-9]+))?(?:\\.([0-9]+))?$")

func versionTriple(parts []string) ([3]uint64, bool) {
	var v [3]uint64
	if len(parts) != 4 {
		return v, false
	}
	for i := range v {
		if parts[i+1] == "" {
			continue
		}
		n, e := strconv.ParseUint(parts[i+1], 10, 32)
		if e != nil {
			return v, false
		}
		v[i] = n
	}
	return v, true
}
func assessNode(actual, requirement string) string {
	a, ok := versionTriple(exactNode.FindStringSubmatch(actual))
	if !ok {
		return "unknown"
	}
	b, ok := versionTriple(minimumNode.FindStringSubmatch(requirement))
	if !ok {
		return "unknown"
	}
	for i := range a {
		if a[i] < b[i] {
			return "below-minimum"
		}
		if a[i] > b[i] {
			return "meets-minimum"
		}
	}
	return "meets-minimum"
}

type versionRunner func(context.Context, string) (string, error)

func applyRuntime(ctx context.Context, s *app.Snapshot, run versionRunner) {
	probe := func(name string) (string, bool) {
		if ctx.Err() != nil {
			return "unknown", false
		}
		c, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		v, e := run(c, name)
		if errors.Is(e, errRuntimeMissing) {
			return "missing", false
		}
		v = strings.TrimSpace(v)
		if e != nil || c.Err() != nil || len(v) > 128 || !runtimeToken.MatchString(v) {
			return "unknown", false
		}
		return v, true
	}
	var r app.Runtime
	r.Node, s.Status.NodeAvailable = probe("node.exe")
	r.Npm, s.Status.NpmAvailable = probe("npm.cmd")
	for _, i := range s.Installations {
		if i.Kind == "dedicated" {
			r.DeclaredNodeRange = i.NodeRange
			break
		}
	}
	r.Assessment = assessNode(r.Node, r.DeclaredNodeRange)
	s.Runtime = r
}

// Resolve only known tool names from PATH, never an implicit working-directory executable.
func runtimePath(name string) (string, error) {
	if name != "node.exe" && name != "npm.cmd" {
		return "", errors.New("unsupported runtime")
	}
	p, e := exec.LookPath(name)
	if errors.Is(e, exec.ErrNotFound) {
		return "", errRuntimeMissing
	}
	if e != nil || !filepath.IsAbs(p) {
		return "", errors.New("untrusted runtime path")
	}
	return p, nil
}

type boundedOutput struct {
	buf      []byte
	overflow bool
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	n := 4096 - len(b.buf)
	if len(p) > n {
		b.overflow = true
	}
	if n > len(p) {
		n = len(p)
	}
	b.buf = append(b.buf, p[:n]...)
	return len(p), nil
}
func runRuntimeVersion(ctx context.Context, name string) (string, error) {
	p, e := runtimePath(name)
	if e != nil {
		return "", e
	}
	var cmd *exec.Cmd
	if name == "npm.cmd" {
		cmd, e = npmVersionCommand(ctx, p)
		if e != nil {
			return "", e
		}
	} else {
		cmd = exec.CommandContext(ctx, p, "--version")
		configure(cmd)
	}
	// Version query does not need the caller's project directory.
	cmd.Dir = filepath.Dir(p)
	cmd.WaitDelay = time.Second
	var out boundedOutput
	cmd.Stdout = &out
	if e = cmd.Run(); e != nil {
		return "", e
	}
	if out.overflow {
		return "", errors.New("version output exceeded limit")
	}
	return string(out.buf), nil
}
