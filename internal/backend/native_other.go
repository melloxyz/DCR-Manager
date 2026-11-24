//go:build !windows

package backend

import (
	"context"
	"dcr-manager/internal/app"
	"errors"
)

func nativeRows() ([]nativeCandidate, error) { return nil, errors.New("Windows process API required") }
func operateNative(context.Context, app.RemoteAction, []app.ProcessIdentity) error {
	return errors.New("Windows process API required")
}
