package backend

import (
	"dcr-manager/internal/app"
	"strings"
	"testing"
)

func TestActionArgs(t *testing.T) {
	a, e := actionArgs(app.Stop, []int{42, 3})
	if e != nil || strings.Join(a, " ") != "-Action stop -Expected 3,42" {
		t.Fatalf("%v %v", a, e)
	}
	for _, tc := range []struct {
		a app.RemoteAction
		p []int
	}{{"other", nil}, {app.Start, []int{3}}, {app.Stop, nil}, {app.Restart, []int{0}}, {app.Stop, []int{4, 4}}} {
		if _, e := actionArgs(tc.a, tc.p); e == nil {
			t.Fatalf("accepted invalid %v", tc)
		}
	}
}
