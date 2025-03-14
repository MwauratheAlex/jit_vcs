package main

import (
	"fmt"
	"jit/command"
	"os"
)

func main() {
	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
