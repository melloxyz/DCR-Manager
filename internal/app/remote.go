package app

import "context"

type RemoteAction string

const (
	Start   RemoteAction = "start"
	Stop    RemoteAction = "stop"
	Restart RemoteAction = "restart"
)

func (a RemoteAction) Valid() bool { return a == Start || a == Stop || a == Restart }

// Created is the Windows creation FILETIME in 100ns ticks.
type ProcessIdentity struct {
	PID     int
	Created uint64
}
type RemoteOperator func(context.Context, RemoteAction, []ProcessIdentity) error
