package tui

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"dcr-manager/internal/app"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

func TestFailedRefreshHidesOldHealth(t *testing.T) {
	m := actionFixture()
	m.selectPage(0)
	m.err = "refresh failed"
	m.sync()
	text := ansi.Strip(m.vp.GetContent())
	if strings.Contains(text, "Running") || !strings.Contains(text, "Snapshot unavailable") {
		t.Fatal(text)
	}
}
func TestUnknownCacheNotZero(t *testing.T) {
	m := New(nil)
	m.ready = true
	m.data.Status = app.Status{CacheKnown: false}
	m.sync()
	text := ansi.Strip(m.vp.GetContent())
	if !strings.Contains(text, "Not assessed") {
		t.Fatal(text)
	}
	if strings.Contains(text, "Cache unavailable") {
		t.Fatal("unavailable cache must use explicit unknown count")
	}
}
func TestConfirmationRemainsVisible(t *testing.T) {
	m := recoveryFixture()
	m.maintainer = func(rctx context.Context, r app.MaintenanceRequest) error {
		t.Fatal("unexpected execution")
		return nil
	}
	m, _ = actionKey(m, 'v')
	n, _ := m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	m = n.(Model)
	if !m.confirmationFits() || !strings.Contains(ansi.Strip(m.View().Content), "[Y] Confirm") {
		t.Fatal("confirmation not fully visible")
	}
	n, _ = m.Update(tea.WindowSizeMsg{Width: 76, Height: 18})
	m = n.(Model)
	if m.confirmationFits() && !strings.Contains(ansi.Strip(m.View().Content), "[Y] Confirm") {
		t.Fatal("hidden confirmation")
	}
}
