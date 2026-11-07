<div align="center">

# ⚡ Ray

[![Typing SVG](https://readme-typing-svg.demolab.com?font=Fira+Code&size=20&pause=1000&color=00ADD8&center=true&vCenter=true&width=600&lines=See+your+codebase.;Understand+your+codebase.;One+binary.+Zero+config.)](https://git.io/typing-svg)

A fast, cross-platform CLI toolkit for deep codebase exploration, environment health, and project management — built with Go.

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![Release](https://img.shields.io/github/v/release/V3DxNT/Ray?style=flat-square&color=00ADD8)](https://github.com/V3DxNT/Ray/releases)
[![YouTube](https://img.shields.io/badge/YouTube-@VedByte-FF0000?style=flat-square&logo=youtube&logoColor=white)](https://www.youtube.com/@VedByte)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-ff69b4.svg?style=flat-square)](#-contributing)

</div>

---

## 📺 Video Walkthrough

> Check out detailed tutorials, demos, and feature walkthroughs on YouTube at [**@VedByte**](https://www.youtube.com/@VedByte). *(Full video walkthrough coming soon!)*

---

## ✨ What is Ray?

**Ray** turns a sprawling, polyglot codebase into something you can understand and manage directly from your terminal. Instead of juggling separate tools like `find`, `du`, `grep`, `tree`, secret scanners, and custom scripts, Ray brings the entire developer introspection workflow into a single, lightning-fast binary.

Whether you are exploring a new monorepo, verifying toolchains, finding duplicate assets, cleaning build caches, or catching leaked API keys before pushing code, Ray handles it with zero configuration.

---

## 🚀 Installation

### Option 1: Homebrew (macOS & Linux) — Recommended

Install via our official Homebrew tap:

```bash
brew tap V3DxNT/tap
brew install ray
```

*Or in a single command:*

```bash
brew install V3DxNT/tap/ray
```

### Option 2: Quick Script Install (macOS / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/V3DxNT/Ray/main/install.sh | bash
```

### Option 3: Pre-Built Binaries (GitHub Releases)

Download pre-compiled binaries and standalone packages (`.deb`, `.rpm`, `.zip`, `.tar.gz`) for macOS, Linux, and Windows directly from [**GitHub Releases**](https://github.com/V3DxNT/Ray/releases):

- 📦 **[Latest Release Assets](https://github.com/V3DxNT/Ray/releases/latest)**
- Supports **macOS** (Apple Silicon `arm64` & Intel `amd64`), **Linux** (`amd64`, `arm64`), and **Windows** (`amd64`, `arm64`).

### Option 4: Go Install / Build from Source

```bash
# Using go install (requires Go 1.21+)
go install github.com/V3DxNT/Ray@latest

# Or clone and build manually
git clone https://github.com/V3DxNT/Ray.git
cd Ray
go build -o ray main.go
```

Verify your installation:

```bash
ray --help
```

---

## 🛠️ Core Commands

Ray includes 11 dedicated commands. Click any command to view its full documentation, flag options, and sample outputs:

| Command | Description | Documentation |
| :--- | :--- | :--- |
| **`ray cast`** | Show a visual, color-coded directory tree while automatically skipping noisy build folders. | [Read Guide →](docs/ray_cast.md) |
| **`ray stats`** | Display an instant dashboard of languages, file counts, code lines, sizes, and frameworks. | [Read Guide →](docs/ray_stats.md) |
| **`ray search`** | Quickly locate exact or similar files by name across your entire project. | [Read Guide →](docs/ray_search.md) |
| **`ray clean`** | Find and safely remove build caches (`node_modules`, `target`), logs, heavy files, and Docker cache. | [Read Guide →](docs/ray_clean.md) |
| **`ray dedup`** | Identify duplicate files across your repository using cryptographic SHA-256 checksums. | [Read Guide →](docs/ray_dedup.md) |
| **`ray audit`** | Scan code for leaked secrets (API keys, passwords), standard repository files, and syntax metrics. | [Read Guide →](docs/ray_audit.md) |
| **`ray doctor`** | Diagnose your system environment and verify that required runtimes and tools are installed. | [Read Guide →](docs/ray_doctor.md) |
| **`ray setup`** | Install missing toolchains via your system package manager and bootstrap project dependencies. | [Read Guide →](docs/ray_setup.md) |
| **`ray env`** | Detect environment variables in source code and verify or auto-fix missing keys in your `.env` file. | [Read Guide →](docs/ray_env.md) |
| **`ray ignore`** | Automatically generate tailored `.gitignore` and `.dockerignore` files for multi-language projects. | [Read Guide →](docs/ray_ignore.md) |
| **`ray scaffold`** | Generate clean folder structures and starter boilerplate from blueprints or via interactive wizard. | [Read Guide →](docs/ray_scaffold.md) |

---

## ⚡ Why Ray?

Modern repositories often span multiple languages, separate microservices, and disparate package managers:

```text
my-project/
├── frontend/             # Next.js / TypeScript (npm / yarn)
├── backend/              # Go REST API (go modules)
├── services/worker/      # Python async background processor (pip)
├── infra/                # Docker & Kubernetes manifests
└── ...
```

### The Problem: Too Many Tools, Too Much Noise

In a multi-stack project:
- **Visualizing folders** with standard `tree` floods your screen with tens of thousands of lines from `node_modules`, `.next`, or `.git`.
- **Finding buried configuration files** requires memorizing complicated `find` or `grep` flags.
- **Debugging missing environment variables** requires manually cross-referencing code against `.env`, usually only noticed after a runtime crash in production or CI.
- **Onboarding team members** is slow because verifying required tools (`node`, `go`, `python`, `docker`) requires tedious manual setup.
- **Reclaiming disk space** turns into dangerous guessing games of `rm -rf` across nested subfolders.

### The Solution: One Command-Line Tool, Complete Clarity

Ray unifies codebase introspection, environment health checks, secret safety, and cleanup into a single, cohesive developer workflow:

| Challenge | Without Ray | With Ray |
| :--- | :--- | :--- |
| **Directory Tree** | `tree` explodes with thousands of junk files | `ray cast` automatically hides noisy caches and highlights code |
| **Codebase Metrics** | Juggling multiple tools (`cloc`, `du`, `find`) | `ray stats` renders an instant polyglot metrics dashboard |
| **Toolchain Health** | Manual README checks & missing runtimes | `ray doctor` checks health & `ray setup` auto-installs via `brew`/`apt` |
| **Environment Sync** | Runtime `undefined` errors & broken builds | `ray env` detects code lookups & `--fix` syncs keys to `.env` |
| **Secret Scanning** | Heavy, slow CI/CD scanning suites | `ray audit` scans & masks credentials locally in milliseconds |
| **Disk Recovery** | Guessing which folders take gigabytes of space | `ray clean` & `ray dedup` safely find and clean gigabytes of cache |

- ⚡ **Zero Configuration**: Works instantly in any directory without YAML config files or background processes.
- 🎨 **Consistent UI**: Aesthetic 63-width bordered cards with interactive spinners and clear status indicators.
- 🚀 **Polyglot Intelligence**: Native recognition for Go, Node/TypeScript, Python, Rust, Java, Docker, and more.

---

## 🤝 Contributing

Contributions, issues, and feature suggestions are always welcome!

```bash
# Clone repository
git clone https://github.com/V3DxNT/Ray.git
cd Ray

# Run tests and verify build
go test ./...
go vet ./...
go build -o ray main.go
```

If you would like to contribute:
1. Fork the repository.
2. Create your feature branch (`git checkout -b feat/my-new-feature`).
3. Commit your changes with conventional commit messages.
4. Push to the branch (`git push origin feat/my-new-feature`).
5. Open a Pull Request.

---

<div align="center">

Built with ❤️ by [**vedx.dev**](https://vedx.dev) · Follow on YouTube [**@VedByte**](https://www.youtube.com/@VedByte)

</div>
