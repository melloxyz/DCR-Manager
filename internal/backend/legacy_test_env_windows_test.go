//go:build windows

package backend

import (
	"os"
	"strings"
)

// Windows PowerShell must initialize its own module search path rather than
// inherit a PowerShell 7 runner's path. This affects test children only.
func legacyTestEnv() []string {
	out := []string{}
	for _, v := range os.Environ() {
		if !strings.HasPrefix(strings.ToLower(v), "psmodulepath=") {
			out = append(out, v)
		}
	}
	return out
}
