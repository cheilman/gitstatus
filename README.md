# gitstatus

Return status of a git repository, suitable for prompt interpretation.

## Version Control Systems

This application currently supports git and mercurial (hg) repositories.  You
can choose what repo you would like the status of:

### --vcs=detect (default)

Figure out which repository among the ones supported applies to the directory.

## Output Formats

### --output=full (default)

The default output will be a JSON blob containing the following:

- Identifying this path as part of a git repository
- The git root
- Changed Files
    - A set of characters indicating different changes, and a count of files
    affected.
    - 'M' -- modified
    - 'A' -- added
    - 'D' -- deleted
    - 'R' -- renamed
    - 'C' -- copied
    - 'U' -- updated
    - '?' -- untracked
    - '!' -- ignored
- The current branch name
    - Plain text
    - Colored according to branch status
- Non-active branches available locally
- Current branch tracking information

### --output=prompt

Two lines containing:

- The branches, colored appropriately, as: `git:<master>`
- The change counts, as: `M:1 -:1 ?:1`

### --output=statusline

Three lines containing:

- The repository name (pulled from root directory)
- Branch tracking information, as: `master...origin/master`
- Change counts, as: `M:1 -:1 ?:1`

## Options

### --dir=(path)

Directory to check for git repository.  Defaults to the working directory.

### --color

Forces color output.  The default is color enalbed when writing to interactive
ttys.

