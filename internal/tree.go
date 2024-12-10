package internal

import (
	"bytes"
	"fmt"
	"jit_vcs/config"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type TreeEntry struct {
	Mode string // filemode. eg. 0644, 0755, 040000
	Type string // blob or tree
	Name string
	Hash string // hash of blob or subtree
}

type Tree struct {
	Hash    string
	Entries []TreeEntry
}

// BuildTreeFromFiles builds a tree from index of tree.
// Recurisively constructs subtrees for directories
// Returns top-level tree object
func BuildTreeFromFiles(files *Index) (*Tree, error) {
	// group Entries by parent directory
	rootEntriesMap := make(map[string][]IndexEntry)
	blobEntries := []TreeEntry{}

	for _, f := range *files {
		cleanPath := filepath.ToSlash(f.Filepath)
		parts := strings.SplitN(cleanPath, "/", 2)

		if len(parts) == 1 {
			// in root
			blobEntries = append(blobEntries, TreeEntry{
				Mode: fmt.Sprintf("%04o", f.Mode.Perm()),
				Type: "blob",
				Name: parts[0],
				Hash: f.Hash,
			})
		} else {
			// in a subdirectory
			dirName := parts[0]
			remainingPath := parts[1]

			rootEntriesMap[dirName] = append(rootEntriesMap[dirName], IndexEntry{
				Filepath: remainingPath,
				Hash:     f.Hash,
				Mode:     f.Mode,
			})
		}
	}

	// build subtree for each directory
	for dirName, dirFiles := range rootEntriesMap {
		subIndex := Index(dirFiles)
		subTree, err := BuildTreeFromFiles(&subIndex)
		if err != nil {
			return nil, err
		}
		blobEntries = append(blobEntries, TreeEntry{
			Mode: "040000", // dir
			Type: "tree",
			Name: dirName,
			Hash: subTree.Hash,
		})
	}

	// sort entries for consistent hashing
	sort.Slice(blobEntries, func(i, j int) bool {
		return blobEntries[i].Name < blobEntries[i].Name
	})

	// tree data
	var buf bytes.Buffer
	for _, e := range blobEntries {
		fmt.Fprintf(&buf, "%s %s %s\n", e.Mode, e.Type, e.Name)
	}

	treeHash := ComputeHash(buf.Bytes())

	t := &Tree{
		Hash:    treeHash,
		Entries: blobEntries,
	}

	if err := t.Save(); err != nil {
		return nil, err
	}

	return t, nil
}

// Save writes the tree object to the objects directory
func (t *Tree) Save() error {
	var sb strings.Builder
	for _, e := range t.Entries {
		sb.WriteString(fmt.Sprintf("%s %s %s\n", e.Mode, e.Type, e.Name))
		sb.WriteString(fmt.Sprintf("%s\n", e.Hash))
	}
	return os.WriteFile(
		filepath.Join(config.REPO_DIR, config.OBJECTS_DIR, t.Hash),
		[]byte(sb.String()), 0644,
	)
}
