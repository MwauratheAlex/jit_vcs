package command

import (
	"fmt"
	"jit_vcs/internal"
	"strings"

	"github.com/fatih/color"
)

func Log() error {
	commits, err := internal.GetCommitHistory()
	if err != nil {
		return err
	}
	for _, commit := range commits {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Date: %s\n", commit.Timestamp))
		sb.WriteString(fmt.Sprintf("\n\t%s\n", commit.Message))

		color.Yellow(fmt.Sprintf("Commit %s\n", commit.Hash))
		fmt.Println(sb.String())
	}
	return nil
}
