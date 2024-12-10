package internal

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"jit_vcs/config"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type IndexEntry struct {
	Hash     string
	Filepath string
	Mode     os.FileMode
}

type Index []IndexEntry

// AddToIndex adds a file with <path> to the staging area
func AddToIndex(path string) error {
	//TODO: check if file is ignored

	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return err
	}

	if info.IsDir() {
		// walk
		return filepath.Walk(
			absPath,
			func(filePath string, fileInfo fs.FileInfo, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				// skip dirs, process only their files
				if fileInfo.IsDir() {
					return nil
				}

				// add files to index
				_, err := filepath.Rel(absPath, filePath)
				if err != nil {
					return err
				}
				return AddToIndex(filePath)
			})
	}

	// handle files
	content, err := os.ReadFile(absPath)
	if err != nil {
		return err
	}

	hash := ComputeHash(content)

	// write obj to if does not exist
	objectPath := filepath.Join(config.REPO_DIR, config.OBJECTS_DIR, hash)
	if _, err := os.Stat(objectPath); err != nil {
		if err := os.WriteFile(objectPath, content, 0644); err != nil {
			return err
		}
	}

	// write to index
	indexPath := filepath.Join(config.REPO_DIR, "index")
	mode := fmt.Sprintf("%04o", info.Mode().Perm())

	indexEntry := fmt.Sprintf("%s %s %s\n", hash, mode, path)

	f, err := os.OpenFile(indexPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(indexEntry)

	return err
}

// CreateCommit creates a new commit with <message> and <timestamp>
func CreateCommit(message string, timestamp time.Time) (string, error) {
	stagedFiles, err := loadIndex()
	if err != nil {
		return "", err
	}
	if len(*stagedFiles) == 0 {
		return "", errors.New("no files staged")
	}

	tree, err := BuildTreeFromFiles(stagedFiles)
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

	err = updateHEAD(commitHash)
	if err != nil {
		return "", err
	}

	// clear index - might remove this when tree is implemented fully
	err = os.WriteFile(filepath.Join(config.REPO_DIR, "index"), []byte(""), 0644)

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

// CopyDir copies <src> directory to <dst> directory
func CopyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		fileInfo, err := os.Stat(srcPath)
		if err != nil {
			return err
		}
		switch {
		case fileInfo.IsDir():
			err = CopyDir(srcPath, dstPath)
			if err != nil {
				return err
			}
		default:
			err = CopyFile(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// CopyFile copies <src> file to <dst> file
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)

	return err
}

// TODO: Implement like real world

// 1. find tree for commit hash.
// 2. recursively extract files from tree and write to working dir

// CheckoutLatestCommit copies all files from srcPath to dstPath, excluding .jit
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

func loadIndex() (*Index, error) {
	var index Index

	data, err := os.ReadFile(filepath.Join(config.REPO_DIR, "index"))
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			idxEntries := strings.Split(l, " ")
			if len(idxEntries) < 3 {
				continue
			}
			modeUint, err := strconv.ParseUint(idxEntries[1], 8, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid mode in index: %s", idxEntries[1])
			}
			mode := os.FileMode(modeUint)
			idxEntry := IndexEntry{
				Hash:     idxEntries[0],
				Mode:     mode,
				Filepath: idxEntries[2],
			}

			index = append(index, idxEntry)
		}

	}

	return &index, nil
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
			// here master is empty, e.g. after init
			return "", err
		}
		// else it has the hash of the latest commit
		return strings.TrimSpace(string(hash)), nil
	}
	return refPath, nil
}

// updateHEAD changes HEAD to point to <commitHash>
func updateHEAD(commitHash string) error {
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
