package command

import (
	"fmt"
	"jit/internal"
)

func Add(paths []string) error {
	patterns, err := internal.LoadIgnorePatterns()
	if err != nil {
		return fmt.Errorf("failed to load .jitignore: %w", err)
	}

	if len(paths) < 1 {
		return fmt.Errorf("%sNo file specified.%s\nUsage: jit add <file1> <file2> ...",
			colorRed, colorNone,
		)
	}

	for _, path := range paths {
		if internal.IsIgnonored(path, patterns) {
			fmt.Printf("Skipping ingored file: %s\n", path)
			continue
		}
		if err := internal.AddToIndex(path); err != nil {
			return err
		}
		fmt.Printf("Added '%s' to staging area.\n", path)
	}

	return nil
}
