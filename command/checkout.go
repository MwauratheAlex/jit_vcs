package command

import (
	"fmt"
	"jit_vcs/internal"
)

func Checkout(branchName string) error {
	if err := internal.CheckoutBranch(branchName); err != nil {
		return err
	}
	fmt.Printf("Checked out to the branch '%s'\n", branchName)
	return nil
}
