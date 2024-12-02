package main

import (
	"fmt"
	"jit_vcs/command"
	"os"
)

func main() {
	args := os.Args[1:]

	if err := command.Execute(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
