package command

import (
	"fmt"
	"jit_vcs/internal"
	"os"
	"strings"
)

const (
	colorRed    = "\033[0;31m"
	colorYellow = "\033[0;33m"
	colorNone   = "\033[0m"
)

func Log() error {
	commits, err := internal.GetCommitHistory()
	if err != nil {
		return err
	}
	for _, commit := range commits {

		fmt.Fprintf(os.Stdout, "%sCommit  %s%s\n", colorYellow, commit.Hash, colorNone)

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Date: %s\n", commit.Timestamp))
		sb.WriteString(fmt.Sprintf("\n\t%s\n", commit.Message))

		fmt.Fprintf(os.Stdout, "%s\n", sb.String())

	}
	return nil
}
