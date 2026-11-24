package main

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"dcr-manager/internal/app"
	"dcr-manager/internal/backend"
	"dcr-manager/internal/tui"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	snapshot := flag.Bool("snapshot", false, "print read-only diagnostic JSON")
	render := flag.Bool("render", false, "render dashboard from real diagnostic data")
	page := flag.Int("render-page", 1, "diagnostic render page (1-9)")
	version := flag.Bool("version", false, "print development version")
	settings := flag.Bool("settings", false, "print read-only manager preferences and startup registration")
	login := flag.Bool("start-remote-on-login", false, "start Remote if verified stopped")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional arguments")
		os.Exit(2)
	}
	if *version {
		fmt.Println(tui.Version)
		return
	}
	if *settings {
		if *snapshot || *render || *login {
			fmt.Fprintln(os.Stderr, "incompatible modes")
			os.Exit(2)
		}
		if e := json.NewEncoder(os.Stdout).Encode(backend.ReadSettings()); e != nil {
			os.Exit(1)
		}
		return
	}
	root, err := backend.Root()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	read := backend.Read(root)
	if *login {
		if *snapshot || *render {
			fmt.Fprintln(os.Stderr, "incompatible modes")
			os.Exit(2)
		}
		s, e := read(context.Background())
		if e != nil || app.LoginDecision(s) == "inventory-unavailable" {
			fmt.Fprintln(os.Stderr, "Remote startup skipped: process inventory unavailable")
			os.Exit(1)
		}
		if app.LoginDecision(s) == "already-running" {
			return
		}
		if app.LoginDecision(s) != "start" {
			fmt.Fprintln(os.Stderr, "Remote startup skipped: prerequisites missing")
			os.Exit(1)
		}
		if e = backend.Operate(root)(context.Background(), app.Start, nil); e != nil {
			fmt.Fprintln(os.Stderr, "Remote startup not verified; open DCR Manager")
			os.Exit(1)
		}
		return
	}
	if *snapshot || *render {
		s, e := read(context.Background())
		if e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
		if *snapshot {
			if e = json.NewEncoder(os.Stdout).Encode(s); e != nil {
				os.Exit(1)
			}
		} else {
			if *page < 1 || *page > 9 {
				fmt.Fprintln(os.Stderr, "render-page must be 1-9")
				os.Exit(2)
			}
			m := tui.New(nil).WithOperator(backend.Operate(root)).WithMaintainer(backend.Maintain(root)).WithSettings(backend.ReadSettings(), backend.ChangeSettings)
			fmt.Println(m.RenderAt(s, 110, 35, *page-1))
		}
		return
	}
	if _, err = tea.NewProgram(tui.New(read).WithOperator(backend.Operate(root)).WithMaintainer(backend.Maintain(root)).WithSettings(backend.ReadSettings(), backend.ChangeSettings)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
