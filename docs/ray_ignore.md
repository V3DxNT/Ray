# `ray ignore`

Generate `.gitignore` and `.dockerignore` files tailored to the languages in your project.

---

## What It Does

`ray ignore` detects the technologies in your codebase and creates ready-to-use `.gitignore` and `.dockerignore` files using built-in, industry-standard templates.

If your project is a monorepo that uses multiple languages (for example, a Go backend alongside a React or Next.js frontend), Ray combines the ignore rules into a unified file so you don't have to assemble them by hand.

It also generates an aggressive `.dockerignore` to make sure your Docker build contexts don't balloon with `.git`, `node_modules`, or local `.env` secrets.

---

## When to Use It

- When starting a new project and wanting a comprehensive `.gitignore` right away.
- When creating a Dockerfile and needing a `.dockerignore` so builds are fast and small.
- When combining multiple programming stacks into one repository.

---

## How to Run

```bash
# Auto-detect stacks and generate both .gitignore and .dockerignore
ray ignore

# Only generate .gitignore
ray ignore --git

# Only generate .dockerignore
ray ignore --docker

# Manually specify languages instead of auto-detecting
ray ignore --stack go,node,python

# Preview what rules will be created without touching any files
ray ignore --dry-run

# Overwrite existing files without asking
ray ignore --force

# Add new rules to the bottom of existing files instead of replacing them
ray ignore --append
```

---

## Flags and Options

| Flag | Shorthand | Default | Description |
| :--- | :--- | :--- | :--- |
| `--git` | | `false` | Only generates `.gitignore`, skipping `.dockerignore`. |
| `--docker` | | `false` | Only generates `.dockerignore`, skipping `.gitignore`. |
| `--stack` | `-s` | `[]` | Specific languages to include (e.g. `go`, `node`, `python`, `rust`, `java`, `cpp`). |
| `--force` | `-f` | `false` | Overwrites existing files without asking for confirmation. |
| `--append` | `-a` | `false` | Appends generated rules to the end of existing ignore files. |
| `--dry-run` | | `false` | Prints the generated ignore rules to the terminal without writing them. |
| `--help` | `-h` | | Shows help and usage information for this command. |

---

## Sample Output

### Generating Ignore Files

Running `ray ignore`:

```text
🔍 Analyzing codebase stacks...

╭─────────────────────────────────────────────────────────────╮
│                   DETECTED CODEBASE STACKS                  │
├───────────────────────┬─────────────────────────────────────┤
│ Primary Stacks        │ Node.js, Go                         │
│ Target Directory      │ .                                   │
╰───────────────────────┴─────────────────────────────────────╯

Generate .gitignore (48 rules) and .dockerignore (24 rules)? [Y/n]: y

✓ Written .gitignore (combined Go + Node rules)
✓ Written .dockerignore (optimized Docker build context)
```

### Previewing with `--dry-run`

Running `ray ignore --git --dry-run`:

```text
[Dry Run] .gitignore preview:

# Base System
.DS_Store
Thumbs.db
*.log

# Node.js
node_modules/
.next/
dist/
npm-debug.log*

# Go
bin/
*.exe
vendor/
```

---

## Helpful Tips

- **Combined rules for monorepos**: In a repository containing both Go and TypeScript, `ray ignore` merges both rule sets together without duplicate entries.
- **Docker context safety**: The generated `.dockerignore` automatically excludes `.git`, `.env`, and build folders, ensuring private API keys and unnecessary gigabytes of cache are never sent to your Docker daemon.

