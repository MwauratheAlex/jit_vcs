package command

import (
	"flag"
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
		paths := args
		return Add(paths)
	case "commit":
		msgFlag := flag.NewFlagSet("commit", flag.ExitOnError)
		msg := msgFlag.String("m", "", "Commit message")
		_ = msgFlag.Parse(args)
		return Commit(*msg)
	case "log":
		return Log()
	case "branch":
		return Branch()
	case "merge":
		return Merge()
	case "diff":
		return Diff()
	case "clone":
		return Clone()
	default:
		return fmt.Errorf("Unknown command: %s", command)
	}
}
