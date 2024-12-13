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

Initialize a new JIT repository:

```bash
jit init
```

Add files to the repository:

```bash
jit add <file>
```

Commit changes:

```bash
jit commit -m "Your commit message"
```

View commit history:

```bash
jit log
```

Revert to a previous commit:

```bash
jit checkout <commit_id>
```

## Contributing

Contributions are welcome! Please fork the repository and submit a pull request.


## Contact

For any questions or feedback, please contact [mwaurambugua12@gmail.com](mailto:mwaurambugua12@gmail.com).