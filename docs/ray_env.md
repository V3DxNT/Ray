# `ray env`

Find environment variables used in code and check whether they exist in your `.env` file.

---

## What It Does

`ray env` prevents runtime crashes caused by missing configuration. It walks your entire codebase and spots where environment variables are read in:
- **Node.js & Next.js**: `process.env.KEY`, `process.env["KEY"]`
- **Go**: `os.Getenv("KEY")`, `os.LookupEnv("KEY")`
- **Python**: `os.environ.get("KEY")`, `os.environ["KEY"]`, `os.getenv("KEY")`
- **Rust**: `env::var("KEY")`
- **Java**: `System.getenv("KEY")`
- **PHP**: `getenv("KEY")`, `$_ENV["KEY"]`
- **Ruby**: `ENV["KEY"]`

It then reads your `.env` file and alerts you to any variables used in your code that are missing from `.env`. It can even append them automatically!

---

## When to Use It

- Before deploying to production to ensure all required configuration keys are documented.
- When pulling changes from teammates that added new environment variable lookups.
- To auto-generate boilerplate entries in your `.env` file (`--fix`).

---

## How to Run

```bash
# Check current project against .env
ray env

# Check against a custom environment file (like .env.local or .env.example)
ray env --file .env.local

# Automatically add missing variables with blank placeholders to your .env file
ray env --fix

# Fail with an error code if missing variables are found (great for CI/CD)
ray env --strict

# Output the report in JSON format
ray env --json

# Skip test directories
ray env --skip tests,fixtures
```

---

## Flags and Options

| Flag | Shorthand | Default | Description |
| :--- | :--- | :--- | :--- |
| `--file` | `-f` | `.env` | Path to the environment file to check against. |
| `--fix` | | `false` | Automatically appends missing variables with blank values to your `.env` file. |
| `--strict` | `-s` | `false` | Exits with an error code (exit status 1) if any missing variables are found. |
| `--json` | | `false` | Outputs results in JSON format. |
| `--skip` | | `[]` | Additional folders or files to skip during the scan. |
| `--help` | `-h` | | Shows help and usage information for this command. |

---

## Sample Output

### Missing Variables Report

Running `ray env`:

```text
🔍 Checking environment references against: .env

╭─────────────────────────────────────────────────────────────╮
│                ENVIRONMENT AUDIT: MISSING KEYS              │
├─────────────────────────────────────────────────────────────┤
│ ❌ Missing from .env: JWT_SECRET                            │
│    Found in: backend/main.go:24                             │
│                                                             │
│ ❌ Missing from .env: STRIPE_KEY                            │
│    Found in: backend/handlers/data.go:18                    │
│                                                             │
│ ❌ Missing from .env: REDIS_URL                             │
│    Found in: services/worker/worker.py:11                   │
╰─────────────────────────────────────────────────────────────╯
╭─────────────────────────────────────────────────────────────╮
│                     ENVIRONMENT SYNC SUMMARY                │
├─────────────────────────────────────────────────────────────┤
│ Defined in .env: 4 keys                                     │
│ Detected in Code: 7 keys                                    │
│ Missing from .env: 3 keys                                   │
│                                                             │
│ 💡 Run 'ray env --fix' to add missing placeholders to .env! │
╰─────────────────────────────────────────────────────────────╯
```

### Auto-Fixing with `--fix`

```bash
ray env --fix
```

```text
✓ Appended 3 missing variables to .env:
  + JWT_SECRET=
  + STRIPE_KEY=
  + REDIS_URL=
```

---

## Helpful Tips

- **One-command sync**: When you pull new code and wonder what `.env` keys you need, run `ray env --fix` to instantly append empty keys ready for you to fill in.
- **CI/CD Quality Gate**: Run `ray env --strict` in your pull request checks to prevent code from being merged with undocumented environment variables.

