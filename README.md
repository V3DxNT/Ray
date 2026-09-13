<div align="center">

# ⚡ Ray

[![Typing SVG](https://readme-typing-svg.demolab.com?font=Fira+Code&size=20&pause=1000&color=00ADD8&center=true&vCenter=true&width=600&lines=See+your+codebase.;Understand+your+codebase.;One+binary.+Zero+config.)](https://git.io/typing-svg)

A fast, cross-platform CLI for deep codebase introspection — built with Go.

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-ff69b4.svg?style=flat-square)](#-contributing)
[![Stars](https://img.shields.io/github/stars/V3DxNT/ray?style=flat-square)](https://github.com/V3DxNT/ray/stargazers)

</div>

---

## ✨ What is Ray?

Ray turns a sprawling repository into something you can understand from your terminal. Instead of jumping between `find`, `du`, `grep`, `tree`, and a handful of one-off scripts, Ray brings the whole codebase-introspection workflow into a single binary.

It identifies technology context, locates deeply nested files, measures repository health, flags potentially exposed secrets, and finds duplicate files — all with zero configuration.

## 🚀 Installation

```bash
curl -fsSL https://raw.githubusercontent.com/V3DxNT/ray/main/install.sh | bash
```

*(Requires [Go](https://go.dev/dl/) to be installed.)*

Verify it worked:

```bash
ray --help
```

## 🛠️ Commands

### `ray cast` — visualize your repo

```bash
ray cast
```

```
📁 backend/
├── 🟦 Go
│   ├── main.go
│   └── handlers.go
└── 🟨 Node
    ├── package.json
    └── vite.config.ts
```

Flags: `--depth N` limits recursion, `--size` shows file sizes, `--time` shows last-modified dates.

### `ray search` — find anything, anywhere

```bash
ray search .env
```

```
/home/user/project/.env
/home/user/project/apps/web/.env
```

Searches recursively, hidden directories included — built for monorepos where config gets buried.

### `ray stats` — understand repo health at a glance

```bash
ray stats
```

```
╭──────────────────────────────────────╮
│ Files              1,482              │
│ Lines of Code      284,912            │
│ Directories        196                │
╰──────────────────────────────────────╯

Go          ████████████████  42%
TypeScript  ████████████      31%
```

### `ray audit` — catch what shouldn't ship

```bash
ray audit
```

```
⚠ Potential secret detected
.env → AWS_SECRET_ACCESS_KEY

Project Health
✓ README.md   ✓ LICENSE   ✗ CONTRIBUTING.md
```

> A developer convenience, not a replacement for dedicated security tooling.

### `ray dedup` — find duplicate files

```bash
ray dedup
```

```
SHA256: 7c4a8d09...
  ./packages/ui/logo.svg
  ./apps/web/assets/logo.svg
```

Uses concurrent SHA-256 hashing to catch duplicates regardless of filename or location.

### `ray upgrade` — stay current

```bash
ray upgrade
```

Builds the new binary in a temp directory and validates it before replacing the current one — a failed upgrade never leaves you without a working `ray`.

## ⚡ Why Ray?

```
my-project/
├── apps/{web, mobile, admin}
├── packages/{ui, database, auth}
├── services/{api, worker, payments}
└── ...
```

In a repo shaped like this, finding one config file is a scavenger hunt, and checking for leaked secrets means reaching for yet another tool. Ray puts all of it behind one interface:

```bash
ray cast && ray search .env && ray stats && ray audit && ray dedup
```

One binary. No daemon. No setup.

## 🧠 Under the Hood

Ray is built around fast filesystem traversal and concurrent processing where it actually helps (hashing, scanning). The goal is to stay fast, cross-platform, modular, and easy to read — no magic, no hidden state.

## 🤝 Contributing

```bash
git clone https://github.com/V3DxNT/ray.git
cd ray
go build ./...
go test ./...
```

Good first contributions: new tech-stack detectors, better audit patterns, Windows support, test coverage, and terminal output polish. Look for issues labeled `good first issue` or `help wanted`.

## 🗺️ Roadmap

- [x] Cast, search, stats, audit, dedup, self-upgrade
- [ ] Interactive TUI
- [ ] Configurable ignore patterns
- [ ] Git-aware analysis
- [ ] JSON / machine-readable output
- [ ] Plugin system

Have an idea? [Open an issue](https://github.com/V3DxNT/ray/issues/new).

<div align="center">

⭐ **[Star the repo](https://github.com/V3DxNT/ray)** if Ray saves you time · built by [vedx.dev](https://vedx.dev)

</div>