# `ray dedup`

Find identical duplicate files across your project and see how much space they waste.

---

## What It Does

`ray dedup` scans your files and computes an exact SHA-256 checksum for each one. If two or more files share the exact same contents—even if they have completely different names or live in separate folders—Ray detects them and calculates how much disk space you can save by removing the duplicates.

---

## When to Use It

- When you have copied asset files, images, or test fixtures into multiple directories.
- When organizing a large download folder, media directory, or archive.
- When auditing a repository to ensure you aren't storing the same big binary multiple times.

---

## How to Run

```bash
# Check your current folder for duplicate files
ray dedup

# Check a specific directory
ray dedup ./assets

# Only check files larger than 1 megabyte (1048576 bytes)
ray dedup --min-size 1048576

# Skip specific folders during checking
ray dedup --skip vendor,build

# Output the duplicate report as structured JSON
ray dedup --json
```

---

## Flags and Options

| Flag | Shorthand | Default | Description |
| :--- | :--- | :--- | :--- |
| `--min-size` | `-m` | `1` | Smallest file size in bytes to include in the duplicate check. |
| `--skip` | `-s` | `[]` | A comma-separated list of folders or files to ignore. |
| `--json` | | `false` | Outputs duplicate groups and checksums in JSON format. |
| `--help` | `-h` | | Shows help and usage information for this command. |

---

## Sample Output

### Duplicate Files Report

Running `ray dedup` in a project with duplicated assets:

```text
╭─────────────────────────────────────────────────────────────╮
│                     DUPLICATE FILES FOUND                   │
├─────────────────────────────────────────────────────────────┤
│ 📦 Group 1 (Size: 2.40 MB per copy)                         │
│   Hash: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca4  │
│   • ./public/images/hero-banner.png                         │
│   • ./src/assets/banner_backup.png                          │
│   • ./docs/images/hero.png                                  │
│                                                             │
│ 📦 Group 2 (Size: 18.50 KB per copy)                        │
│   Hash: a591a6d40bf420404a011733cfb7b190d62c65bf0bcda32b57  │
│   • ./backend/config/sample.json                            │
│   • ./frontend/mock/sample.json                             │
╰─────────────────────────────────────────────────────────────╯
╭─────────────────────────────────────────────────────────────╮
│                     RECLAIMABLE SUMMARY                     │
├─────────────────────────────────────────────────────────────┤
│ Total Scanned Files: 84                                     │
│ Duplicate Sets Found: 2                                     │
│ Potential Space Recovered: 4.81 MB                          │
╰─────────────────────────────────────────────────────────────╯
```

### JSON Output (`--json`)

```json
{
  "target_directory": "/Users/user/project",
  "total_files_scanned": 84,
  "reclaimable_bytes": 5046272,
  "reclaimable_formatted": "4.81 MB",
  "duplicate_groups": [
    {
      "sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
      "file_size": 2516582,
      "file_size_formatted": "2.40 MB",
      "files": [
        "./public/images/hero-banner.png",
        "./src/assets/banner_backup.png",
        "./docs/images/hero.png"
      ]
    }
  ]
}
```

---

## Helpful Tips

- **Content-based matching**: Ray does not rely on filenames or file modification dates. It computes exact cryptographic hashes, so even if files were renamed, duplicates are reliably caught.
- **Save time on big repos**: Use `--min-size` (e.g. `--min-size 102400` for 100 KB) if you only care about large assets and want to skip small files.

