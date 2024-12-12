package command

import (
	"fmt"
	"jit/internal"
)

func Checkout(branchName string) error {
	if err := internal.CheckoutBranch(branchName); err != nil {
		return err
	}
	fmt.Printf("Switched to branch '%s'\n", branchName)
	return nil
}
