package command

import (
	"fmt"
	"jit_vcs/internal"
	"time"
)

func Commit(message string) error {
	if message == "" {
		return fmt.Errorf(
			"%sCommit message is missing.%s\nUsage: jit commit -m 'commit message'",
			colorRed, colorNone,
		)
	}

	commitID, err := internal.CreateCommit(message, time.Now())
	if err != nil {
		return fmt.Errorf("Error creating commit: %v\n", err)
	}

	fmt.Printf("Committed as %s\n", commitID)
	return nil
}
