//go:build !windows

package backend

import "context"

func artifactNpm(context.Context, string, string) error { return maintenanceError("UNSAFE") }
