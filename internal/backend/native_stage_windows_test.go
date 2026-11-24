//go:build windows

package backend

import (
	"context"
	"dcr-manager/internal/app"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNativeArtifactAcceptance(t *testing.T) {
	if os.Getenv("DCR_NATIVE_ARTIFACT_TEST") != "1" {
		t.Skip("explicit network opt-in required")
	}
	base := t.TempDir()
	root := filepath.Join(base, "install")
	state := filepath.Join(base, "state")
	stageFixture(t, root, "0.3.4")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	p, e := prepareNativePackage(ctx, root, state, app.RepairPackage, "0.3.4", func(_ context.Context, path string) error {
		if path != root {
			t.Fatal("unexpected fixture path")
		}
		return nil
	}, runNativeArtifactStage)
	if e != nil || p.Version != "0.3.4" {
		t.Fatal("real npm artifact staging failed", e)
	}
	raw, e := os.ReadFile(filepath.Join(root, "node_modules", "@wonderwhy-er", "desktop-commander", "dist", "index.js"))
	if e != nil || string(raw) != "// fixture" {
		t.Fatal("original fixture was replaced")
	}
	t.Log("Pinned npm artifact validated; no lifecycle scripts or live installation changes")
}
