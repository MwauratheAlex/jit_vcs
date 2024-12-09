package command

import (
	"fmt"
	"jit_vcs/internal"
)

func Add(paths []string) error {

	if len(paths) < 1 {
		return fmt.Errorf("No file specified.\nUsage: jit add <file>")
	}

	// TODO: Make this concurrent
	for _, path := range paths {
		if err := internal.AddToIndex(path); err != nil {
			return err
		}
		fmt.Printf("Added '%s' to staging area.\n", path)
	}

	return nil
}
