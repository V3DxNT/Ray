# `ray cast`

Show a clean, visual tree of your folders and files.

---

## What It Does

`ray cast` prints a visual tree of your files and folders in your terminal. By default, it automatically hides noisy folders you rarely need to see—such as `node_modules`, `.git`, `build`, and cache folders—so you get an instant, readable overview of your project without clutter.

---

## When to Use It

- When you just cloned a new repository and want to see how it is organized.
- When sharing your project structure in a README, pull request, or issue.
- When saving your folder structure to a text file for team documentation.

---

## How to Run

```bash
# View the tree for your current folder
ray cast

# View the tree for a specific folder
ray cast src/

# Save the tree output directly to a file
ray cast --output tree.txt

# Show absolutely everything, including hidden and skipped folders
ray cast --all

# Skip custom folders you don't want to see
ray cast --skip dist,coverage
```

---

## Flags and Options

| Flag | Shorthand | Default | Description |
| :--- | :--- | :--- | :--- |
| `--all` | `-a` | `false` | Shows every file and folder, ignoring the default skip list. |
| `--output` | `-o` | `""` | Saves the tree output to a text file instead of printing to the terminal. |
| `--skip` | `-s` | `node_modules, .git, build, data` | A comma-separated list of folders or files you want to hide from the tree. |
| `--help` | `-h` | | Shows help and usage information for this command. |

---

## Sample Output

### Standard Tree View

Running `ray cast` inside a web project:

```text
Printing Files
├── .env.example
├── README.md
├── package.json
├── tsconfig.json
├── src
│   ├── index.ts
│   ├── config
│   │   └── database.ts
│   ├── controllers
│   │   ├── auth.ts
│   │   └── user.ts
│   └── routes
│       └── api.ts
└── tests
    └── user.test.ts
```

### Saving to a File

When you provide the `--output` flag:

```bash
ray cast . --output project-structure.txt
```

```text
Directory tree successfully saved to: project-structure.txt
```

The file `project-structure.txt` is created with the full tree layout, ready to be attached to your project notes or documentation.

---

## Helpful Tips

- **Default clean view**: You don't need to manually configure ignores for everyday use. Ray automatically skips heavy dependency and build directories by default.
- **Exporting for documentation**: Use `--output tree.txt` whenever you want a clean snapshot of your project to include in Markdown files or architecture reviews.

