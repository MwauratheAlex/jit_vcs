package command

import (
	"jit/internal"
)

func Branch(name string) error {
	if err := internal.CreateBranch(name); err != nil {
		return err
	}
	return nil
}

func ListBranches() error {
	if err := internal.ListBranches(); err != nil {
		return err
	}
	return nil
}
