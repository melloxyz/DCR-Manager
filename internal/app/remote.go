package app

import "context"

type RemoteAction string

const (
	Start   RemoteAction = "start"
	Stop    RemoteAction = "stop"
	Restart RemoteAction = "restart"
)

func (a RemoteAction) Valid() bool { return a == Start || a == Stop || a == Restart }

type RemoteOperator func(context.Context, RemoteAction, []int) error
