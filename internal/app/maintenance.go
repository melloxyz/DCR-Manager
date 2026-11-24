package app

import "context"

type MaintenanceAction string

const (
	RepairPackage   MaintenanceAction = "repair"
	UpdatePackage   MaintenanceAction = "update"
	QuarantineCache MaintenanceAction = "quarantine"
	RestoreBackup   MaintenanceAction = "restore"
)

type CacheCandidate struct{ ID, Version, Token string }
type MaintenanceRequest struct {
	Action                            MaintenanceAction
	Expected, CacheID, CurrentVersion string
}
type Maintainer func(context.Context, MaintenanceRequest) error
