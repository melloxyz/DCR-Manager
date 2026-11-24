//go:build windows

package backend

import "golang.org/x/sys/windows"

func movePackageNoReplace(from, to string) error {
	return movePackageWindows(from, to, windows.MOVEFILE_WRITE_THROUGH)
}
func movePackageJournal(from, to string, replace bool) error {
	flags := uint32(windows.MOVEFILE_WRITE_THROUGH)
	if replace {
		flags |= windows.MOVEFILE_REPLACE_EXISTING
	}
	return movePackageWindows(from, to, flags)
}
func movePackageWindows(from, to string, flags uint32) error {
	a, e := windows.UTF16PtrFromString(from)
	if e != nil {
		return e
	}
	b, e := windows.UTF16PtrFromString(to)
	if e != nil {
		return e
	}
	return windows.MoveFileEx(a, b, flags)
}
