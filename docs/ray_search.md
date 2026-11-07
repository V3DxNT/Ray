# `ray search`

Quickly find files anywhere in your project by name.

---

## What It Does

`ray search` helps you locate files across your entire project. If you remember the exact name, it finds where it lives. If you only remember part of the name or made a slight typo, it also finds similar matching files so you never lose track of a file.

---

## When to Use It

- When you know a file exists in a project but forgot which subfolder it is in.
- When searching for multiple files at once across different directories.
- When you want to see all files matching a specific prefix or keyword (like `config` or `test`).

---

## How to Run

```bash
# Find a specific file
ray search config.json

# Search for multiple files in a single run
ray search main.go utils.py database.ts

# Stop searching as soon as the first exact match is found
ray search --first app.js

# Show all similar matches without truncating the list
ray search --all schema

# Skip specific folders during the search
ray search --skip vendor,build .env
```

---

## Flags and Options

| Flag | Shorthand | Default | Description |
| :--- | :--- | :--- | :--- |
| `--first` | `-f` | `false` | Stops searching immediately after finding the first exact match. |
| `--all` | `-a` | `false` | Shows all partial and similar matches instead of cutting off the list. |
| `--skip` | `-s` | `[]` | A comma-separated list of folders or files to skip while searching. |
| `--help` | `-h` | | Shows help and usage information for this command. |

---

## Sample Output

### Searching for a File

Running `ray search main.go`:

```text
╭─────────────────────────────────────────────────────────────╮
│                        SEARCH RESULTS                       │
├─────────────────────────────────────────────────────────────┤
│ 🎯 Target: main.go                                          │
├─────────────────────────────────────────────────────────────┤
│ Exact Matches Found (1):                                    │
│  ✓ ./backend/cmd/api/main.go                                │
│                                                             │
│ Similar Matches (2):                                        │
│  ~ ./backend/main_test.go                                   │
│  ~ ./scripts/main_helper.go                                 │
╰─────────────────────────────────────────────────────────────╯
```

### Searching for Multiple Files

Running `ray search config.ts .env`:

```text
╭─────────────────────────────────────────────────────────────╮
│                        SEARCH RESULTS                       │
├─────────────────────────────────────────────────────────────┤
│ 🎯 Target: config.ts, .env                                  │
├─────────────────────────────────────────────────────────────┤
│ Exact Matches Found:                                        │
│  ✓ ./frontend/src/config/config.ts                          │
│  ✓ ./.env                                                   │
│  ✓ ./backend/.env                                           │
╰─────────────────────────────────────────────────────────────╯
```

---

## Helpful Tips

- **Search multiple targets at once**: You can pass as many filenames as you need: `ray search package.json tsconfig.json .eslintrc.json`.
- **First match speedup**: If you just need to know whether a file exists and where the first instance is, use `--first` for faster response times.

