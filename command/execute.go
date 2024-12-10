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
		if len(args) < 1 {
			return fmt.Errorf("Please provide a branch name.\nUsage: jit branch <branch name>")
		}
		return Branch(args[0])
	case "merge":
		// TODO
		return Merge()
	case "diff":
		// TODO
		return Diff()
	case "clone":
		if len(args) < 2 {
			return fmt.Errorf("Please provide source and destination paths.\nUsage: jit clone <src path> <dest path>")
		}
		return Clone(args[0], args[1])
	default:
		return fmt.Errorf("Unknown command: %s", command)
	}
}
