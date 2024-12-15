package internal

import (
	"fmt"
	"jit/config"
	"os"
	"path/filepath"
	"strings"
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

// changeHEAD updates HEAD pointer to point to branchName
func changeHEAD(branchName string) error {
	headPath := filepath.Join(config.REPO_DIR, config.HEAD_PATH)
	return os.WriteFile(
		headPath, []byte(fmt.Sprintf("ref: refs/heads/%s\n", branchName)), 0644)
}
