package internal

import (
	"bufio"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func LoadIgnorePatterns() ([]string, error) {
	file, err := os.Open(".jitignore")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []string{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var patterns []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// ignore empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return patterns, nil
}

func IsIgnonored(path string, patterns []string) bool {
	relPath, err := filepath.Rel(".", path)
	if err != nil {
		return false
	}

	for _, pattern := range patterns {
		if strings.HasPrefix(pattern, "/") {
			if strings.HasPrefix(relPath, strings.TrimPrefix(pattern, "/")) {
				return true
			}
		}

		// wildcards matching
		matched, _ := filepath.Match(pattern, filepath.Base(relPath))
		if matched {
			return true
		}

		// specific file paths
		if relPath == pattern {
			return true
		}
	}

	return false
}
