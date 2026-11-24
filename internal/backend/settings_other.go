//go:build !windows

package backend

import "errors"

func readStartup() (string, error)           { return "unknown", errors.New("Windows required") }
func changeStartup(string, string) error     { return errors.New("Windows required") }
func withSettingsLock(fn func() error) error { return errors.New("Windows required") }
