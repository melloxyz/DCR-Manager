package tui

import (
	"charm.land/lipgloss/v2"
	"dcr-manager/internal/app"
	"github.com/charmbracelet/x/ansi"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var sgrRE = regexp.MustCompile("\x1b\\[([0-9;]*)m")

// Terminal default colors may be translucent. Every painted cell must have a
// background, even after nested styles emit SGR resets.
func assertOpaque(t *testing.T, row string) {
	t.Helper()
	active := false
	offset := 0
	for _, p := range sgrRE.FindAllStringSubmatchIndex(row, -1) {
		if p[0] > offset && ansi.StringWidth(row[offset:p[0]]) > 0 && !active {
			t.Fatalf("unpainted cells: %q", row[offset:p[0]])
		}
		codes := strings.Split(row[p[2]:p[3]], ";")
		for i := 0; i < len(codes); i++ {
			n, _ := strconv.Atoi(codes[i])
			switch {
			case n == 0 || n == 49:
				active = false
			case n == 48 || n == 38:
				if n == 48 {
					active = true
				}
				if i+1 < len(codes) {
					if codes[i+1] == "2" {
						i += 4
					} else if codes[i+1] == "5" {
						i += 2
					}
				}
			case n >= 40 && n <= 47 || n >= 100 && n <= 107:
				active = true
			}
		}
		offset = p[1]
	}
	if ansi.StringWidth(row[offset:]) > 0 && !active {
		t.Fatalf("unpainted suffix: %q", row[offset:])
	}
}
func TestOpaqueFrameAllPages(t *testing.T) {
	for _, size := range [][2]int{{112, 28}, {120, 36}, {75, 20}, {42, 10}} {
		for page := range pages {
			m := New(nil)
			m.width, m.height = size[0], size[1]
			m.ready = true
			m.data = app.Snapshot{Status: app.Status{Version: "0.3.4", Installed: true, RemoteProcesses: 3, RemoteRoots: 1, CacheKnown: true, CacheCandidates: 1}, Installations: []app.Installation{{Kind: "dedicated", Version: "0.3.4", EntryExists: true}, {Kind: "npx", Version: "0.3.3", EntryExists: true}}}
			m.selectPage(page)
			rows := strings.Split(m.View().Content, "\n")
			if len(rows) != size[1] {
				t.Fatalf("page %d: got %d rows want %d", page, len(rows), size[1])
			}
			for _, row := range rows {
				if lipgloss.Width(row) != size[0] {
					t.Fatalf("page %d: row width %d want %d", page, lipgloss.Width(row), size[0])
				}
				assertOpaque(t, row)
			}
		}
	}
}
func TestPackageLanguage(t *testing.T) {
	m := New(nil)
	m.ready = true
	m.data.Installations = []app.Installation{{Kind: "dedicated", Version: "0.3.4", EntryExists: true}, {Kind: "npx", Version: "0.3.3", EntryExists: true}}
	m.selectPage(2)
	text := ansi.Strip(m.vp.GetContent())
	for _, want := range []string{"Dedicated installation", "Legacy npm cache", "Installed", "v0.3.4"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q", want)
		}
	}
	if strings.Contains(text, "Entry present: true") {
		t.Fatal("raw boolean in presentation")
	}
}

func TestExactPalette(t *testing.T) {
	if backgroundSequence("#212121") != "\x1b[48;2;33;33;33m" {
		t.Fatal("canvas RGB mismatch")
	}
	if backgroundSequence("#171717") != "\x1b[48;2;23;23;23m" {
		t.Fatal("sidebar RGB mismatch")
	}
	m := New(nil)
	m.width, m.height = 112, 28
	m.ready = true
	m.data.Installations = []app.Installation{{Kind: "dedicated", Version: "0.3.4", EntryExists: true}}
	m.selectPage(2)
	rows := strings.Split(m.View().Content, "\n")
	if !strings.Contains(rows[8], backgroundSequence("#303030")) {
		t.Fatal("selected nav surface missing")
	}
	if !strings.Contains(strings.Join(rows, "\n"), backgroundSequence("#292929")) {
		t.Fatal("card surface missing")
	}
}
