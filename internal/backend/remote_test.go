package backend

import (
	"dcr-manager/internal/app"
	"testing"
)

func TestActionArgs(t *testing.T) {
	if e := validateAction(app.Stop, []app.ProcessIdentity{{PID: 42, Created: 1}}); e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		a app.RemoteAction
		p []app.ProcessIdentity
	}{{"other", nil}, {app.Start, []app.ProcessIdentity{{PID: 3, Created: 1}}}, {app.Stop, nil}, {app.Restart, []app.ProcessIdentity{{PID: 0, Created: 1}}}, {app.Stop, []app.ProcessIdentity{{PID: 4, Created: 1}, {PID: 4, Created: 1}}}, {app.Stop, []app.ProcessIdentity{{PID: 3}}}} {
		if e := validateAction(tc.a, tc.p); e == nil {
			t.Fatalf("accepted invalid %v", tc)
		}
	}
}
