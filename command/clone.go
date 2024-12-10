package command

import (
	"fmt"
	"jit_vcs/internal"
)

func Clone(srcPath, destPath string) error {
	if err := internal.CloneRepo(srcPath, destPath); err != nil {
		return err
	}
	fmt.Printf("Cloned repository to '%s'\n", destPath)
	return nil
}
