package internal

import (
	"errors"
	"fmt"
	"io/fs"
	"jit/config"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CreateCommit creates a new commit with <message> and <timestamp>
func CreateCommit(message string, timestamp time.Time) (string, error) {
	stagedFiles, err := loadIndex()
	if err != nil {
		return "", err
	}
	if len(*stagedFiles) == 0 {
		return "", errors.New("no files staged")
	}

	tree, err := BuildTreeFromIndex(stagedFiles)
	if err != nil {
		return "", err
	}
	err = tree.Save()
	if err != nil {
		return "", err
	}

	commit := &Commit{
		Message:   message,
		Timestamp: timestamp,
		TreeID:    tree.Hash,
		ParentIDs: []string{},
	}

	repoPath, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("could not get current working directory: %w", err)
	}
	headCommit, _ := getHEADCommit(repoPath)
	if headCommit != "" {
		// first commit will not have any parents
		commit.ParentIDs = append(commit.ParentIDs, headCommit)
	}

	commitHash, err := commit.Save()
	if err != nil {
		return "", err
	}

	err = updateHEADCommitHash(commitHash)
	if err != nil {
		return "", err
	}

	return commitHash, nil
}

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
		mode, err := parseMode(entry.Mode)
		if err != nil {
			return err
		}
		index = append(index, IndexEntry{
			Filepath: entry.Name,
			Hash:     entry.Hash,
			Mode:     mode,
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
	fmt.Println()
	fmt.Println("Rebuilding tree")
	printTree(targetTree)
	fmt.Println()

	for _, entry := range targetTree.Entries {
		path := filepath.Join(".", entry.Name)
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
			err := extractBlob(&entry, path)
			if err != nil {
				return fmt.Errorf("failed to create file '%s': %w", path, err)
			}
		}
	}
	return nil
}

// extractBlob writes blob with hash to path
func extractBlob(treeEntry *TreeEntry, path string) error {
	blobPath := filepath.Join(config.REPO_DIR, config.OBJECTS_DIR, treeEntry.Hash)
	content, err := os.ReadFile(blobPath)
	if err != nil {
		return fmt.Errorf("failed to read blob '%s': %w", treeEntry.Hash, err)
	}

	err = os.WriteFile(path, content, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file '%s': %w", path, err)
	}

	mode, err := parseMode(treeEntry.Mode)
	if err != nil {
		return err
	}

	return os.Chmod(path, mode)
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

	workingTree, err := buildWorkingDirectoryTree(".")
	if err != nil {
		return false, fmt.Errorf("failed to build working directory tree")
	}
	headCommit, err := LoadCommit(".", currHeadHash)
	if err != nil {
		return false, fmt.Errorf("failed to load HEAD Commit")
	}

	hasUnstagedChanges := idxTree.Hash != workingTree.Hash
	hasUncommittedChange := idxTree.Hash != headCommit.TreeID
	fmt.Println("hasUnstagedChanges: ", hasUnstagedChanges)
	fmt.Println("hasUncommittedChange: ", hasUncommittedChange)

	fmt.Println()
	fmt.Println("idxTree")
	printTree(idxTree)
	fmt.Println()
	fmt.Println("workingTree")
	printTree(workingTree)
	fmt.Println()
	fmt.Println("currTree")
	currTree, err := loadTree(headCommit.TreeID)
	printTree(currTree)

	return (hasUncommittedChange || hasUnstagedChanges), nil
}

func printTree(tree *Tree) {
	for _, entry := range tree.Entries {
		fmt.Printf("Name: %s Type: %s, Mode: %s, Hash: %s\n", entry.Name, entry.Type, entry.Mode, entry.Hash)
	}
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
