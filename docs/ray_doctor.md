# `ray doctor`

Check if required runtimes, compilers, and tools are installed on your computer.

---

## What It Does

`ray doctor` diagnoses your local development environment. It looks at your project files to see what technologies are used (such as Node.js, Go, Python, Java, Docker, or databases), and then verifies whether your system has each required tool installed, checks the installed version, and highlights anything missing.

---

## When to Use It

- When setting up a new developer machine or onboarding a teammate to a repository.
- When builds fail unexpectedly and you suspect a missing tool or incompatible version.
- Before running or building a project to ensure all prerequisites are satisfied.

---

## How to Run

```bash
# Check toolchain prerequisites for current project
ray doctor

# Check a specific project directory
ray doctor ./my-app

# Output the diagnostic report in JSON format
ray doctor --json
```

---

## Flags and Options

| Flag | Shorthand | Default | Description |
| :--- | :--- | :--- | :--- |
| `--json` | | `false` | Outputs the diagnosis results and tool versions in JSON format. |
| `--help` | `-h` | | Shows help and usage information for this command. |

---

## Sample Output

### Health Report

Running `ray doctor` on a polyglot project:

```text
🔍 Inspecting: /Users/user/project

╭─────────────────────────────────────────────────────────────╮
│                 DETECTED PROJECT ENVIRONMENT                │
├───────────────────────┬─────────────────────────────────────┤
│ Monorepo / Components │ 2 Subprojects Detected              │
│ • Next.js (Node.js)   │ ./frontend (package.json)           │
│ • Go Modules          │ ./backend (go.mod)                  │
│ • Containerization    │ ./docker-compose.yml                │
╰───────────────────────┴─────────────────────────────────────╯
╭─────────────────────────────────────────────────────────────╮
│               SYSTEM TOOLCHAIN & DEPENDENCIES               │
├──────────┬──────────────────┬──────────┬────────────────────┤
│ Status   │ Tool / Runtime   │ Required │ Version / State    │
├──────────┼──────────────────┼──────────┼────────────────────┤
│    ✓     │ Node.js          │ Yes      │ v20.11.0           │
│    ✓     │ npm              │ Yes      │ 10.2.4             │
│    ✓     │ Go Compiler      │ Yes      │ go1.22.1           │
│    ✓     │ Docker           │ Optional │ 24.0.7             │
│    ✗     │ Docker Compose   │ Optional │ Not Installed      │
╰──────────┴──────────────────┴──────────┴────────────────────╯
╭─────────────────────────────────────────────────────────────╮
│                     HEALTH SUMMARY REPORT                   │
├─────────────────────────────────────────────────────────────┤
│ 🎯 Toolchain Health: 3/3 Required Satisfied (100%)           │
│ 📦 Package Dependencies: Synced and installed               │
│                                                             │
│ 💡 Verdict: All required tools are ready to build!          │
╰─────────────────────────────────────────────────────────────╯
```

---

## Helpful Tips

- **Next step**: If `ray doctor` reports any missing required tools, run `ray setup` to automatically install them with your system package manager in one go!

