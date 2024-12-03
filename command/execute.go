package command

import (
	"fmt"
	"os"
)

func Execute() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("Usage: jit <command> [options]")
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "init":
		return Init(args)
	case "add":
		return Add(args)
	case "commit":
		return Commit(args)
	default:
		return fmt.Errorf("Unknown command: %s", command)
	}
}
