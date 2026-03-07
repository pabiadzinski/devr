package main

import (
	"fmt"
	"os"

	"github.com/pabiadzinski/devr"
)

const name = "devr"

var version = "dev"

func main() {
	var (
		dir   string
		debug bool
	)

	a := devr.NewApp(name, "")

	c := &devr.CLI{
		Name:    name,
		Version: version,
		Flags: []devr.Flag{
			{Name: "dir", Short: "C", Usage: "Change to directory before running", Value: &dir},
			{Name: "debug", Short: "v", Usage: "Enable debug logging", Bool: &debug},
		},
		Setup: func() error {
			devr.LogDebug = debug

			if dir != "" {
				if err := os.Chdir(dir); err != nil {
					return err
				}

				*a = *devr.NewApp(name, "")
			}

			return nil
		},
	}

	devr.Register(c, a)

	if err := c.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
