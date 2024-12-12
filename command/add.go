package command

import (
	"fmt"
	"jit/internal"
)

func Add(paths []string) error {

	if len(paths) < 1 {
		return fmt.Errorf("%sNo file specified.%s\nUsage: jit add <file>",
			colorRed, colorNone,
		)
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
