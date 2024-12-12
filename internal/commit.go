package internal

import (
	"fmt"
	"jit/config"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mrk21/go-diff-fmt/difffmt"
	"github.com/sergi/go-diff/diffmatchpatch"
)

type Commit struct {
	Hash      string
	Message   string
	Timestamp time.Time
	TreeID    string
	ParentIDs []string
}

func (c *Commit) Serialize() []byte {
	// format
	// tree <TreeID>
	// parent <ParentID 1>
	// parent <ParentID 2>...
	// timestamp <UNIX timestamps>
	//
	// <commit message>
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("tree %s\n", c.TreeID))
	for _, p := range c.ParentIDs {
		sb.WriteString(fmt.Sprintf("parent %s\n", p))

	}

	sb.WriteString(fmt.Sprintf("timestamp %d\n", c.Timestamp.Unix()))
	sb.WriteString(fmt.Sprintf("\n%s\n", c.Message))

	return []byte(sb.String())
}

func (c *Commit) Save() (string, error) {
	data := c.Serialize()
	hash := ComputeHash(data)

	err := os.WriteFile(
		filepath.Join(config.REPO_DIR, config.OBJECTS_DIR, hash), data, 0644,
	)
	if err != nil {
		return "", err
	}
	return hash, nil
}

var c *Commit = nil

// LoadCommit returns the commit with the given <commitHash>
// caches because commits are immutable
func LoadCommit(repoPath, commitHash string) (*Commit, error) {
	if c != nil && c.Hash == commitHash {
		return c, nil
	}
	c = &Commit{}

	data, err := os.ReadFile(filepath.Join(repoPath,
		config.REPO_DIR, config.OBJECTS_DIR, commitHash))
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var i int
	for ; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			i++
			break // blank line, rest is message
		}
		switch {
		case strings.HasPrefix(line, "tree"):
			c.TreeID = strings.TrimSpace(strings.TrimPrefix(line, "tree"))
		case strings.HasPrefix(line, "parent"):
			parentID := strings.TrimSpace(strings.TrimPrefix(line, "parent"))
			c.ParentIDs = append(c.ParentIDs, parentID)
		case strings.HasPrefix(line, "timestamp"):
			timestamp := strings.TrimSpace(strings.TrimPrefix(line, "timestamp"))
			unixTime, err := strconv.ParseInt(timestamp, 10, 64)
			if err != nil {
				return nil, err
			}
			c.Timestamp = time.Unix(unixTime, 0)
		}
	}
	if i < len(lines) {
		c.Message = strings.TrimSpace(lines[i])
	}
	c.Hash = commitHash

	return c, err
}

func GetCommitHistory() ([]Commit, error) {
	var commits []Commit

	repoPath, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("could not get current working directory: %w", err)
	}

	commitHash, err := getHEADCommit(repoPath)
	if err != nil {
		return nil, err
	}

	for len(commitHash) > 0 {

		commit, err := LoadCommit(repoPath, commitHash)

		if err == nil {
			commits = append(commits, *commit)
		}

		if len(commit.ParentIDs) > 0 {
			commitHash = commit.ParentIDs[0]
		} else {
			commitHash = ""
		}
	}

	return commits, nil
}

// DiffCommits compares the contents of two commits and
// Returns a map of filename -> diff text.
func DiffCommits(hash1, hash2 string) (map[string]string, error) {
	currDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	commit1, err := LoadCommit(currDir, hash1)
	if err != nil {
		return nil, err
	}
	commit2, err := LoadCommit(currDir, hash2)
	if err != nil {
		return nil, err
	}

	treeHash1 := commit1.TreeID
	treeHash2 := commit2.TreeID

	// filemaps for each tree
	filesA, err := buildFileMapFromTree(treeHash1)
	if err != nil {
		return nil, fmt.Errorf("failed to build filemap for commit %s: %w", hash1, err)
	}

	filesB, err := buildFileMapFromTree(treeHash2)
	if err != nil {
		return nil, fmt.Errorf("failed to build filemap for commit %s: %w", hash1, err)
	}

	diff := make(map[string]string)

	// get all unique paths
	allPaths := make(map[string]struct{})
	for path := range filesA {
		allPaths[path] = struct{}{}
	}
	for path := range filesB {
		allPaths[path] = struct{}{}
	}

	for path := range allPaths {
		hashA, inA := filesA[path]
		hashB, inB := filesB[path]

		switch {
		case inA && !inB:
			// file was removed
			oldContent, err := loadBlobContent(hashA)
			if err != nil {
				return nil, err
			}
			diff[path] = generateUnifiedDiff(path, oldContent, "")
		case !inA && inB:
			// file was added
			newContent, err := loadBlobContent(hashB)
			if err != nil {
				return nil, err
			}
			diff[path] = generateUnifiedDiff(path, "", newContent)
		case inA && inB && hashA != hashB:
			// file modified
			oldContent, err := loadBlobContent(hashA)
			if err != nil {
				return nil, err
			}
			newContent, err := loadBlobContent(hashB)
			if err != nil {
				return nil, err
			}
			d := generateUnifiedDiff(path, oldContent, newContent)
			if d != "" {
				diff[path] = d
			}
		default:
			// no change
		}
	}

	return diff, nil
}

// buildFileMapFromTree returns a map of filepath -> blobHash for all files
// under the given tree
func buildFileMapFromTree(treeHash string) (map[string]string, error) {
	result := make(map[string]string)
	err := walkTree("", treeHash, result)
	return result, err
}

// walkTree recursively reads the tree object and populates 'result' with
// filepath -> blobHash
func walkTree(prefix, treeHash string, result map[string]string) error {
	treePath := filepath.Join(config.REPO_DIR, config.OBJECTS_DIR, treeHash)
	data, err := os.ReadFile(treePath)
	if err != nil {
		return fmt.Errorf("failed to read tree object %s: %w", treeHash, err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 3)
		if len(parts) != 3 {
			return fmt.Errorf("malformed tree entry line: %s", line)
		}

		typ := parts[0]
		name := parts[1]
		hash := parts[2]

		fullPath := name
		if prefix == "" {
			fullPath = prefix + "/" + name
		}

		switch typ {
		case "tree":
			if err := walkTree(fullPath, hash, result); err != nil {
				return err
			}
		case "blob":
			result[fullPath] = hash
		default:
			return fmt.Errorf("unknown type %s in tree %s", typ, treeHash)
		}
	}
	return nil
}

// loadBlobContent reads blob content from object store
func loadBlobContent(blobHash string) (string, error) {
	blobPath := filepath.Join(config.REPO_DIR, config.OBJECTS_DIR, blobHash)
	data, err := os.ReadFile(blobPath)
	if err != nil {
		return "", fmt.Errorf("failed to read blob %s: %w", blobHash, err)
	}
	return string(data), nil
	// return blobPath, nil
}

func generateUnifiedDiff(filename, oldContentPath, newContentPath string) string {
	// compute line mode diffing
	dmp := diffmatchpatch.New()
	runes1, runes2, lineArray := dmp.DiffLinesToRunes(oldContentPath, newContentPath)
	diffs := dmp.DiffMainRunes(runes1, runes2, false)
	diffs = dmp.DiffCharsToLines(diffs, lineArray)

	// format []diffmatchpatch.Diff to Unified format
	lineDiffs := difffmt.MakeLineDiffsFromDMP(diffs)
	hunks := difffmt.MakeHunks(lineDiffs, 3)
	unifiedFmt := difffmt.NewUnifiedFormat(difffmt.UnifiedFormatOption{
		ColorMode: difffmt.ColorTerminalOnly,
	})

	unified := unifiedFmt.Sprint(
		&difffmt.DiffTarget{Path: filename},
		&difffmt.DiffTarget{Path: filename},
		hunks,
	)

	return unified
}
