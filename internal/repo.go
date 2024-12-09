package internal

import (
	"errors"
	"fmt"
	"io/fs"
	"jit_vcs/config"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type IndexEntry struct {
	Hash     string
	Filepath string
}

type Index []IndexEntry

// AddToIndex adds a file to the staging area
func AddToIndex(path string) error {
	//TODO: check if file is ignored

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	hash := ComputeHash(content)

	objectPath := filepath.Join(config.REPO_DIR, config.OBJECTS_DIR, hash)
	if _, err := os.Stat(objectPath); errors.Is(err, fs.ErrNotExist) {
		if err := os.WriteFile(objectPath, content, 0644); err != nil {
			return err
		}
	}

	indexPath := filepath.Join(config.REPO_DIR, "index")
	indexEntry := fmt.Sprintf("%s %s\n", hash, path)
	f, err := os.OpenFile(indexPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(indexEntry)

	return err
}

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
		TreeID:    tree.ID,
		ParentIDs: []string{},
	}

	headCommit, _ := getHEADCommit()
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

	// clear index
	err = os.WriteFile(filepath.Join(config.REPO_DIR, "index"), []byte(""), 0644)

	if err != nil {
		return "", err
	}

	return commitHash, nil
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

			idxEntry := IndexEntry{
				Hash:     idxEntries[0],
				Filepath: idxEntries[1],
			}

			index = append(index, idxEntry)
		}
	}

	return &index, nil
}

func getHEADCommit() (string, error) {
	ref, err := os.ReadFile(filepath.Join(config.REPO_DIR, config.HEAD_PATH))
	if err != nil {
		return "", err
	}
	refPath := strings.TrimSpace(string(ref))
	if strings.HasPrefix(refPath, "ref:") {
		refPath = filepath.Join(
			config.REPO_DIR,
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
		return os.WriteFile(refFilepath, []byte(commitHash), 0644)
	} else {
		// for jit checkout <commit>, HEAD -> commit
		return os.WriteFile(
			filepath.Join(config.REPO_DIR, config.HEAD_PATH),
			[]byte(commitHash), 0644,
		)
	}
}
