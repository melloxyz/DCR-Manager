package backend

import (
	"dcr-manager/internal/app"
	"strings"
	"testing"
)

func TestMaintenanceRequestValidation(t *testing.T) {
	token := strings.Repeat("a", 64)
	good := []app.MaintenanceRequest{
		{Action: app.RepairPackage, Expected: "0.3.4"},
		{Action: app.RepairPackage, Expected: "absent"},
		{Action: app.UpdatePackage, Expected: "unknown"},
		{Action: app.QuarantineCache, Expected: token, CacheID: "4b4c857f6efdfb61"},
	}
	for _, r := range good {
		if _, err := maintenanceArgs(r); err != nil {
			t.Fatal(r.Action, err)
		}
	}
	bad := []app.MaintenanceRequest{
		{Action: "delete", Expected: "0.3.4"},
		{Action: app.RepairPackage, Expected: "unknown"},
		{Action: app.UpdatePackage, Expected: "--force"},
		{Action: app.QuarantineCache, Expected: token, CacheID: "../other"},
		{Action: app.QuarantineCache, Expected: "invalid", CacheID: "4b4c857f6efdfb61"},
	}
	for _, r := range bad {
		if _, err := maintenanceArgs(r); err == nil {
			t.Fatal("accepted invalid request", r)
		}
	}
}
func TestMaintenanceErrorsAreSanitized(t *testing.T) {
	for _, code := range []string{"EBUSY", "EPERM", "EACCES"} {
		if !strings.Contains(maintenanceError(code).Error(), code) {
			t.Fatal("lost code", code)
		}
	}
	if strings.Contains(maintenanceError("private raw npm output").Error(), "private") {
		t.Fatal("raw output exposed")
	}
}

func TestRecoveryRequestValidation(t *testing.T) {
	token := strings.Repeat("a", 64)
	good := app.MaintenanceRequest{Action: app.RestoreBackup, Expected: token, CacheID: "package-backup-" + strings.Repeat("a", 32), CurrentVersion: "0.3.4"}
	args, err := maintenanceArgs(good)
	if err != nil || !strings.Contains(strings.Join(args, " "), "-CurrentVersion 0.3.4") {
		t.Fatal(args, err)
	}
	good.CurrentVersion = "--force"
	if _, err = maintenanceArgs(good); err == nil {
		t.Fatal("invalid current version accepted")
	}
	good.CacheID = "stage-" + strings.Repeat("a", 32)
	if _, err = maintenanceArgs(good); err == nil {
		t.Fatal("stage accepted for restore")
	}
}
