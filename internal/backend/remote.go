package backend

import (
	"context"
	"dcr-manager/internal/app"
)

func Operate(root string) app.RemoteOperator {
	return func(ctx context.Context, a app.RemoteAction, ids []app.ProcessIdentity) error {
		if err := validateAction(a, ids); err != nil {
			return err
		}
		return operateNative(ctx, a, ids)
	}
}
