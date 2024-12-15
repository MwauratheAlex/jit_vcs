package internal

import (
	"bytes"
	"fmt"
	"github.com/sergi/go-diff/diffmatchpatch"
	"jit/config"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MergeCommits merges the current branch to targetBranch
func MergeCommits(targetBranch string) error {
	targetBranchPath := filepath.Join(config.REPO_DIR, config.REFS_DIR, "heads", targetBranch)
	if _, err := os.Stat(targetBranchPath); err != nil {
		return fmt.Errorf("branch '%s' does not exist", targetBranch)
	}

	headCommitHash, err := getHEADCommit(".")
	if err != nil {
		return fmt.Errorf("failed to get HEAD commit: %w", err)
	}

	targetCommitHashBytes, err := os.ReadFile(targetBranchPath)
	if err != nil {
		return fmt.Errorf("failed to read target branch hash: %w", err)
	}
	targetCommitHash := strings.TrimSpace(string(targetCommitHashBytes))

	mergeBaseHash, err := findMergeBaseCommitHash(headCommitHash, targetCommitHash)
	if err != nil {
		return fmt.Errorf("failed to find merge base: %w", err)
	}

	fmt.Println("mergeBaseHash: ", mergeBaseHash)

	baseToHeadDiff, err := diffCommits(mergeBaseHash, headCommitHash)
	if err != nil {
		return fmt.Errorf("failed to generate diff  for base to HEAD: %w", err)
	}

	baseToTargetDiff, err := diffCommits(mergeBaseHash, targetCommitHash)
	if err != nil {
		return fmt.Errorf("failed to generate diff  for base to target: %w", err)
	}

	if len(baseToHeadDiff) == 0 && len(baseToTargetDiff) == 0 {
		fmt.Println("No changes in either branch since merge base.")
		return nil
	}
	if len(baseToHeadDiff) == 0 {
		fmt.Println("No changes in HEAD branch since the merge base.")
	}
	if len(baseToTargetDiff) == 0 {
		fmt.Println("No changes in target branch since the merge base.")
	}

	uniqueFiles := make(map[string]struct{})
	for file := range baseToHeadDiff {
		uniqueFiles[file] = struct{}{}
	}
	for file := range baseToTargetDiff {
		uniqueFiles[file] = struct{}{}
	}

	// reconcile changes
	fmt.Println("Looping")
	mergedTree := make(map[string]string)

	for file := range uniqueFiles {
		headDiffs, existsInHead := baseToHeadDiff[file]
		targetDiffs, existsInTarget := baseToTargetDiff[file]
		fmt.Printf("Processing file: %s\n", file)

		if !existsInHead && existsInTarget {
			fmt.Println("Only target branch changed")
			mergedContent := applyDiffsToBase("", targetDiffs)
			mergedTree[file] = mergedContent

		} else if existsInHead && !existsInTarget {
			fmt.Println("Only HEAD branch changed")
			mergedContent := applyDiffsToBase("", headDiffs)
			mergedTree[file] = mergedContent

		} else if existsInHead && existsInTarget {
			fmt.Println("Both branches changed")
			mergedContent, confict := mergeDiffs(headDiffs, targetDiffs)
			if confict {
				fmt.Printf("Conflict detected in file: %s\n", file)
			}
			mergedTree[file] = mergedContent
		} else {
			fmt.Println("file unchanged; load content from base commit")
		}
	}

	fmt.Println("mergedTree")
	for i, j := range mergedTree {
		fmt.Println(i, "\n", j)
	}

	if err := writeMergedTree(mergedTree, "."); err != nil {
		return fmt.Errorf("failed to write merged tree: %w", err)
	}

	if err := addMergedFilesToIndex(mergedTree, "."); err != nil {
		return fmt.Errorf("failed to add merged files to index: %w", err)
	}

	mergeMessage := fmt.Sprintf("Merged branch %s into HEAD", targetBranch)
	_, err = CreateCommit(mergeMessage, time.Now(), &targetCommitHash)
	if err != nil {
		return fmt.Errorf("failed to create merge commit: %w", err)
	}

	fmt.Println("Merged branch", targetBranch, "into HEAD")

	return nil
}

func printDiff(diffs map[string]string) {
	for file, diff := range diffs {
		fmt.Printf("Difference in '%s':\n%s\n", file, diff)
	}
}

func applyDiffsToBase(baseContent string, diffs []diffmatchpatch.Diff) string {
	dmp := diffmatchpatch.New()

	// handle empty base content
	if baseContent == "" {
		var result strings.Builder
		for _, diff := range diffs {
			if diff.Type == diffmatchpatch.DiffInsert || diff.Type == diffmatchpatch.DiffEqual {
				result.WriteString(diff.Text)
			}
		}
		return result.String()
	}

	patches := dmp.PatchMake(baseContent, diffs)
	mergedContent, _ := dmp.PatchApply(patches, baseContent)
	return mergedContent
}

func findMergeBaseCommitHash(commitAHash, commitBHash string) (string, error) {
	// get histories
	historyA, err := getCommitHistoryFromHash(commitAHash)
	if err != nil {
		return "", fmt.Errorf(
			"failed to get commit history for '%s': %w", commitAHash, err)
	}

	historyB, err := getCommitHistoryFromHash(commitBHash)
	if err != nil {
		return "", fmt.Errorf(
			"failed to get commit history for '%s': %w", commitBHash, err)
	}

	historyAMap := make(map[string]struct{}, len(historyA))
	for _, commit := range historyA {
		historyAMap[commit.Hash] = struct{}{}
	}

	// get first common commit
	for _, commit := range historyB {
		if _, exists := historyAMap[commit.Hash]; exists {
			return commit.Hash, nil
		}
	}

	return "", fmt.Errorf("no common ancestor found")
}

func mergeDiffs(headDiffs, targetDiffs []diffmatchpatch.Diff) (string, bool) {
	confict := false
	var mergedDiffs []diffmatchpatch.Diff

	i, j := 0, 0

	for i < len(headDiffs) && j < len(targetDiffs) {
		headDiff := headDiffs[i]
		targetDiff := targetDiffs[j]

		if headDiff.Type == targetDiff.Type && headDiff.Text == targetDiff.Text {
			// same changes
			mergedDiffs = append(mergedDiffs, headDiff)
			i++
			j++
		} else if headDiff.Type == diffmatchpatch.DiffEqual {
			// head has no change, apply target change
			mergedDiffs = append(mergedDiffs, targetDiff)
			i++
			j++
		} else if targetDiff.Type == diffmatchpatch.DiffEqual {
			// target has no change, apply head change
			mergedDiffs = append(mergedDiffs, headDiff)
			i++
			j++
		} else {
			// conflicts detected
			confict = true
			mergedDiffs = append(mergedDiffs,
				diffmatchpatch.Diff{Type: diffmatchpatch.DiffInsert, Text: "<<<<<<< HEAD\n"},
				headDiff,
				diffmatchpatch.Diff{Type: diffmatchpatch.DiffInsert, Text: "=======\n"},
				targetDiff,
				diffmatchpatch.Diff{Type: diffmatchpatch.DiffInsert, Text: ">>>>>>> target_branch\n"},
			)
			i++
			j++
		}

	}

	// append remaining diffs
	for i < len(headDiffs) {
		mergedDiffs = append(mergedDiffs, headDiffs[i])
		i++
	}
	for j < len(targetDiffs) {
		mergedDiffs = append(mergedDiffs, targetDiffs[j])
		j++
	}

	mergedContent := diffsToString(mergedDiffs)

	return mergedContent, confict
}

func diffsToString(diffs []diffmatchpatch.Diff) string {
	var sb bytes.Buffer
	for _, diff := range diffs {
		sb.WriteString(diff.Text)
	}
	return sb.String()
}

func writeMergedTree(mergedTree map[string]string, repoDir string) error {
	for path, content := range mergedTree {
		fullpath := filepath.Join(repoDir, path)
		// make sure parent exists
		if err := os.MkdirAll(filepath.Dir(fullpath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", path, err)
		}

		if err := os.WriteFile(fullpath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write merged content to %s: %w", path, err)
		}

	}
	return nil
}

func addMergedFilesToIndex(mergedTree map[string]string, repoDir string) error {
	for path := range mergedTree {
		fullpath := filepath.Join(repoDir, path)
		if err := AddToIndex(fullpath); err != nil {
			return fmt.Errorf("failed to add file %s to index: %w", path, err)
		}
	}
	return nil
}
