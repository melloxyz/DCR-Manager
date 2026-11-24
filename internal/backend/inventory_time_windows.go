//go:build windows

package backend

import (
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"strings"
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

// PowerShell DirectoryInfo.FullName expands Windows 8.3 aliases. Use the same
// long path for legacy tokens, and verify that it still identifies this item.
func legacyTokenPath(p string, info os.FileInfo) (string, bool) {
	if !filepath.IsAbs(p) {
		return "", false
	}
	ptr, e := windows.UTF16PtrFromString(p)
	if e != nil {
		return "", false
	}
	buf := make([]uint16, 32768)
	n, e := windows.GetLongPathName(ptr, &buf[0], uint32(len(buf)))
	if e != nil || n == 0 || n >= uint32(len(buf)) {
		return "", false
	}
	full := windows.UTF16ToString(buf[:n])
	if !filepath.IsAbs(full) {
		return "", false
	}
	actual, e := plainInfo(full)
	if e != nil || !os.SameFile(info, actual) {
		return "", false
	}
	return strings.ToLower(filepath.Clean(full)), true
}
