# `ray clean`

Find and remove junk files, build caches, and heavy folders to free up disk space.

---

## What It Does

`ray clean` scans your project for things taking up unnecessary disk space. It identifies:
- **Build cache folders**: Such as `node_modules`, `target/`, `dist/`, `.next/`, `build/`, and `__pycache__/`.
- **System junk and log files**: Such as `.DS_Store`, `Thumbs.db`, `npm-debug.log`, and `*.log` files.
- **Large individual files**: Files exceeding your chosen size limit (default: 10 MB).
- **Docker clutter (optional)**: Unused Docker images, dangling volumes, and build cache.

You can preview the space you will recover and interactively choose exactly what to delete.

---

## When to Use It

- When your computer or project drive is running low on disk space.
- When you want to clean a project before zipping, archiving, or sharing it.
- When build caches become corrupted and you need a fresh start.

---

## How to Run

```bash
# Interactively find and review cleanup targets
ray clean

# Just list files and folders that can be cleaned without deleting anything
ray clean --list

# Change the minimum file size limit to 25 MB
ray clean --threshold 25

# Clean everything immediately without asking for confirmation
ray clean --force

# Also clean unused Docker images and build caches
ray clean --docker

# Keep specific folders safe while cleaning everything else
ray clean --skip node_modules
```

---

## Flags and Options

| Flag | Shorthand | Default | Description |
| :--- | :--- | :--- | :--- |
| `--list` | `-l` | `false` | Shows what can be cleaned with sizes without asking to delete anything. |
| `--threshold` | `-t` | `10` | Minimum size in megabytes (MB) for large files and folders to be considered. |
| `--force` | `-f` | `false` | Deletes all discovered junk items immediately without waiting for confirmation. |
| `--docker` | `-d` | `false` | Checks for and cleans dangling Docker images and temporary build cache. |
| `--skip` | `-s` | `[]` | Folders or files you want to protect and leave untouched. |
| `--help` | `-h` | | Shows help and usage information for this command. |

---

## Sample Output

### Previewing with `--list`

Running `ray clean --list`:

```text
╭─────────────────────────────────────────────────────────────╮
│                     CLEANUP TARGETS DISCOVERED              │
├───────────────────────┬──────────────┬──────────────────────┤
│ Target Name / Path    │ Category     │ Size                 │
├───────────────────────┼──────────────┼──────────────────────┤
│ ./frontend/node_modules│ Junk Dir    │ 184.20 MB            │
│ ./frontend/.next      │ Junk Dir     │ 42.15 MB             │
│ ./backend/build       │ Junk Dir     │ 28.50 MB             │
│ ./app.log             │ Junk File    │ 12.40 MB             │
│ ./dump.sql            │ Heavy File   │ 115.00 MB            │
╰───────────────────────┴──────────────┴──────────────────────╯
╭─────────────────────────────────────────────────────────────╮
│                     CLEANUP SUMMARY                         │
├─────────────────────────────────────────────────────────────┤
│ Total Items: 5                                              │
│ Reclaimable Space: 382.25 MB                                │
╰─────────────────────────────────────────────────────────────╯
```

### Interactive Cleanup Prompt

When running without `--list`, Ray displays the items and asks:

```text
Enter items to delete (e.g. 1, 3-5, 'a' for all, or 'q' to cancel):
```

You can type:
- `a` to delete all detected junk.
- `1, 3` to delete only items 1 and 3.
- `q` to safely exit without touching any files.

---

## Helpful Tips

- **Safe by default**: Ray never deletes anything without your confirmation unless you explicitly provide the `--force` flag.
- **Docker integration**: Passing `--docker` is a great way to recover gigabytes of space left behind by old Docker builds and untagged images.

