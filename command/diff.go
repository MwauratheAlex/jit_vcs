package command

import (
	"fmt"
	"jit_vcs/internal"
)

func Diff(commitHash1, commitHash2 string) error {
	diffs, err := internal.DiffCommits(commitHash1, commitHash2)
	if err != nil {
		return fmt.Errorf("Failed to diff commits '%s' and '%s':%w",
			commitHash1, commitHash2, err)
	}

	for file, diff := range diffs {
		fmt.Printf("Difference in '%s':\n%s\n", file, diff)
	}

	return nil
}
