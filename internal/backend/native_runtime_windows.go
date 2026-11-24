//go:build windows

package backend

import (
	"context"
	"errors"
	"golang.org/x/sys/windows"
	"os/exec"
	"path/filepath"
	"strings"
)

func npmVersionCommand(ctx context.Context, p string) (*exec.Cmd, error) {
	// Batch scripts need cmd.exe. Only a fixed --version argument is permitted.
	// Explicit raw quoting is needed because cmd.exe uses different parsing rules.
	if !filepath.IsAbs(p) || !strings.EqualFold(filepath.Base(p), "npm.cmd") || strings.ContainsAny(p, "%!^&|<>\"\r\n\x00") {
		return nil, errors.New("unsupported npm launcher path")
	}
	dir, e := windows.GetSystemDirectory()
	if e != nil {
		return nil, e
	}
	exe := filepath.Join(dir, "cmd.exe")
	cmd := exec.CommandContext(ctx, exe)
	configure(cmd)
	cmd.SysProcAttr.CmdLine = "\"" + exe + "\" /d /v:off /s /c \"\"" + p + "\" --version\""
	return cmd, nil
}
