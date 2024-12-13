# JIT Version Control System

JIT is a simple version control system implemented in Go. It allows you to track changes to your files and collaborate with others.

## Features

- Track changes to files
- Commit changes with messages
- View commit history
- Revert to previous versions

## Installation


1. Clone the repository
```bash
git clone https://github.com/MwauratheAlex/jit_vcs.git
cd jit_vcs
```

2. Resolve dependancies
```bash
go mod tidy
```

3. a. Install jit globally(requires go installed)
```bash
go install 
```
- ensure your $PATH includes go binaries directory (default is $HOME/go/bin or $GOBIN)
You can verify this with:
```bash
echo $PATH
```
- if the directory is not in your $PATH, add it. 
Example in bash
```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

3. b. Alternatively, build the JIT binary
```bash
go build -o jit main.go
```

- this will create an executable jit binary in the current directory
 - Copy it to a directory in your $PATH to use globally
```bash
cp jit /usr/local/bin/
```

or Run it directly from any directory.
```bash
./jit
```

## Testing
```bash
go test ./command
```


## Usage

### Initialize a new JIT repository:

```bash
jit init
```

### Add files to the repository:

```bash
jit add <file>
```

### Commit changes:

```bash
jit commit -m "Your commit message"
```

### View commit history:

```bash
jit log
```

### View commit diffs

```bash
jit diff <old-commit-hash> <new-commit-hash>
```

### Create new branch

```bash
jit branch <branch-name>
```

### View Branches

```bash
jit branch
```

### Switch branch:

```bash
jit checkout <branch-name>
```

### Merge Branch

```bash
jit merge <branch-name>
```

### Clone Repo
```bash
jit clone <repo-to-clone> <destination-folder>
```

### Handling Merge Conflicts
- incase of merge conflicts, conflict markers are added to the file
- Conflict resolution is not implemented yet
- Example file with conflicts after merge

```bash
# jit testing

Here I'm just testing JIT.
Two lines added in readme

one more commit

one last commit
<<<<<<< HEAD
One more thing
add in master now
=======
One more thing
>>>>>>> target_branch
```

### Ignoring files

Create a ```bash .jitignore``` file and list the full paths of the files you want to ignore

Example ```bash .jitignore```

```bash .jitignore
git.c
secret.txt
```

- This files will be automatically ignored during jit operations.
- jit also ingores .jit folder and .jitignore itself.
- this means that the .jitignore folder will not be copied during cloning.
and in other relevant operations like ```bash jit add <file> ...```


## Issues
- Refactoring
- Needs more comprehensive testing


## Contributing

Contributions are welcome! Please fork the repository and submit a pull request.


## Contact

For any questions or feedback, please contact [mwaurambugua12@gmail.com](mailto:mwaurambugua12@gmail.com).
