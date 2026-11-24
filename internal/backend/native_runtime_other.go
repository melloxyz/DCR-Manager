//go:build !windows

package backend

import (
	"context"
	"errors"
	"os/exec"
)

func npmVersionCommand(context.Context, string) (*exec.Cmd, error) {
	return nil, errors.New("Windows required")
}
