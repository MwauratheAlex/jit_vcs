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
			return fmt.Errorf(
				"%sPlease provide a branch name%s.\nUsage: jit branch <branch name>",
				colorRed, colorNone)
		}
		return Branch(args[0])
	case "merge":
		// TODO
		return Merge()
	case "diff":
		if len(args) < 2 {
			return fmt.Errorf(
				"%sPlease provide commits hashes to diff.%s\nUsage: jit diff <commit1Hash> <commit2Hash>",
				colorRed, colorNone)
		}

		return Diff(args[0], args[1])
	case "clone":
		if len(args) < 2 {
			return fmt.Errorf(
				"%sPlease provide source and destination paths.%s\nUsage: jit clone <src path> <dest path>",
				colorRed, colorNone)
		}
		return Clone(args[0], args[1])
	default:
		return fmt.Errorf("Unknown command: %s", command)
	}
}
