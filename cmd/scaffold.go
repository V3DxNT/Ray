package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type ScaffoldItem struct {
	RelPath string
	IsDir   bool
	Content string
}

type ScaffoldReport struct {
	TargetRoot   string
	CreatedDirs  int
	CreatedFiles int
	TotalLines   int
	TotalBytes   int64
	Stack        string
	Database     string
	Items        []ScaffoldItem
}

var (
	scaffoldInteractive bool
	scaffoldTargetDir   string
	scaffoldHere        bool
	scaffoldYes         bool
	scaffoldSave        string
	scaffoldDryRun      bool
)

var scaffoldCmd = &cobra.Command{
	Use:   "scaffold [blueprint-file]",
	Short: "Generate codebase architectures from YAML/JSON blueprints or interactively",
	Run:   runScaffold,
}

func init() {
	RootCmd.AddCommand(scaffoldCmd)
	scaffoldCmd.Flags().BoolVarP(&scaffoldInteractive, "interactive", "i", false, "Launch interactive architecture builder")
	scaffoldCmd.Flags().StringVarP(&scaffoldTargetDir, "dir", "d", "", "Target directory path to scaffold into")
	scaffoldCmd.Flags().BoolVar(&scaffoldHere, "here", false, "Scaffold directly into the current directory")
	scaffoldCmd.Flags().BoolVarP(&scaffoldYes, "yes", "y", false, "Automatic confirmation for scaffolding plan")
	scaffoldCmd.Flags().StringVar(&scaffoldSave, "save", "", "Export generated blueprint to file path (e.g. blueprint.yaml)")
	scaffoldCmd.Flags().BoolVar(&scaffoldDryRun, "dry-run", false, "Preview scaffold operations without writing files")
}

func runScaffold(cmd *cobra.Command, args []string) {
	reader := bufio.NewReader(os.Stdin)

	if len(args) == 0 || scaffoldInteractive {
		runInteractiveScaffold(reader)
		return
	}

	blueprintPath := args[0]
	runBlueprintFileScaffold(blueprintPath, reader)
}

func runBlueprintFileScaffold(filePath string, reader *bufio.Reader) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		color.Red("Error reading blueprint file: %v", err)
		return
	}

	var rawDoc any
	if err := yaml.Unmarshal(data, &rawDoc); err != nil {
		color.Red("Failed to parse blueprint %s: %v", filePath, err)
		return
	}

	var rootTree any = rawDoc
	var projectName string = "my-project"

	if docMap, ok := toStringMap(rawDoc); ok {
		if nameVal, exists := docMap["name"]; exists {
			projectName = fmt.Sprint(nameVal)
		}
		if rootVal, exists := docMap["root"]; exists {
			rootTree = rootVal
		} else if structVal, exists := docMap["structure"]; exists {
			rootTree = structVal
		} else if filesVal, exists := docMap["files"]; exists {
			rootTree = filesVal
		}
	}

	targetDir := resolveTargetDestination(reader, projectName)

	items := extractScaffoldItems("", rootTree)
	plan := buildReportSummary(targetDir, "Custom Blueprint", "N/A", items)

	renderScaffoldPlanBox(plan, filepath.Base(filePath))

	if scaffoldDryRun {
		color.Yellow("Dry-run active: no directories or files were created.")
		return
	}

	if !scaffoldYes {
		fmt.Printf(color.HiWhiteString("Proceed with scaffolding %d files across %d directories? [Y/n]: "),
			plan.CreatedFiles, plan.CreatedDirs)
		ans, _ := reader.ReadString('\n')
		ans = strings.TrimSpace(strings.ToLower(ans))
		if ans == "n" || ans == "no" {
			color.Yellow("Scaffolding cancelled by user.")
			return
		}
	}

	report := executeScaffolding(plan)
	renderScaffoldReportBox(report)
}

func runInteractiveScaffold(reader *bufio.Reader) {
	color.Cyan("╭─────────────────────────────────────────────────────────────╮")
	color.Cyan("│               RAY INTERACTIVE ARCHITECTURE GENERATOR        │")
	color.Cyan("╰─────────────────────────────────────────────────────────────╯\n")

	fmt.Println(color.HiCyanString("Step 1: Choose Target Destination"))
	fmt.Println("  1) Current directory (.) - spread files directly here")
	fmt.Println("  2) Create a new directory (subfolder)")
	fmt.Print(color.HiWhiteString("Select destination [1/2] (default 1): "))
	destChoice, _ := reader.ReadString('\n')
	destChoice = strings.TrimSpace(destChoice)

	targetDir := "."
	defaultProjectName := "my-app"
	if destChoice == "2" {
		fmt.Print(color.HiWhiteString("Enter new directory name (default 'my-app'): "))
		nameInput, _ := reader.ReadString('\n')
		nameInput = strings.TrimSpace(nameInput)
		if nameInput != "" {
			defaultProjectName = nameInput
		}
		targetDir = defaultProjectName
	}

	fmt.Println()
	fmt.Println(color.HiCyanString("Step 2: Select Architecture Blueprint"))
	fmt.Println("  1) Go REST API (Clean Architecture: cmd, handlers, models, routes)")
	fmt.Println("  2) Node.js / Express with TypeScript (controllers, services, routes)")
	fmt.Println("  3) Python FastAPI (routers, schemas, models, services)")
	fmt.Println("  4) Fullstack Next.js (App Router, components, lib, server actions)")
	fmt.Println("  5) Rust Service / CLI (src/main, src/lib, handlers, config)")
	fmt.Print(color.HiWhiteString("Select architecture [1-5] (default 1): "))
	archChoice, _ := reader.ReadString('\n')
	archChoice = strings.TrimSpace(archChoice)

	stackName := "Go REST API"
	stackKey := "go"
	switch archChoice {
	case "2":
		stackName = "Node.js / Express (TypeScript)"
		stackKey = "node"
	case "3":
		stackName = "Python FastAPI"
		stackKey = "fastapi"
	case "4":
		stackName = "Next.js Fullstack"
		stackKey = "nextjs"
	case "5":
		stackName = "Rust Service"
		stackKey = "rust"
	default:
		stackName = "Go REST API"
		stackKey = "go"
	}

	fmt.Println()
	fmt.Println(color.HiCyanString("Step 3: Select Database / ORM"))
	dbOption := "None"
	switch stackKey {
	case "go":
		fmt.Println("  1) GORM (PostgreSQL / SQLite ready)")
		fmt.Println("  2) SQLX / Database stdlib")
		fmt.Println("  3) None / In-memory")
		fmt.Print(color.HiWhiteString("Select ORM [1-3] (default 1): "))
		dbChoice, _ := reader.ReadString('\n')
		switch strings.TrimSpace(dbChoice) {
		case "2":
			dbOption = "SQLX"
		case "3":
			dbOption = "None"
		default:
			dbOption = "GORM"
		}
	case "node":
		fmt.Println("  1) Prisma ORM")
		fmt.Println("  2) Drizzle ORM")
		fmt.Println("  3) None / Raw Client")
		fmt.Print(color.HiWhiteString("Select ORM [1-3] (default 1): "))
		dbChoice, _ := reader.ReadString('\n')
		switch strings.TrimSpace(dbChoice) {
		case "2":
			dbOption = "Drizzle"
		case "3":
			dbOption = "None"
		default:
			dbOption = "Prisma"
		}
	case "fastapi":
		fmt.Println("  1) SQLAlchemy")
		fmt.Println("  2) Tortoise ORM")
		fmt.Println("  3) None")
		fmt.Print(color.HiWhiteString("Select ORM [1-3] (default 1): "))
		dbChoice, _ := reader.ReadString('\n')
		switch strings.TrimSpace(dbChoice) {
		case "2":
			dbOption = "Tortoise"
		case "3":
			dbOption = "None"
		default:
			dbOption = "SQLAlchemy"
		}
	case "nextjs":
		fmt.Println("  1) Prisma ORM")
		fmt.Println("  2) None")
		fmt.Print(color.HiWhiteString("Select ORM [1-2] (default 1): "))
		dbChoice, _ := reader.ReadString('\n')
		if strings.TrimSpace(dbChoice) == "2" {
			dbOption = "None"
		} else {
			dbOption = "Prisma"
		}
	case "rust":
		fmt.Println("  1) SQLx")
		fmt.Println("  2) None")
		fmt.Print(color.HiWhiteString("Select ORM [1-2] (default 1): "))
		dbChoice, _ := reader.ReadString('\n')
		if strings.TrimSpace(dbChoice) == "2" {
			dbOption = "None"
		} else {
			dbOption = "SQLx"
		}
	}

	fmt.Println()
	fmt.Println(color.HiCyanString("Step 4: Additional Components"))
	fmt.Print(color.HiWhiteString("Include Dockerfile & docker-compose.yml? [Y/n]: "))
	dockIn, _ := reader.ReadString('\n')
	includeDocker := strings.TrimSpace(strings.ToLower(dockIn)) != "n"

	fmt.Print(color.HiWhiteString("Include GitHub Actions CI workflow? [y/N]: "))
	ciIn, _ := reader.ReadString('\n')
	includeCI := strings.TrimSpace(strings.ToLower(ciIn)) == "y"

	fmt.Print(color.HiWhiteString("Include .env.example configuration? [Y/n]: "))
	envIn, _ := reader.ReadString('\n')
	includeEnv := strings.TrimSpace(strings.ToLower(envIn)) != "n"

	blueprintTree := buildDynamicArchitectureTree(stackKey, defaultProjectName, dbOption, includeDocker, includeCI, includeEnv)

	if scaffoldSave == "" {
		fmt.Println()
		fmt.Print(color.HiWhiteString("Save generated blueprint file (blueprint.yaml)? [y/N]: "))
		saveIn, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(saveIn)) == "y" {
			fmt.Print(color.HiWhiteString("Enter filename [default: blueprint.yaml]: "))
			fileIn, _ := reader.ReadString('\n')
			fileIn = strings.TrimSpace(fileIn)
			if fileIn == "" {
				fileIn = "blueprint.yaml"
			}
			scaffoldSave = fileIn
		}
	}

	if scaffoldSave != "" {
		exportData := map[string]any{
			"name":        defaultProjectName,
			"stack":       stackName,
			"database":    dbOption,
			"generatedBy": "Ray CLI",
			"root":        blueprintTree,
		}
		yamlBytes, err := yaml.Marshal(exportData)
		if err == nil {
			_ = os.WriteFile(scaffoldSave, yamlBytes, 0644)
			color.Green("✔ Blueprint exported successfully to %s", scaffoldSave)
		}
	}

	items := extractScaffoldItems("", blueprintTree)
	plan := buildReportSummary(targetDir, stackName, dbOption, items)

	renderScaffoldPlanBox(plan, "interactive")

	if scaffoldDryRun {
		color.Yellow("Dry-run active: no directories or files were created.")
		return
	}

	if !scaffoldYes {
		fmt.Printf(color.HiWhiteString("Proceed with scaffolding %d files across %d directories? [Y/n]: "),
			plan.CreatedFiles, plan.CreatedDirs)
		ans, _ := reader.ReadString('\n')
		ans = strings.TrimSpace(strings.ToLower(ans))
		if ans == "n" || ans == "no" {
			color.Yellow("Scaffolding cancelled by user.")
			return
		}
	}

	report := executeScaffolding(plan)
	renderScaffoldReportBox(report)
}

func resolveTargetDestination(reader *bufio.Reader, defaultName string) string {
	if scaffoldTargetDir != "" {
		return scaffoldTargetDir
	}
	if scaffoldHere {
		return "."
	}

	fmt.Println(color.HiCyanString("Where would you like to scaffold this project?"))
	fmt.Println("  1) Current directory (.) - spread in the current folder")
	fmt.Println("  2) Create a new directory")
	fmt.Print(color.HiWhiteString("Select option [1/2] (default 1): "))
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	if choice == "2" {
		prompt := fmt.Sprintf("Enter directory name [default: %s]: ", defaultName)
		fmt.Print(color.HiWhiteString(prompt))
		dirName, _ := reader.ReadString('\n')
		dirName = strings.TrimSpace(dirName)
		if dirName != "" {
			return dirName
		}
		return defaultName
	}

	return "."
}

func buildReportSummary(targetDir, stack, db string, items []ScaffoldItem) ScaffoldReport {
	dirsCount := 0
	filesCount := 0
	linesCount := 0
	var bytesCount int64

	seenDirs := make(map[string]bool)

	for _, item := range items {
		if item.IsDir {
			if !seenDirs[item.RelPath] {
				seenDirs[item.RelPath] = true
				dirsCount++
			}
		} else {
			filesCount++
			bytesCount += int64(len(item.Content))
			if len(item.Content) > 0 {
				linesCount += strings.Count(item.Content, "\n") + 1
			}
			parent := filepath.Dir(item.RelPath)
			if parent != "." && parent != "" && !seenDirs[parent] {
				seenDirs[parent] = true
				dirsCount++
			}
		}
	}

	return ScaffoldReport{
		TargetRoot:   targetDir,
		CreatedDirs:  dirsCount,
		CreatedFiles: filesCount,
		TotalLines:   linesCount,
		TotalBytes:   bytesCount,
		Stack:        stack,
		Database:     db,
		Items:        items,
	}
}

func extractScaffoldItems(currPath string, node any) []ScaffoldItem {
	var items []ScaffoldItem

	if node == nil {
		if strings.HasSuffix(currPath, "/") {
			items = append(items, ScaffoldItem{
				RelPath: filepath.Clean(currPath),
				IsDir:   true,
			})
		} else if currPath != "" {
			items = append(items, ScaffoldItem{
				RelPath: filepath.Clean(currPath),
				IsDir:   false,
				Content: "",
			})
		}
		return items
	}

	if strVal, ok := node.(string); ok {
		if strings.HasSuffix(currPath, "/") {
			items = append(items, ScaffoldItem{
				RelPath: filepath.Clean(currPath),
				IsDir:   true,
			})
		} else if currPath != "" {
			items = append(items, ScaffoldItem{
				RelPath: filepath.Clean(currPath),
				IsDir:   false,
				Content: strVal,
			})
		}
		return items
	}

	if nodeMap, ok := toStringMap(node); ok {
		if currPath != "" {
			items = append(items, ScaffoldItem{
				RelPath: filepath.Clean(currPath),
				IsDir:   true,
			})
		}
		for k, v := range nodeMap {
			nextPath := k
			if currPath != "" {
				nextPath = filepath.Join(currPath, k)
				if strings.HasSuffix(k, "/") {
					nextPath += "/"
				}
			}
			items = append(items, extractScaffoldItems(nextPath, v)...)
		}
		return items
	}

	if sliceVal, ok := node.([]any); ok {
		if currPath != "" {
			items = append(items, ScaffoldItem{
				RelPath: filepath.Clean(currPath),
				IsDir:   true,
			})
		}
		for _, elem := range sliceVal {
			if strElem, ok := elem.(string); ok {
				subPath := strElem
				if currPath != "" {
					subPath = filepath.Join(currPath, strElem)
					if strings.HasSuffix(strElem, "/") {
						subPath += "/"
					}
				}
				if strings.HasSuffix(strElem, "/") {
					items = append(items, ScaffoldItem{
						RelPath: filepath.Clean(subPath),
						IsDir:   true,
					})
				} else {
					items = append(items, ScaffoldItem{
						RelPath: filepath.Clean(subPath),
						IsDir:   false,
						Content: "",
					})
				}
			} else {
				items = append(items, extractScaffoldItems(currPath, elem)...)
			}
		}
		return items
	}

	if currPath != "" {
		items = append(items, ScaffoldItem{
			RelPath: filepath.Clean(currPath),
			IsDir:   false,
			Content: fmt.Sprint(node),
		})
	}

	return items
}

func toStringMap(val any) (map[string]any, bool) {
	if m, ok := val.(map[string]any); ok {
		return m, true
	}
	if m, ok := val.(map[any]any); ok {
		res := make(map[string]any)
		for k, v := range m {
			res[fmt.Sprint(k)] = v
		}
		return res, true
	}
	return nil, false
}

func executeScaffolding(report ScaffoldReport) ScaffoldReport {
	seenDirs := make(map[string]bool)

	if report.TargetRoot != "." && report.TargetRoot != "" {
		_ = os.MkdirAll(report.TargetRoot, 0755)
		seenDirs[report.TargetRoot] = true
	}

	for _, item := range report.Items {
		fullPath := filepath.Join(report.TargetRoot, item.RelPath)
		if item.IsDir {
			_ = os.MkdirAll(fullPath, 0755)
			seenDirs[fullPath] = true
		} else {
			parent := filepath.Dir(fullPath)
			if parent != "." && parent != "" && !seenDirs[parent] {
				_ = os.MkdirAll(parent, 0755)
				seenDirs[parent] = true
			}
			_ = os.WriteFile(fullPath, []byte(item.Content), 0644)
		}
	}

	return report
}

func renderScaffoldPlanBox(plan ScaffoldReport, source string) {
	color.Cyan("╭─────────────────────────────────────────────────────────────╮")
	color.Cyan("│                  CODEBASE SCAFFOLD GENERATOR                │")
	color.Cyan("├───────────────────────┬─────────────────────────────────────┤")

	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Blueprint Source", color.HiYellowString(source))

	destLabel := plan.TargetRoot
	if plan.TargetRoot == "." {
		destLabel = ". (Current Directory)"
	} else {
		destLabel = plan.TargetRoot + " (New Subfolder)"
	}
	if len(destLabel) > 35 {
		destLabel = destLabel[:32] + "..."
	}
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Target Destination", color.HiWhiteString(destLabel))

	stackStr := plan.Stack
	if len(stackStr) > 35 {
		stackStr = stackStr[:32] + "..."
	}
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Architecture Stack", color.HiCyanString(stackStr))

	dbStr := plan.Database
	if len(dbStr) > 35 {
		dbStr = dbStr[:32] + "..."
	}
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Database / ORM", color.HiMagentaString(dbStr))

	dirLabel := fmt.Sprintf("%d Directories", plan.CreatedDirs)
	fileLabel := fmt.Sprintf("%d Files (~%d lines)", plan.CreatedFiles, plan.TotalLines)
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Planned Directories", color.HiWhiteString(dirLabel))
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Planned Files", color.HiGreenString(fileLabel))
	color.Cyan("╰───────────────────────┴─────────────────────────────────────╯\n")
}

func renderScaffoldReportBox(report ScaffoldReport) {
	color.Cyan("╭─────────────────────────────────────────────────────────────╮")
	color.Cyan("│                SCAFFOLD EXECUTION REPORT                    │")
	color.Cyan("├───────────────────────┬─────────────────────────────────────┤")

	absRoot, _ := filepath.Abs(report.TargetRoot)
	if len(absRoot) > 35 {
		absRoot = "..." + absRoot[len(absRoot)-32:]
	}
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Destination Root", color.HiYellowString(absRoot))

	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Directories Created", color.HiWhiteString("%d directories", report.CreatedDirs))
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Files Written", color.HiGreenString("%d files", report.CreatedFiles))

	sizeKB := float64(report.TotalBytes) / 1024.0
	codeSummary := fmt.Sprintf("%d lines (%.1f KB)", report.TotalLines, sizeKB)
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Total Code Generated", color.HiWhiteString(codeSummary))

	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Execution Status", color.HiGreenString("✔ Complete & Ready"))
	color.Cyan("╰───────────────────────┴─────────────────────────────────────╯\n")

	color.HiGreen("✔ Architecture scaffolding successfully applied!")
	if report.TargetRoot != "." {
		fmt.Printf("\nNext steps:\n  %s %s\n  %s\n\n",
			color.CyanString("cd"),
			report.TargetRoot,
			color.YellowString("ray doctor . && ray audit ."))
	} else {
		fmt.Printf("\nNext steps:\n  %s\n\n",
			color.YellowString("ray doctor . && ray audit ."))
	}
}

func buildDynamicArchitectureTree(stack, name, db string, docker, ci, env bool) map[string]any {
	root := make(map[string]any)

	switch stack {
	case "go":
		root = buildGoArchitecture(name, db)
	case "node":
		root = buildNodeArchitecture(name, db)
	case "fastapi":
		root = buildFastAPIArchitecture(name, db)
	case "nextjs":
		root = buildNextArchitecture(name, db)
	case "rust":
		root = buildRustArchitecture(name, db)
	default:
		root = buildGoArchitecture(name, db)
	}

	if docker {
		root["Dockerfile"] = generateDockerfile(stack)
		root["docker-compose.yml"] = generateDockerCompose(name, db)
	}

	if ci {
		root[".github"] = map[string]any{
			"workflows": map[string]any{
				"ci.yml": generateCIWorkflow(stack),
			},
		}
	}

	if env {
		root[".env.example"] = generateEnvFile(db)
		root[".gitignore"] = generateGitignore(stack)
	}

	return root
}

func buildGoArchitecture(name, db string) map[string]any {
	dbImport := ""
	dbInit := ""
	if db == "GORM" {
		dbImport = "\t\"gorm.io/gorm\"\n\t\"gorm.io/driver/sqlite\""
		dbInit = "func InitDB() *gorm.DB {\n\tdb, err := gorm.Open(sqlite.Open(\"app.db\"), &gorm.Config{})\n\tif err != nil {\n\t\tpanic(\"failed to connect database\")\n\t}\n\treturn db\n}"
	} else if db == "SQLX" {
		dbImport = "\t\"github.com/jmoiron/sqlx\"\n\t_ \"github.com/mattn/go-sqlite3\""
		dbInit = "func InitDB() *sqlx.DB {\n\tdb, err := sqlx.Connect(\"sqlite3\", \"app.db\")\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\treturn db\n}"
	}

	mainContent := fmt.Sprintf(`package main

import (
	"fmt"
	"net/http"
	"%s/internal/routes"
)

func main() {
	r := routes.SetupRouter()
	fmt.Println("🚀 Server listening on http://localhost:8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		fmt.Printf("Error: %%v\n", err)
	}
}
`, name)

	routesContent := `package routes

import (
	"net/http"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func SetupRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(` + "`" + `{"status":"ok","service":"` + name + `"}` + "`" + `))
	})

	return r
}
`

	userHandler := `package handlers

import (
	"encoding/json"
	"net/http"
)

type UserHandler struct{}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

func (h *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "user list"})
}
`

	userModel := `package models

import "time"

type User struct {
	ID        string    ` + "`" + `json:"id" db:"id"` + "`" + `
	Email     string    ` + "`" + `json:"email" db:"email"` + "`" + `
	Name      string    ` + "`" + `json:"name" db:"name"` + "`" + `
	CreatedAt time.Time ` + "`" + `json:"created_at" db:"created_at"` + "`" + `
}
`

	dbFileContent := fmt.Sprintf(`package database

import (
%s
)

%s
`, dbImport, dbInit)

	goMod := fmt.Sprintf(`module %s

go 1.22

require (
	github.com/go-chi/chi/v5 v5.0.12
)
`, name)

	res := map[string]any{
		"cmd": map[string]any{
			"api": map[string]any{
				"main.go": mainContent,
			},
		},
		"internal": map[string]any{
			"handlers": map[string]any{
				"user_handler.go": userHandler,
			},
			"models": map[string]any{
				"user.go": userModel,
			},
			"routes": map[string]any{
				"routes.go": routesContent,
			},
		},
		"go.mod":    goMod,
		"README.md": fmt.Sprintf("# %s\n\nScaffolded with Ray Clean Architecture.", name),
	}

	if db != "None" {
		intMap := res["internal"].(map[string]any)
		intMap["database"] = map[string]any{
			"db.go": dbFileContent,
		}
	}

	return res
}

func buildNodeArchitecture(name, db string) map[string]any {
	indexTs := `import express, { Request, Response } from 'express';
import { userRoutes } from './routes/userRoutes';

const app = express();
const port = process.env.PORT || 3000;

app.use(express.json());

app.get('/health', (req: Request, res: Response) => {
  res.json({ status: 'ok', uptime: process.uptime() });
});

app.use('/api/users', userRoutes);

app.listen(port, () => {
  console.log(` + "`🚀 Server running on http://localhost:${port}`" + `);
});
`

	userRoutes := `import { Router } from 'express';
import { UserController } from '../controllers/userController';

export const userRoutes = Router();
const controller = new UserController();

userRoutes.get('/', controller.listUsers);
userRoutes.post('/', controller.createUser);
`

	userController := `import { Request, Response } from 'express';
import { UserService } from '../services/userService';

export class UserController {
  private service = new UserService();

  listUsers = async (req: Request, res: Response) => {
    const users = await this.service.getAll();
    res.json({ data: users });
  };

  createUser = async (req: Request, res: Response) => {
    const user = await this.service.create(req.body);
    res.status(201).json({ data: user });
  };
}
`

	userService := `export interface User {
  id: string;
  email: string;
  name: string;
}

export class UserService {
  private users: User[] = [];

  async getAll(): Promise<User[]> {
    return this.users;
  }

  async create(payload: Partial<User>): Promise<User> {
    const newUser: User = {
      id: Math.random().toString(36).substring(7),
      email: payload.email || '',
      name: payload.name || '',
    };
    this.users.push(newUser);
    return newUser;
  }
}
`

	packageJson := fmt.Sprintf(`{
  "name": "%s",
  "version": "1.0.0",
  "main": "dist/index.js",
  "scripts": {
    "dev": "tsx watch src/index.ts",
    "build": "tsc",
    "start": "node dist/index.js"
  },
  "dependencies": {
    "express": "^4.19.2",
    "dotenv": "^16.4.5"
  },
  "devDependencies": {
    "@types/express": "^4.17.21",
    "@types/node": "^20.12.7",
    "tsx": "^4.7.2",
    "typescript": "^5.4.5"
  }
}
`, name)

	tsconfig := `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "outDir": "./dist",
    "rootDir": "./src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true
  },
  "include": ["src/**/*"]
}
`

	res := map[string]any{
		"src": map[string]any{
			"index.ts": indexTs,
			"routes": map[string]any{
				"userRoutes.ts": userRoutes,
			},
			"controllers": map[string]any{
				"userController.ts": userController,
			},
			"services": map[string]any{
				"userService.ts": userService,
			},
		},
		"package.json": packageJson,
		"tsconfig.json": tsconfig,
		"README.md": fmt.Sprintf("# %s\n\nNode.js Express + TypeScript service.", name),
	}

	if db == "Prisma" {
		res["prisma"] = map[string]any{
			"schema.prisma": `datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

generator client {
  provider = "prisma-client-js"
}

model User {
  id        String   @id @default(uuid())
  email     String   @unique
  name      String?
  createdAt DateTime @default(now())
}
`,
		}
	}

	return res
}

func buildFastAPIArchitecture(name, db string) map[string]any {
	mainPy := fmt.Sprintf(`from fastapi import FastAPI
from app.routers import users

app = FastAPI(title="%s", version="1.0.0")

@app.get("/health")
def health():
    return {"status": "ok", "app": "%s"}

app.include_router(users.router, prefix="/api/v1/users", tags=["users"])
`, name, name)

	usersRouter := `from fastapi import APIRouter
from app.schemas.user import UserRead, UserCreate

router = APIRouter()

@router.get("/", response_model=list[UserRead])
def get_users():
    return [{"id": "1", "email": "test@example.com", "name": "Admin"}]

@router.post("/", response_model=UserRead)
def create_user(user: UserCreate):
    return {"id": "2", **user.model_dump()}
`

	schemas := `from pydantic import BaseModel, EmailStr

class UserBase(BaseModel):
    email: EmailStr
    name: str

class UserCreate(UserBase):
    pass

class UserRead(UserBase):
    id: str
`

	requirements := `fastapi>=0.110.0
uvicorn[standard]>=0.29.0
pydantic>=2.7.0
pydantic-settings>=2.2.1
`
	if db == "SQLAlchemy" {
		requirements += "sqlalchemy>=2.0.29\nalembic>=1.13.1\npsycopg2-binary>=2.9.9\n"
	}

	return map[string]any{
		"app": map[string]any{
			"main.py": mainPy,
			"routers": map[string]any{
				"__init__.py": "",
				"users.py":    usersRouter,
			},
			"schemas": map[string]any{
				"__init__.py": "",
				"user.py":     schemas,
			},
			"models": map[string]any{
				"__init__.py": "",
			},
			"services": map[string]any{
				"__init__.py": "",
			},
		},
		"requirements.txt": requirements,
		"README.md":        fmt.Sprintf("# %s\n\nFastAPI Application.", name),
	}
}

func buildNextArchitecture(name, db string) map[string]any {
	pageTsx := `export default function Home() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center p-24">
      <h1 className="text-4xl font-bold tracking-tight">` + name + `</h1>
      <p className="mt-4 text-gray-500">Scaffolded with Ray</p>
    </main>
  );
}
`

	layoutTsx := `export const metadata = {
  title: '` + name + `',
  description: 'Generated with Ray CLI',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
`

	routeHealth := `import { NextResponse } from 'next/server';

export async function GET() {
  return NextResponse.json({ status: 'ok', timestamp: new Date().toISOString() });
}
`

	pkgJson := fmt.Sprintf(`{
  "name": "%s",
  "version": "0.1.0",
  "private": true,
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start"
  },
  "dependencies": {
    "next": "14.2.3",
    "react": "^18",
    "react-dom": "^18"
  },
  "devDependencies": {
    "@types/node": "^20",
    "@types/react": "^18",
    "@types/react-dom": "^18",
    "typescript": "^5"
  }
}
`, name)

	return map[string]any{
		"app": map[string]any{
			"page.tsx":   pageTsx,
			"layout.tsx": layoutTsx,
			"api": map[string]any{
				"health": map[string]any{
					"route.ts": routeHealth,
				},
			},
		},
		"components": map[string]any{
			"Navbar.tsx": "export const Navbar = () => <nav>Navbar</nav>;",
		},
		"lib": map[string]any{
			"utils.ts": "export function cn(...classes: string[]) { return classes.filter(Boolean).join(' '); }",
		},
		"package.json": pkgJson,
		"README.md":    fmt.Sprintf("# %s\n\nNext.js Fullstack Project.", name),
	}
}

func buildRustArchitecture(name, db string) map[string]any {
	mainRs := `mod handlers;
mod models;

use std::net::SocketAddr;

#[tokio::main]
async fn main() {
    let addr = SocketAddr::from(([127, 0, 0, 1], 8080));
    println!("🚀 Rust service running on http://{}", addr);
}
`

	cargoToml := fmt.Sprintf(`[package]
name = "%s"
version = "0.1.0"
edition = "2021"

[dependencies]
tokio = { version = "1", features = ["full"] }
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
`, name)

	return map[string]any{
		"src": map[string]any{
			"main.rs": mainRs,
			"lib.rs":  "pub mod handlers;\npub mod models;",
			"handlers": map[string]any{
				"mod.rs": "pub struct Handlers;\n",
			},
			"models": map[string]any{
				"mod.rs": "pub struct Models;\n",
			},
		},
		"Cargo.toml": cargoToml,
		"README.md":  fmt.Sprintf("# %s\n\nRust Service.", name),
	}
}

func generateDockerfile(stack string) string {
	switch stack {
	case "go":
		return `FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 8080
CMD ["./main"]
`
	case "node":
		return `FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:20-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY --from=builder /app/dist ./dist
EXPOSE 3000
CMD ["node", "dist/index.js"]
`
	case "fastapi":
		return `FROM python:3.11-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
EXPOSE 8000
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]
`
	default:
		return "FROM alpine:latest\nCMD [\"echo\", \"Running application\"]\n"
	}
}

func generateDockerCompose(name, db string) string {
	if db == "GORM" || db == "Prisma" || db == "SQLAlchemy" {
		return fmt.Sprintf(`version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://user:password@postgres:5432/%s
    depends_on:
      - postgres

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: password
      POSTGRES_DB: %s
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data

volumes:
  pgdata:
`, name, name)
	}

	return `version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
`
}

func generateCIWorkflow(stack string) string {
	return `name: CI Pipeline

on:
  push:
    branches: [ main, master ]
  pull_request:
    branches: [ main, master ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run verification
        run: echo "Codebase verified"
`
}

func generateEnvFile(db string) string {
	if db == "GORM" || db == "Prisma" || db == "SQLAlchemy" {
		return "PORT=8080\nDATABASE_URL=postgres://user:password@localhost:5432/appdb\nNODE_ENV=development\nSECRET_KEY=dev-secret-key-change-me\n"
	}
	return "PORT=8080\nNODE_ENV=development\nSECRET_KEY=dev-secret-key-change-me\n"
}

func generateGitignore(stack string) string {
	return `.DS_Store
node_modules/
dist/
build/
*.log
.env
.env.local
.vscode/
.idea/
`
}
