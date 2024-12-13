# JIT Version Control System

JIT is a simple version control system implemented in Go. It allows you to track changes to your files and collaborate with others.

## Features

- Track changes to files
- Commit changes with messages
- View commit history
- Revert to previous versions

## Installation

To install JIT, clone the repository and install the required dependencies:

```bash
git clone https://github.com/MwauratheAlex/jit_vcs.git
cd jit_vcs
go install # install the app in your system
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
README.md
new_dir/Hello.py
```

- This files will be automatically ignored during jit operations


## Issues
- Ignoring files is not yet implemented
- Directories can cause issues


## Contributing

Contributions are welcome! Please fork the repository and submit a pull request.


## Contact

For any questions or feedback, please contact [mwaurambugua12@gmail.com](mailto:mwaurambugua12@gmail.com).
