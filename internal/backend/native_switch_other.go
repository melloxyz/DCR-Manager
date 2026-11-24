//go:build !windows

package backend

func movePackageNoReplace(string, string) error     { return maintenanceError("UNSAFE") }
func movePackageJournal(string, string, bool) error { return maintenanceError("UNSAFE") }
