package command

import "jit/internal"

func Merge(targetBranch string) error {
	return internal.MergeCommits(targetBranch)
}
