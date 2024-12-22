# Jit VCS - Documentation
Jit VCS is a simple version control system implemented in golang.
The system has 3 packages:
- main package - the entry point to the application.
- command package - handles command line arguements validation and calls the appropriate functions in 
    the internal package based on the arguements
- internal package - this handles all the internal operations of the system such as creating
    commits and staging files.

# Entry Point
## main.go
The main function calls `command.Execute()` and checks the returned error value.
The error is logged if it exists and the system exits.

## Design choices
The system is designed to funnel all the errors back to main.
This way, we have a central location for error handling,
and we avoid having `os.Exit(1)` function calls scattered all over our system.

# Command Package
## /command
### /command/execute.go
`func Execute() error`
This is a dispach function.
It validates the command line arguements and returns the appropriate function based on 
the arguements.

We will now cover each command flow.

 ## Initializing a new Jit Repository - `jit init`
`command/init.go`
The function `func Init(args []string) error` initializes a new repository by performing the following steps:
 1. ### Create `.jit` directory
This is the root directory of the repository.

 2. ### Create `.jit/refs` directory
 This directory will store the `heads` subdirectory, which will contain pointers to the latest commit for each branch.

 3. ### Create `.jit/objects` directory
 This directory is where all objects (blobs, trees and commits) will be stored.

 4. ### Create `./jit/HEAD` file
This file will store a pointer to the current branch and is initialized with the following content:
`ref: refs/heads/master`
It points to `master` branch in the `refs/heads` directory.

 - The `refs/heads` is empty during initialization.

## Adding files to the staging area - `jit add <file1> <file2> ...`
`command/add.go`
The function `func Add(paths []string) error` adds files to the staging area(index).
It performs the following steps:
1. ### Load Ignore Patterns
The function calls `internal.LoadIgnorePatterns` which reads ignore patters 
located in the `.jitignore` file.

#### Loading Ignore Patterns
`internal/ignore.go`
The function `func LoadIgnorePatterns() ([]string, error)` reads the `.jitignore`
file and returns an array of all the patterns  to be ignored.
- The files `.jitignore` and `.jit` are ignored by default.

2. ### Input Validation
An error is returned if:
 - no files are specified (`len(paths) < 1`)
 - if `path` is `"."` (`len(paths) < 1`)

3. ### Loop through each path
For each file path in `paths`, the function:
 - calls `internal.IsIgnored`, which checks whether the path matches any patterns in `.jitignore`
   - if a path is ignored, it is skipped and a log is displayed.
 - calls `internal.AddToIndex(path)` to add the file to the index.
 - logs a confirmation message showing success.

#### Checking if `path` is ignored
`internal/ignore.go`
The function `func IsIgnonored(path string, patterns []string) bool` returns true if a path
matches any of the patterns and false if not.

#### Adding to `file` index
`internal/index.go`
- The function `func AddToIndex(path string) error` adds files to the staging area(index).
- If `path` is a directory:
    - If the directory is empty, the function returns nil (empty directories are not added to index)
    - If the directory contains files, the function recursively walks through the directory structure using `filepath.Walk`
      and calls `AddToIndex` on each file path.
- If `path` is a file:
    - The function reads the `file content` and computes a hash of the content. ie. `hash := ComputeHash(content)` 
    - It checks if the object named `hash` exists in the `.jit/objects` directory.
        - If the object does not exist, it creates the object by writing the `file content` to `.jit/objects/hash`
        - The check ensures that files with similar content are only stored once which saves on memory.
    - The index (`.jit/index`) file is then updated with the format: `<hash> <path>`.
    - A map of `path -> index entry` is used to ensure the index does not have duplicate paths.

#### Hashing
The function `func ComputeHash(data []byte) string` returns the `sha1` hash of data as a string.

4. After the loop is complete, all the non-ignored files have been added to the staging area.

### Potential Improvements
- It would be more efficient to load and update the index in memory and only save after all the operations are done.
- This would eliminate unnessessary disk read and write operations.


## Creating Commits - `jit commit -m <commit message>`
### command/commit.go
`func Commit(message string) error` 
This function calls `internal.CreateCommit` passing in the message and the current time.

## Branching 
### command/branch.go
`func Branch(name string) error`
This function calls `internal.CreateBranch` passing in the name of the branch.
`ListBranches`
This function calls `internal.listBranches()` to list all the created branches.

## Checking out a branch - `jit checkout <branch name>`
### command/checkout.go

## Logging Commit History - `jit log`
### command/log.go

## Diffing Commits - `jit diff <old commit hash> <new commit hash>`
### command/diff.go

## Clonig a repository - `jit clone <repo name> <new repo name>`
### command/clone.go

## Merging into a branch - `jit merge <branch name>`
### command/merge.go
