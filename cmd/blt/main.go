package main

import (
	"github.com/rdo34/blt/internal/ui"
	"os"
)

func main() {
	// If a CLI subcommand is provided, handle it and exit.
	if len(os.Args) > 1 {
		if handled, code := runCLI(os.Args[1:]); handled {
			os.Exit(code)
			return
		}
	}
	if err := ui.New().Run(); err != nil {
		panic(err)
	}
}
