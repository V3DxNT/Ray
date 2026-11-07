# `ray stats`

Get an instant dashboard showing languages, file counts, sizes, and frameworks in your project.

---

## What It Does

`ray stats` scans your project to give you a quick, clean summary of everything inside. It counts files by language, tallies total lines of code and folders, calculates the total project size, and automatically identifies which technologies, frameworks, and tools your project is built with.

---

## When to Use It

- When you want to understand what programming languages make up a project.
- When you want to know how big a repository is and how many files it contains.
- When working with a monorepo that contains multiple apps (like a frontend in React and a backend in Go).

---

## How to Run

```bash
# Analyze your current project folder
ray stats

# Analyze a specific directory
ray stats ./backend

# Show the exact paths of all detected apps and frameworks
ray stats --list

# Skip specific folders during analysis
ray stats --skip vendor,build
```

---

## Flags and Options

| Flag | Shorthand | Default | Description |
| :--- | :--- | :--- | :--- |
| `--list` | `-l` | `false` | Displays the exact folder locations of each detected app or framework. |
| `--skip` | `-s` | `[]` | A comma-separated list of folders or files to ignore during analysis. |
| `--help` | `-h` | | Shows help and usage information for this command. |

---

## Sample Output

### Standard Dashboard

Running `ray stats` in a project with a Go backend and a Next.js frontend:

```text
╭─────────────────────────────────────────────────────────────╮
│                      CODEBASE METRICS                       │
├───────────────────────┬─────────────────────────────────────┤
│ Total Files           │ 142                                 │
│ Total Folders         │ 28                                  │
│ Codebase Size         │ 4.82 MB                             │
│ Primary Language      │ TypeScript                          │
│ Detected Frameworks   │ Next.js, Go, Docker                 │
╰───────────────────────┴─────────────────────────────────────╯
╭─────────────────────────────────────────────────────────────╮
│                    LANGUAGE BREAKDOWN                       │
├───────────────────────┬────────────┬────────────────────────┤
│ Language / Type       │ Files      │ Percentage             │
├───────────────────────┼────────────┼────────────────────────┤
│ .ts / .tsx            │ 84         │ █████████████ 59.2%    │
│ .go                   │ 32         │ █████ 22.5%            │
│ .json                 │ 14         │ ██ 9.9%                │
│ .md                   │ 8          │ █ 5.6%                 │
│ .yaml / .yml          │ 4          │ █ 2.8%                 │
╰───────────────────────┴────────────┴────────────────────────╯
```

### Listing Detected Subprojects

Running `ray stats --list`:

```text
╭─────────────────────────────────────────────────────────────╮
│                    DETECTED SUBPROJECTS                     │
├────────────────────────────────┬────────────────────────────┤
│ Framework / Stack              │ Location                   │
├────────────────────────────────┼────────────────────────────┤
│ Next.js (Node.js)              │ ./frontend                 │
│ Go Modules                     │ ./backend                  │
│ Docker Container               │ ./docker-compose.yml       │
╰────────────────────────────────┴────────────────────────────╯
```

---

## Helpful Tips

- **Monorepos made easy**: Ray recursively checks subfolders to uncover nested projects, making it simple to see all microservices or packages at a glance.
- **Fast scanning**: The scanner uses worker threads to read files concurrently, so even projects with thousands of files complete in seconds.

