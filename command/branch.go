package command

import (
	"fmt"
	"jit_vcs/internal"
)

func Branch(name string) error {
	if err := internal.CreateBranch(name); err != nil {
		return err
	}
	fmt.Printf("Created branch '%s'\n", name)
	return nil
}

func ListBranches() error {
	if err := internal.ListBranches(); err != nil {
		return err
	}
	return nil
}
