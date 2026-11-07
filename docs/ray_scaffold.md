# `ray scaffold`

Create clean folder structures and starter code from blueprint files or interactively.

---

## What It Does

`ray scaffold` helps you bootstrap a new project structure with starter code in seconds. Instead of creating folders like `handlers/`, `models/`, `routes/`, and `tests/` manually every time, Ray builds them for you.

You can create projects in two ways:
1. **Interactive wizard (`--interactive` or running without arguments)**: Ray asks simple questions (Which language? Which framework? Which database? Include Dockerfile?) and generates the code.
2. **From a blueprint file (`ray scaffold blueprint.yaml` or `api.json`)**: Ray reads a YAML or JSON blueprint definition and creates all folders and starter files described in the file.

---

## When to Use It

- When starting a new microservice, backend API, or web project from scratch.
- When creating standard starter templates for your team or organization.
- When you want starter files that compile and run immediately out of the box.

---

## How to Run

```bash
# Launch the interactive wizard
ray scaffold

# Explicitly launch the interactive mode
ray scaffold --interactive

# Create architecture from a YAML blueprint
ray scaffold blueprint.yaml

# Create architecture from a JSON blueprint into a new folder
ray scaffold api.json --dir my-service

# Create directly in the current directory (like 'git clone .')
ray scaffold blueprint.yaml --here

# Answer questions interactively and save the configuration to a file for later
ray scaffold --interactive --save template.yaml

# Preview files that would be created without writing anything to disk
ray scaffold blueprint.yaml --dry-run
```

---

## Flags and Options

| Flag | Shorthand | Default | Description |
| :--- | :--- | :--- | :--- |
| `--interactive` | `-i` | `false` | Launches the interactive step-by-step project builder wizard. |
| `--dir` | `-d` | `""` | The name or path of the folder where files should be created. |
| `--here` | | `false` | Creates files directly inside the current folder instead of making a subfolder. |
| `--yes` | `-y` | `false` | Confirms and creates the files immediately without pausing to ask. |
| `--save` | | `""` | Saves the generated configuration into a YAML blueprint file for future reuse. |
| `--dry-run` | | `false` | Previews the files and folders that will be created without writing to disk. |
| `--help` | `-h` | | Shows help and usage information for this command. |

---

## Blueprint File Syntax (YAML & JSON)

Ray supports nested maps for folders and strings for file contents.

### Example YAML Blueprint (`api.yaml`)

```yaml
version: "1.0"
name: "order-service"
structure:
  cmd:
    api:
      main.go: |
        package main

        import "fmt"

        func main() {
            fmt.Println("Order Service API is running!")
        }
  internal:
    handlers:
      orders.go: |
        package handlers
    models:
      order.go: |
        package models
  Dockerfile: |
    FROM golang:1.22-alpine
    WORKDIR /app
    COPY . .
    RUN go build -o /server cmd/api/main.go
    CMD ["/server"]
  README.md: "# Order Service"
```

You can pass this blueprint directly:
```bash
ray scaffold api.yaml
```

---

## Sample Output

### Interactive Wizard

Running `ray scaffold`:

```text
🚀 Welcome to Ray Scaffold Architecture Builder!

? Select programming language & framework:
  ▸ Go (Gin Router)
    Go (Fiber)
    Node.js (Express + TypeScript)
    Python (FastAPI)

? Select database layer:
  ▸ PostgreSQL (with GORM)
    SQLite
    None

? Include production Dockerfile? [Y/n]: y
? Scaffold directly into current directory (.) or create a new folder?
  ▸ Create a new folder
    Spread into current directory (.)

? Enter folder name: order-api
```

### Creation Summary

```text
╭─────────────────────────────────────────────────────────────╮
│                     SCAFFOLD SUMMARY                        │
├───────────────────────┬─────────────────────────────────────┤
│ Target Directory      │ ./order-api                         │
│ Folders Created       │ 4                                   │
│ Files Written         │ 6                                   │
│ Total Lines of Code   │ 128                                 │
│ Total Size            │ 3.42 KB                             │
╰───────────────────────┴─────────────────────────────────────╯

🎉 Architecture successfully generated!
Next steps:
  cd order-api
  ray doctor
```

---

## Helpful Tips

- **Blueprint flexibility**: Blueprints accept both YAML and JSON, making them easy to generate from scripts or store in version control.
- **Save your templates**: Use `--save my-template.yaml` during an interactive session to keep your favorite setup so you can recreate it anytime with `ray scaffold my-template.yaml`.

