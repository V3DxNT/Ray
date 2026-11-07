# `ray audit`

Scan your codebase for leaked secrets, standard files, and code metrics.

---

## What It Does

`ray audit` acts as a security and code quality checker for your project. In a single fast scan, it:
1. **Detects leaked secrets**: Searches your source code for accidentally committed API keys, passwords, private tokens, AWS credentials, database connection strings, Stripe keys, OpenAI keys, and Slack webhooks, automatically masking sensitive values in the report.
2. **Checks standard repository files**: Verifies whether essential open-source and team files are present (such as `README.md`, `LICENSE`, `.gitignore`, and `SECURITY.md`).
3. **Counts code metrics**: Calculates lines of code, comments, blank lines, and tallies syntax elements like functions, classes, interfaces, and imports across all supported languages.

---

## When to Use It

- Before pushing code or creating a public repository to make sure no credentials are leaked.
- In CI/CD pipelines to automatically fail builds if secrets are introduced (`--strict`).
- When preparing an open-source project and verifying required documentation files exist.
- When generating a high-level summary of code size and structural components.

---

## How to Run

```bash
# Run a full audit of your current project
ray audit

# Fail with an error code if any secrets are discovered (ideal for CI/CD)
ray audit --strict

# Only check for leaked secrets and skip code metrics
ray audit --secrets-only

# Only calculate code metrics (lines, functions, classes)
ray audit --metrics-only

# Also check if environment variables used in code exist in .env
ray audit --env

# Output audit results in structured JSON
ray audit --json

# Skip test fixture folders
ray audit --skip testdata,fixtures
```

---

## Flags and Options

| Flag | Shorthand | Default | Description |
| :--- | :--- | :--- | :--- |
| `--strict` | | `false` | Exits with an error code (exit status 1) if any secrets are found. |
| `--secrets-only` | | `false` | Runs only the secret detection scanner, skipping metrics and standard files. |
| `--metrics-only` | | `false` | Runs only the syntax metrics scanner, skipping secrets. |
| `--env` | | `false` | Also audits environment variable usage against `.env` files. |
| `--json` | | `false` | Outputs the complete audit findings in JSON format. |
| `--skip` | `-s` | `[]` | Folders or files to ignore during the audit. |
| `--help` | `-h` | | Shows help and usage information for this command. |

---

## Sample Output

### Standard Security & Health Audit

Running `ray audit` on a project:

```text
╭─────────────────────────────────────────────────────────────╮
│                    STANDARD REPOSITORY FILES                │
├───────────────────────┬────────────┬────────────────────────┤
│ Standard File         │ Status     │ Priority               │
├───────────────────────┼────────────┼────────────────────────┤
│ README.md             │ ✓ Present  │ Required               │
│ LICENSE               │ ✓ Present  │ Required               │
│ .gitignore            │ ✓ Present  │ Recommended            │
│ CONTRIBUTING.md       │ ✗ Missing  │ Optional               │
│ SECURITY.md           │ ✗ Missing  │ Recommended            │
╰───────────────────────┴────────────┴────────────────────────╯
╭─────────────────────────────────────────────────────────────╮
│                      SECURITY FINDINGS                      │
├─────────────────────────────────────────────────────────────┤
│ ⚠️  Exposed Secrets Discovered (1):                          │
│                                                             │
│ • AWS Access Key ID                                         │
│   File: backend/config/keys.go:12                           │
│   Value: AKIA****************                               │
╰─────────────────────────────────────────────────────────────╯
╭─────────────────────────────────────────────────────────────╮
│                       CODE SYNTAX METRICS                   │
├───────────────────────┬─────────────────────────────────────┤
│ Total Source Files    │ 46                                  │
│ Total Lines of Code   │ 3,420                               │
│ Functions / Methods   │ 118                                 │
│ Classes / Structs     │ 24                                  │
│ Interfaces            │ 6                                   │
│ Comment Lines         │ 280                                 │
│ Blank Lines           │ 410                                 │
╰───────────────────────┴─────────────────────────────────────╯
```

---

## Helpful Tips

- **Safe terminal display**: Secret values are automatically masked (e.g. `AKIA****************`) so sensitive credentials are never displayed in full in terminal logs or recordings.
- **CI/CD Integration**: Add `ray audit --strict` as a pre-commit hook or GitHub Actions step to prevent private keys or database passwords from ever reaching your git history.

