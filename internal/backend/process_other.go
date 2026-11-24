//go:build !windows

package backend

import "os/exec"

func configure(cmd *exec.Cmd) {}
