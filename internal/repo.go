package internal

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/sergi/go-diff/diffmatchpatch"
	"io/fs"
	"jit/config"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CreateBranch creates a new branch with <name>
func CreateBranch(name string) error {
	repoPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w",
			err)
	}
	headCommitHash, err := getHEADCommit(repoPath)
	if err != nil {
		return err
	}
	branchRefPath := filepath.Join(config.REPO_DIR, config.REFS_DIR, "heads", name)

	return os.WriteFile(branchRefPath, []byte(headCommitHash+"\n"), 0644)
}

// ListBranches lists all the branches in the refs/heads
func ListBranches() error {
	refsDir := filepath.Join(config.REPO_DIR, config.REFS_DIR, "heads")

	files, err := os.ReadDir(refsDir)
	if err != nil {
		return fmt.Errorf("failed to list branches: %w", err)
	}

	currBranch, err := getCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed get current Branch: %w", err)
	}
	const (
		colorGreen = "\033[32m"
		colorReset = "\033[0m"
	)

	fmt.Println("Branches:")
	for _, file := range files {
		branchName := file.Name()
		if branchName == currBranch {
			fmt.Printf("* %s%s%s\n", colorGreen, branchName, colorReset)
		} else {
			fmt.Printf("  %s\n", branchName)
		}
	}
	return nil
}

// getCurrentBranch returns the currentBranch name
func getCurrentBranch() (string, error) {
	headPath := filepath.Join(config.REPO_DIR, config.HEAD_PATH)
	data, err := os.ReadFile(headPath)
	if err != nil {
		return "", fmt.Errorf("failed to read HEAD: %w", err)
	}
	ref := strings.TrimSpace(string(data))
	if strings.HasPrefix(ref, "ref:") {
		return filepath.Base(ref), nil
	}

	return "", fmt.Errorf("HEAD is not pointing to a branch")
}

// CheckoutBranch changes the current Branch to branchName
func CheckoutBranch(branchName string) error {
	branchPath := filepath.Join(config.REPO_DIR, config.REFS_DIR, "heads", branchName)
	branchHashBytes, err := os.ReadFile(branchPath)
	if err != nil {
		return fmt.Errorf("branch '%s' does not exist", branchName)
	}

	branchHash := strings.TrimSpace(string(branchHashBytes))
	currDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	currHeadHash, err := getHEADCommit(currDir)
	if err != nil {
		return fmt.Errorf("failed to get current HEAD commit: %w", err)
	}

	// unstaged and uncommitted changes
	hasChanges, err := hasChanges()
	if err != nil {
		return fmt.Errorf("failed to check for uncommitted changes: %w", err)
	}

	if hasChanges {
		return fmt.Errorf(
			"cannot switch branch: unstaged or uncommitted changes. Please commit your changes before switching branches.")
	}

	// switching to same commit, just update HEAD
	if branchHash == currHeadHash {
		if err := changeHEAD(branchName); err != nil {
			return fmt.Errorf("failed to update HEAD: %w", err)
		}
		return nil
	}

	// update index to match target branch
	targetCommit, err := LoadCommit(".", branchHash)
	if err != nil {
		return fmt.Errorf("failed to load target branch commit: %w", err)
	}

	err = updateIndexFromTree(targetCommit.TreeID)
	if err != nil {
		return fmt.Errorf("failed to update index: %w", err)
	}

	err = rebuildWorkingDirectory(currHeadHash, branchHash)
	if err != nil {
		return fmt.Errorf("failed to rebuild working directory: %w", err)
	}

	// update HEAD to point to new branch
	err = changeHEAD(branchName)
	if err != nil {
		return fmt.Errorf("failed to update HEAD: %w", err)
	}

	return nil
}

func updateIndexFromTree(treeHash string) error {
	tree, err := loadTree(treeHash)
	if err != nil {
		return fmt.Errorf("failed to load tree: %w", err)
	}

	var index Index
	for _, entry := range tree.Entries {
		index = append(index, IndexEntry{
			Filepath: entry.Name,
			Hash:     entry.Hash,
		})
	}

	err = saveIndex(&index)
	if err != nil {
		return fmt.Errorf("failed to save index: %w", err)
	}

	return nil
}

func rebuildWorkingDirectory(currentCommitHash, targetCommitHash string) error {
	currCommit, err := LoadCommit(".", currentCommitHash)
	if err != nil {
		return fmt.Errorf(
			"Error loading current commit '%s': %w", currentCommitHash, err)
	}
	targetCommit, err := LoadCommit(".", targetCommitHash)
	if err != nil {
		return fmt.Errorf(
			"Error loading target commit '%s': %w", targetCommitHash, err)
	}

	currTreeHash := currCommit.TreeID
	targetTreeHash := targetCommit.TreeID

	currTree, err := loadTree(currTreeHash)
	if err != nil {
		return fmt.Errorf(
			"failed to load current tree %s: %w", currTreeHash, err)
	}
	targetTree, err := loadTree(targetTreeHash)
	if err != nil {
		return fmt.Errorf(
			"failed to load target tree %s: %w", targetTreeHash, err)
	}

	err = updateWorkingDirectoryFromTrees(currTree, targetTree)
	if err != nil {
		return fmt.Errorf("failed to update working directory: %w", err)
	}
	return nil
}

func updateWorkingDirectoryFromTrees(currentTree, targetTree *Tree) error {
	// map for easy lookup
	targetEntries := make(map[string]TreeEntry)
	for _, entry := range targetTree.Entries {
		targetEntries[entry.Name] = entry
	}

	// handle files in curr tree but not in target tree (delete)
	for _, entry := range currentTree.Entries {
		if _, exists := targetEntries[entry.Name]; !exists {
			path := filepath.Join(".", entry.Name)
			if entry.Type == "tree" {
				err := os.RemoveAll(path)
				if err != nil {
					return fmt.Errorf("failed to remove directory '%s': %w", path, err)
				}
			} else {
				err := os.Remove(path)
				if err != nil {
					return fmt.Errorf("failed to remove file '%s': %w", path, err)
				}
			}
		}
	}

	// handle files in target tree (create or update)
	currEntries := make(map[string]TreeEntry)
	for _, entry := range currentTree.Entries {
		currEntries[entry.Name] = entry
	}

	for _, entry := range targetTree.Entries {
		path := filepath.Join(".", entry.Name)
		if currentEntry, exists := currEntries[entry.Name]; exists {
			// file exists in both trees, check if it needs updating
			if currentEntry.Hash != entry.Hash {
				err := extractBlob(entry.Hash, path)
				if err != nil {
					return fmt.Errorf("failed to update file '%s': %w", path, err)
				}
			}
		} else {
			// file or dir does not exist, create it
			if entry.Type == "tree" {
				err := os.MkdirAll(path, 0755)
				if err != nil {
					return fmt.Errorf("failed to create directory '%s': %w", path, err)
				}
				err = ExtractTree(".", entry.Hash, ".")
				if err != nil {
					return fmt.Errorf("failed to extract directory '%s': %w", path, err)
				}
			} else {
				err := extractBlob(entry.Hash, path)
				if err != nil {
					return fmt.Errorf("failed to create file '%s': %w", path, err)
				}
			}
		}
	}
	return nil
}

// extractBlob writes blob with hash to path
func extractBlob(hash, path string) error {
	blobPath := filepath.Join(config.REPO_DIR, config.OBJECTS_DIR, hash)
	content, err := os.ReadFile(blobPath)
	if err != nil {
		return fmt.Errorf("failed to read blob '%s': %w", hash, err)
	}

	err = os.WriteFile(path, content, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file '%s': %w", path, err)
	}

	return nil
}

// hasChanges compares headTreeHash, IndexTreeHash, and workingDirHash
// to detect uncommitted  or untracked changes
func hasChanges() (bool, error) {
	currHeadHash, err := getHEADCommit(".")
	if err != nil {
		return false, fmt.Errorf("failed to get HEAD commit: %w", err)
	}

	stagedFiles, err := loadIndex()
	if err != nil {
		return false, fmt.Errorf("failed to load Index: %w", err)
	}

	// tree in index
	idxTree, err := BuildTreeFromIndex(stagedFiles)
	if err != nil {
		return false, fmt.Errorf("failed to build Index Tree")
	}

	fakeWorkingIdx, err := CreateFakeIndex(".")
	if err != nil {
		return false, err
	}

	// workingTree, err := buildWorkingDirectoryTree(".")
	// if err != nil {
	// 	return false, fmt.Errorf("failed to build working directory tree")
	// }
	workingTree, err := BuildTreeFromIndex(fakeWorkingIdx)
	if err != nil {
		return false, err
	}

	headCommit, err := LoadCommit(".", currHeadHash)
	if err != nil {
		return false, fmt.Errorf("failed to load HEAD Commit")
	}

	hasUnstagedChanges := idxTree.Hash != workingTree.Hash
	hasUncommittedChange := idxTree.Hash != headCommit.TreeID
	fmt.Println("hasUnstagedChanges: ", hasUnstagedChanges)
	fmt.Println("hasUncommittedChange: ", hasUncommittedChange)

	return (hasUncommittedChange || hasUnstagedChanges), nil
}

func updateWorkingDirectory(commitHash string) error {
	currDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("Error getting current directory: %w", err)
	}

	commit, err := LoadCommit(currDir, commitHash)
	if err != nil {
		return fmt.Errorf("failed to read commit object: %w", err)
	}

	treeHash := commit.TreeID

	err = ExtractTree(currDir, treeHash, currDir)
	if err != nil {
		return fmt.Errorf("failed to extract tree: %w", err)
	}

	return nil
}

func changeHEAD(branchName string) error {
	headPath := filepath.Join(config.REPO_DIR, config.HEAD_PATH)
	return os.WriteFile(headPath, []byte(fmt.Sprintf("ref: refs/heads/%s\n", branchName)), 0644)
}

// CloneRepo makes a new repo in <dstPath> identical to repo in <srcPath>
func CloneRepo(srcPath, dstPath string) error {
	srcRepo := filepath.Join(srcPath, config.REPO_DIR)
	dstRepo := filepath.Join(dstPath, config.REPO_DIR)
	if err := CopyDir(srcRepo, dstRepo); err != nil {
		return fmt.Errorf("failed to copy .jit: %w", err)
	}

	if err := CheckoutLatestCommit(dstPath); err != nil {
		return fmt.Errorf("failed to checkout latest commit: %w", err)
	}

	return nil
}

// CheckoutLatestCommit recteates the repository in repoPath from .jit
func CheckoutLatestCommit(repoPath string) error {
	headCommitHash, err := getHEADCommit(repoPath)
	if err != nil {
		return fmt.Errorf("failed to read HEAD: %w", err)
	}

	headCommit, err := LoadCommit(repoPath, headCommitHash)
	if err != nil {
		return fmt.Errorf("failed to read commit object %s: %w", headCommitHash, err)
	}

	treeHash := headCommit.TreeID

	return ExtractTree(repoPath, treeHash, repoPath)
}

// getHEADCommit returns the <hash> of latest commit
func getHEADCommit(repoPath string) (string, error) {
	ref, err := os.ReadFile(filepath.Join(
		repoPath, config.REPO_DIR, config.HEAD_PATH))
	if err != nil {
		return "", err
	}
	refPath := strings.TrimSpace(string(ref))
	if strings.HasPrefix(refPath, "ref:") {
		refPath = filepath.Join(
			repoPath, config.REPO_DIR,
			strings.TrimSpace(strings.TrimPrefix(refPath, "ref:")),
		)
		// we read the file master to get latest commit
		hash, err := os.ReadFile(refPath)
		if err != nil {
			// we cannot create a branch if master does not exist
			if errors.Is(err, fs.ErrNotExist) {
				return "", fmt.Errorf("fatal: no valid object named 'master'")
			}
			return "", err
		}
		// else it has the hash of the latest commit
		return strings.TrimSpace(string(hash)), nil
	}
	return refPath, nil
}

// updateHEAD changes HEAD to point to <commitHash>
func updateHEADCommitHash(commitHash string) error {
	headContent, err := os.ReadFile(
		filepath.Join(config.REPO_DIR, config.HEAD_PATH),
	)
	if err != nil {
		return err
	}

	refLine := strings.TrimSpace(string(headContent))
	if strings.HasPrefix(refLine, "ref:") {
		refRelPath := strings.TrimSpace(strings.TrimPrefix(refLine, "ref:"))
		refFilepath := filepath.Join(config.REPO_DIR, refRelPath)
		return os.WriteFile(refFilepath, []byte(commitHash+"\n"), 0644)
	} else {
		// for jit checkout <commithash>, HEAD -> commit
		return os.WriteFile(
			filepath.Join(config.REPO_DIR, config.HEAD_PATH),
			[]byte(commitHash), 0644,
		)
	}
}

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
