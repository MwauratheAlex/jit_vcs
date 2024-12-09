package internal

import (
	"fmt"
	"jit_vcs/config"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Commit struct {
	Message   string
	Timestamp time.Time
	TreeID    string
	ParentIDs []string
}

func (c *Commit) Serialize() []byte {
	// format
	// tree <TreeID>
	// parent <ParentID 1>
	// parent <ParentID 2>
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

func LoadCommit(hash string) (*Commit, error) {
	data, err := os.ReadFile(filepath.Join(
		config.REPO_DIR, config.OBJECTS_DIR, hash))
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var c Commit

	var i int
	for ; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			// blank line, rest is message
			i++
			break
		}

		switch {
		case strings.HasPrefix(line, "tree"):
			c.TreeID = strings.TrimPrefix(line, "tree")
		case strings.HasPrefix(line, "parent"):
			parentID := strings.TrimPrefix(line, "parent")
			c.ParentIDs = append(c.ParentIDs, parentID)
		case strings.HasPrefix(line, "timestamp"):
			tsStr := strings.TrimPrefix(line, "timestamp")
			unixTime, err := strconv.ParseInt(tsStr, 10, 64)
			if err != nil {
				return nil, err
			}
			c.Timestamp = time.Unix(unixTime, 0)
		case strings.HasPrefix(line, "tree"):
			c.TreeID = strings.TrimPrefix(line, "tree")
		case strings.HasPrefix(line, "tree"):
			c.TreeID = strings.TrimPrefix(line, "tree")
		}
	}

	if i < len(lines) {
		c.Message = strings.Join(lines[i:], "\n")
	}

	return &c, err
}