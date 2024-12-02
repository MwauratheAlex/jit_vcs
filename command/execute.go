package command

import "fmt"

func Execute(args []string) error {
	command := args[0]

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
