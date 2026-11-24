package tui

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"dcr-manager/internal/app"
	"errors"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

func actionKey(m Model, k rune) (Model, tea.Cmd) {
	n, c := m.Update(tea.KeyPressMsg{Code: k, Text: string(k)})
	return n.(Model), c
}
func actionFixture() Model {
	m := New(nil)
	m.ready = true
	m.data.Status = app.Status{Version: "0.3.4", Installed: true, NodeAvailable: true, RemoteProcesses: 1, RemoteRoots: 1}
	m.data.Processes = []app.Process{{PID: 123, Role: "Remote"}}
	m.selectPage(1)
	return m
}
func TestStopRequiresConfirmationAndIsAsync(t *testing.T) {
	m := actionFixture()
	calls := 0
	m.operator = func(_ context.Context, a app.RemoteAction, pids []int) error {
		calls++
		if a != app.Stop || len(pids) != 1 || pids[0] != 123 {
			t.Fatal("wrong action target")
		}
		return nil
	}
	m, cmd := actionKey(m, 'x')
	if cmd != nil || calls != 0 || m.pending != app.Stop {
		t.Fatal("stop did not ask for confirmation")
	}
	if !strings.Contains(ansi.Strip(m.View().Content), "disconnect") {
		t.Fatal("missing warning")
	}
	m, cmd = actionKey(m, 'y')
	if cmd == nil || calls != 0 || !m.busy {
		t.Fatal("operation not scheduled asynchronously")
	}
	_, other := actionKey(m, 'x')
	if other != nil {
		t.Fatal("concurrent action allowed")
	}
	msg := cmd()
	if calls != 1 {
		t.Fatal("operation not invoked")
	}
	n, _ := m.Update(msg)
	if n.(Model).busy {
		t.Fatal("busy state not cleared")
	}
}
func TestCancelDoesNotCallOperator(t *testing.T) {
	m := actionFixture()
	m.operator = func(context.Context, app.RemoteAction, []int) error { t.Fatal("operation executed"); return nil }
	m, _ = actionKey(m, 't')
	m, cmd := actionKey(m, 'n')
	if cmd != nil || m.pending != "" || m.busy {
		t.Fatal("cancel did not clear confirmation")
	}
}
func TestStaleSnapshotAndDuplicateStartRefused(t *testing.T) {
	m := actionFixture()
	m.operator = func(context.Context, app.RemoteAction, []int) error { return nil }
	_, cmd := actionKey(m, 's')
	if cmd != nil {
		t.Fatal("duplicate start allowed")
	}
	m.err = "stale"
	m, cmd = actionKey(m, 'x')
	if cmd != nil || m.pending != "" {
		t.Fatal("stale action permitted")
	}
}
func TestFailureClearsBusyAndRequestsRefresh(t *testing.T) {
	m := actionFixture()
	m.operator = func(context.Context, app.RemoteAction, []int) error { return errors.New("test failure") }
	m.reader = func(context.Context) (app.Snapshot, error) { return app.Snapshot{}, nil }
	m, _ = actionKey(m, 'x')
	m, cmd := actionKey(m, 'y')
	n, refresh := m.Update(cmd())
	got := n.(Model)
	if got.busy || !got.loading || refresh == nil || !strings.Contains(got.actionNotice, "test failure") {
		t.Fatal("failure not reported/refreshed")
	}
}

func TestStartIsAsyncAndOnlyAvailableWhenStopped(t *testing.T) {
	m := actionFixture()
	m.data.Status.RemoteProcesses = 0
	m.data.Status.RemoteRoots = 0
	m.data.Processes = nil
	called := false
	m.operator = func(_ context.Context, a app.RemoteAction, pids []int) error {
		called = true
		if a != app.Start || len(pids) != 0 {
			t.Fatal("invalid start request")
		}
		return nil
	}
	m, c := actionKey(m, 's')
	if c == nil || called || !m.busy {
		t.Fatal("start did not schedule")
	}
	c()
	if !called {
		t.Fatal("start did not call operator")
	}
}
func TestConfirmationVisibleAtSmallTerminal(t *testing.T) {
	m := actionFixture()
	m.width, m.height = 55, 14
	m.operator = func(context.Context, app.RemoteAction, []int) error { return nil }
	m, _ = actionKey(m, 'x')
	if !strings.Contains(ansi.Strip(m.View().Content), "Confirm") {
		t.Fatal("confirmation not visible")
	}
}
