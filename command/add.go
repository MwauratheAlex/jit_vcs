package command

import (
	"fmt"
	"jit_vcs/vcs"
)

func Add(paths []string) error {
	// TODO: Make this concurrent

	for _, path := range paths {
		if err := vcs.AddToIndex(path); err != nil {
			return err
		}
		fmt.Printf("Added '%s' to staging area.\n", path)
	}

	return nil
}
