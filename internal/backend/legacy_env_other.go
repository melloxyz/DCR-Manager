//go:build !windows

package backend

import "os"

func legacyPowerShellEnv() []string { return os.Environ() }
