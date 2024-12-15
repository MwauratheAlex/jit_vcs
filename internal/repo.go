package internal

import (
	"fmt"
	"jit/config"
	"os"
	"path/filepath"
)

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
	fmt.Println("Staged Index")
	printIndex(stagedFiles)
	fmt.Println()

	// tree in index
	idxTree, err := BuildTreeFromIndex(stagedFiles)
	if err != nil {
		return false, fmt.Errorf("failed to build Index Tree")
	}

	fakeWorkingIdx, err := CreateFakeIndex(".")
	if err != nil {
		return false, err
	}
	fmt.Println("Fake Index")
	printIndex(fakeWorkingIdx)
	fmt.Println()

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
	// fmt.Println("hasUnstagedChanges: ", hasUnstagedChanges)
	// fmt.Println("hasUncommittedChange: ", hasUncommittedChange)
	// fmt.Println()
	// fmt.Println("working tree")
	// printTree(workingTree)
	// fmt.Println()
	// fmt.Println("Index tree")
	// printTree(idxTree)
	// fmt.Println()
	// wkingtree, err := loadTree(headCommit.Hash)
	// fmt.Println("commitTree")
	// if err != nil {
	// 	printTree(wkingtree)
	// } else {
	// 	fmt.Println(err)
	// }

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
