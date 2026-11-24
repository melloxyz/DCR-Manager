package main

import (
	tea "charm.land/bubbletea/v2"
	"context"
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
	version := flag.Bool("version", false, "print development version")
	flag.Parse()
	if *version {
		fmt.Println(tui.Version)
		return
	}
	root, err := backend.Root()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	read := backend.Read(root)
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
			fmt.Println(tui.RenderSnapshot(s, 110, 35))
		}
		return
	}
	if _, err = tea.NewProgram(tui.New(read)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
