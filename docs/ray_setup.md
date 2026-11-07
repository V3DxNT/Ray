# `ray setup`

Install missing system tools and bootstrap project package dependencies automatically.

---

## What It Does

`ray setup` takes the friction out of onboarding. It inspects your project, detects what tools are missing on your machine (like Node.js, Go, Python, or Docker), calculates the estimated disk space needed, and offers to install them in one batch using your operating system's package manager:
- **macOS**: Uses Homebrew (`brew`)
- **Linux**: Uses `apt` (or package manager)
- **Windows**: Uses `winget` or `choco`

Once system tools are verified, it can also run package managers for your project components (such as `npm install`, `pip install -r requirements.txt`, or `go mod download`).

If your system lacks a package manager (such as Homebrew on a clean Mac), Ray alerts you with the exact command to install it first.

---

## When to Use It

- When setting up a new computer or virtual environment.
- When you clone a repo and want to get it running with a single command.
- When running in automated setup scripts (`--yes`).

---

## How to Run

```bash
# Interactively inspect and install missing prerequisites
ray setup

# Preview what would be installed without making any changes
ray setup --dry-run

# Automatically answer yes to all prompts
ray setup --yes

# Only install system tools (like Go or Node), skipping package dependencies
ray setup --system-only

# Only install project package dependencies (like npm or pip), skipping system tools
ray setup --deps-only

# Output setup plan and results in JSON format
ray setup --json
```

---

## Flags and Options

| Flag | Shorthand | Default | Description |
| :--- | :--- | :--- | :--- |
| `--yes` | `-y` | `false` | Automatically confirms all installation prompts without pausing. |
| `--dry-run` | | `false` | Shows the full installation plan and required space without executing installs. |
| `--system-only` | | `false` | Installs only missing system tools and compilers, skipping project dependencies. |
| `--deps-only` | | `false` | Installs only project package dependencies, skipping system toolchains. |
| `--json` | | `false` | Outputs the setup plan and results in JSON format. |
| `--help` | `-h` | | Shows help and usage information for this command. |

---

## Sample Output

### Setup Execution Plan

Running `ray setup`:

```text
╭─────────────────────────────────────────────────────────────╮
│                    RAY SETUP EXECUTION PLAN                 │
├───────────────────────┬─────────────────────────────────────┤
│ Operating System      │ darwin (macOS)                      │
│ Package Manager       │ Homebrew (brew)                     │
│ Required Disk Space   │ ~140 MB                             │
╰───────────────────────┴─────────────────────────────────────╯
╭─────────────────────────────────────────────────────────────╮
│                     MISSING SYSTEM TOOLS                    │
├───────────────────────┬──────────────┬──────────────────────┤
│ Tool / Component      │ Estimated MB │ Installation Command │
├───────────────────────┼──────────────┼──────────────────────┤
│ Go Compiler           │ ~140 MB      │ brew install go      │
╰───────────────────────┴──────────────┴──────────────────────╯
╭─────────────────────────────────────────────────────────────╮
│                PROJECT DEPENDENCY TASKS                     │
├───────────────────────┬──────────────┬──────────────────────┤
│ Subproject Path       │ Manager      │ Setup Command        │
├───────────────────────┼──────────────┼──────────────────────┤
│ ./frontend            │ npm          │ npm install          │
│ ./services/worker     │ pip          │ pip install -r reqs  │
╰───────────────────────┴──────────────┴──────────────────────╯

Install missing system tools and run project dependency setups? [y/N]: y
```

### Installation Summary

After confirming:

```text
╭─────────────────────────────────────────────────────────────╮
│                     SETUP COMPLETION REPORT                 │
├─────────────────────────────────────────────────────────────┤
│ ✓ Go Compiler: Installed successfully                       │
│ ✓ ./frontend: npm dependencies installed                    │
│ ✓ ./services/worker: Python packages installed              │
│                                                             │
│ 🎉 Environment is fully configured and ready to run!        │
╰─────────────────────────────────────────────────────────────╯
```

---

## Helpful Tips

- **Safe preview with `--dry-run`**: Use `ray setup --dry-run` to see what tools Ray intends to install and check disk space requirements before downloading anything.
- **Continuous Integration**: Use `ray setup --yes` in CI or Docker containers to bootstrap all project dependencies automatically.

