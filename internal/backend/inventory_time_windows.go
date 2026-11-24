//go:build windows

package backend

import (
	"os"
	"syscall"
)

func creationTicks(info os.FileInfo) (uint64, bool) {
	a, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return 0, false
	}
	f := a.CreationTime
	ticks := (uint64(f.HighDateTime) << 32) | uint64(f.LowDateTime)
	if ticks == 0 {
		return 0, false
	}
	return ticks + 504911232000000000, true
}
