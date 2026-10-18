package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type DetectedComponent struct {
	RelativePath string `json:"path"`
	Stack        string `json:"stack"`
	Anchor       string `json:"anchor"`
	DepsOk       bool   `json:"dependencies_installed"`
	DepsNotice   string `json:"dependency_notice,omitempty"`
}

type ToolStatus struct {
	Name        string `json:"name"`
	Installed   bool   `json:"installed"`
	Version     string `json:"version,omitempty"`
	Required    bool   `json:"required"`
	InstallHint string `json:"install_hint"`
}

type ProjectDiagnostic struct {
	Components  []DetectedComponent `json:"components"`
	Tools       []ToolStatus        `json:"tools"`
	AllReady    bool                `json:"all_ready"`
	MissingReqs []string            `json:"missing_required_tools"`
}

var doctorJson bool

var doctorCmd = &cobra.Command{
	Use:   "doctor [path]",
	Short: "Validate development environment, toolchains, and project dependencies",
	Run:   runDoctor,
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorJson, "json", false, "Output doctor health report in JSON format")
	RootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) {
	var targetDir string
	if len(args) == 0 {
		pwd, err := os.Getwd()
		if err != nil {
			color.Red("Error getting current working directory: %v", err)
			return
		}
		targetDir = pwd
	} else {
		targetDir = args[0]
	}

	absPath, err := filepath.Abs(targetDir)
	if err != nil {
		color.Red("Invalid path: %v", err)
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		color.Red("Path does not exist: %v", err)
		return
	}

	if !info.IsDir() {
		color.Red("Target path must be a directory")
		return
	}

	diag := inspectFullArchitecture(absPath)

	if doctorJson {
		jsonBytes, err := json.MarshalIndent(diag, "", "  ")
		if err != nil {
			color.Red("Error marshaling json: %v", err)
			return
		}
		fmt.Println(string(jsonBytes))
		return
	}

	fmt.Println()
	fmt.Printf("🔍 Inspecting: %s\n\n", color.YellowString(absPath))

	renderArchitectureBox(diag.Components)
	renderToolchainBox(diag.Tools)
	renderVerdictBox(diag)
}

func inspectFullArchitecture(root string) ProjectDiagnostic {
	var components []DetectedComponent
	toolMap := make(map[string]ToolStatus)

	scanSubprojects(root, root, 0, &components)

	if len(components) == 0 {
		components = append(components, DetectedComponent{
			RelativePath: "./",
			Stack:        "Generic Codebase",
			Anchor:       "Directory",
			DepsOk:       true,
		})
	}

	if fileExists(filepath.Join(root, ".git")) {
		registerTool(toolMap, "git", []string{"--version"}, false, "https://git-scm.com/")
	}

	for _, c := range components {
		st := strings.ToLower(c.Stack)

		if strings.Contains(st, "go lang") {
			registerTool(toolMap, "go", []string{"version"}, true, "brew install go")
		}
		if strings.Contains(st, "node") || strings.Contains(st, "react") || strings.Contains(st, "next") || strings.Contains(st, "vite") {
			registerTool(toolMap, "node", []string{"--version"}, true, "https://nodejs.org/")
			registerTool(toolMap, "npm", []string{"--version"}, true, "https://nodejs.org/")
		}
		if strings.Contains(st, "expo") || strings.Contains(st, "react native") {
			registerTool(toolMap, "expo", []string{"--version"}, false, "npm install -g expo-cli")
			registerTool(toolMap, "adb", []string{"version"}, false, "Android SDK Platform-Tools")
		}
		if strings.Contains(st, "docker") {
			registerTool(toolMap, "docker", []string{"--version"}, true, "https://www.docker.com/")
			registerTool(toolMap, "docker compose", []string{"compose", "version"}, false, "Docker Desktop")
		}
		if strings.Contains(st, "python") {
			registerTool(toolMap, "python3", []string{"--version"}, true, "brew install python")
			registerTool(toolMap, "pip3", []string{"--version"}, false, "python3 -m ensurepip")
		}
		if strings.Contains(st, "rust") {
			registerTool(toolMap, "cargo", []string{"--version"}, true, "https://rustup.rs/")
			registerTool(toolMap, "rustc", []string{"--version"}, true, "https://rustup.rs/")
		}
		if strings.Contains(st, "java") {
			registerTool(toolMap, "java", []string{"-version"}, true, "brew install openjdk")
		}
		if strings.Contains(st, "c++") || strings.Contains(st, "cmake") {
			registerTool(toolMap, "cmake", []string{"--version"}, true, "brew install cmake")
			registerTool(toolMap, "make", []string{"--version"}, true, "xcode-select --install")
		}
		if strings.Contains(st, "prisma") {
			registerTool(toolMap, "npx", []string{"--version"}, true, "npm install -g npm")
		}
	}

	var tools []ToolStatus
	for _, t := range toolMap {
		tools = append(tools, t)
	}

	var missing []string
	allReady := true
	for _, t := range tools {
		if t.Required && !t.Installed {
			missing = append(missing, t.Name)
			allReady = false
		}
	}
	for _, c := range components {
		if !c.DepsOk {
			allReady = false
		}
	}

	return ProjectDiagnostic{
		Components:  components,
		Tools:       tools,
		AllReady:    allReady,
		MissingReqs: missing,
	}
}

func scanSubprojects(root, current string, depth int, components *[]DetectedComponent) {
	if depth > 3 {
		return
	}

	entries, err := os.ReadDir(current)
	if err != nil {
		return
	}

	rel, err := filepath.Rel(root, current)
	if err != nil || rel == "." {
		rel = "./"
	} else {
		rel = "./" + rel
	}

	comp := detectDirComponent(current, rel)
	if comp != nil {
		*components = append(*components, *comp)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if isDoctorSkippedDir(name) || (name == "app" && strings.HasSuffix(current, "android")) {
			continue
		}
		scanSubprojects(root, filepath.Join(current, name), depth+1, components)
	}
}

func detectDirComponent(dir, relPath string) *DetectedComponent {
	hasGoMod := fileExists(filepath.Join(dir, "go.mod"))
	hasPkgJson := fileExists(filepath.Join(dir, "package.json"))
	hasNextConfig := fileExists(filepath.Join(dir, "next.config.js")) || fileExists(filepath.Join(dir, "next.config.ts")) || fileExists(filepath.Join(dir, "next.config.mjs"))
	hasAppJson := fileExists(filepath.Join(dir, "app.json")) || fileExists(filepath.Join(dir, "metro.config.js"))
	hasDockerfile := fileExists(filepath.Join(dir, "Dockerfile")) || fileExists(filepath.Join(dir, "docker-compose.yml")) || fileExists(filepath.Join(dir, "compose.yaml"))
	hasPy := fileExists(filepath.Join(dir, "requirements.txt")) || fileExists(filepath.Join(dir, "pyproject.toml"))
	hasCargo := fileExists(filepath.Join(dir, "Cargo.toml"))
	hasJava := fileExists(filepath.Join(dir, "pom.xml")) || fileExists(filepath.Join(dir, "build.gradle"))
	hasCmake := fileExists(filepath.Join(dir, "CMakeLists.txt"))
	hasPrisma := fileExists(filepath.Join(dir, "schema.prisma")) || fileExists(filepath.Join(dir, "prisma", "schema.prisma"))

	if !hasGoMod && !hasPkgJson && !hasNextConfig && !hasAppJson && !hasDockerfile && !hasPy && !hasCargo && !hasJava && !hasCmake && !hasPrisma {
		return nil
	}

	var stackParts []string
	var anchorParts []string
	depsOk := true
	notice := ""

	if hasGoMod {
		stackParts = append(stackParts, "Go Lang Backend")
		anchorParts = append(anchorParts, "go.mod")
		if !fileExists(filepath.Join(dir, "go.sum")) {
			depsOk = false
			notice = "go.sum missing (run 'go mod tidy')"
		}
	}

	if hasAppJson {
		stackParts = append(stackParts, "React Native (Expo Mobile)")
		anchorParts = append(anchorParts, "app.json")
		if !fileExists(filepath.Join(dir, "node_modules")) {
			depsOk = false
			notice = "node_modules missing (run 'npm install')"
		}
	} else if hasNextConfig {
		stackParts = append(stackParts, "Next.js Web Application")
		anchorParts = append(anchorParts, "next.config")
		if !fileExists(filepath.Join(dir, "node_modules")) {
			depsOk = false
			notice = "node_modules missing (run 'npm install')"
		}
	} else if hasPkgJson {
		stackParts = append(stackParts, "Node.js / React Frontend")
		anchorParts = append(anchorParts, "package.json")
		if !fileExists(filepath.Join(dir, "node_modules")) {
			depsOk = false
			notice = "node_modules missing (run 'npm install')"
		}
	}

	if hasDockerfile {
		stackParts = append(stackParts, "Docker Containerization")
		anchorParts = append(anchorParts, "Dockerfile")
	}

	if hasPrisma {
		stackParts = append(stackParts, "Prisma ORM & Database")
		anchorParts = append(anchorParts, "schema.prisma")
	}

	if hasPy {
		stackParts = append(stackParts, "Python Application")
		anchorParts = append(anchorParts, "requirements.txt")
	}

	if hasCargo {
		stackParts = append(stackParts, "Rust Service")
		anchorParts = append(anchorParts, "Cargo.toml")
	}

	if hasJava {
		stackParts = append(stackParts, "Java Application")
		anchorParts = append(anchorParts, "pom.xml")
	}

	if hasCmake {
		stackParts = append(stackParts, "C/C++ Project")
		anchorParts = append(anchorParts, "CMakeLists.txt")
	}

	return &DetectedComponent{
		RelativePath: relPath,
		Stack:        strings.Join(stackParts, " + "),
		Anchor:       strings.Join(anchorParts, ", "),
		DepsOk:       depsOk,
		DepsNotice:   notice,
	}
}

func registerTool(m map[string]ToolStatus, name string, args []string, req bool, hint string) {
	if existing, ok := m[name]; ok {
		if req {
			existing.Required = true
			m[name] = existing
		}
		return
	}

	cmdName := name
	if strings.Contains(name, " ") {
		parts := strings.Split(name, " ")
		cmdName = parts[0]
	}

	path, err := exec.LookPath(cmdName)
	if err != nil {
		m[name] = ToolStatus{
			Name:        name,
			Installed:   false,
			Required:    req,
			InstallHint: hint,
		}
		return
	}

	out, err := exec.Command(path, args...).CombinedOutput()
	ver := "Available"
	if err == nil {
		ver = cleanVersionOutput(string(out))
	}

	m[name] = ToolStatus{
		Name:        name,
		Installed:   true,
		Version:     ver,
		Required:    req,
		InstallHint: hint,
	}
}

func cleanVersionOutput(raw string) string {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	if len(lines) == 0 {
		return "Available"
	}
	firstLine := strings.TrimSpace(lines[0])

	reVer := regexp.MustCompile(`v?[0-9]+(?:\.[0-9]+)+(?:-[a-zA-Z0-9.]+)?`)
	if m := reVer.FindString(firstLine); m != "" {
		return m
	}

	if len(firstLine) > 18 {
		return firstLine[:18]
	}
	return firstLine
}

func isDoctorSkippedDir(name string) bool {
	skips := map[string]bool{
		".git":         true,
		"node_modules": true,
		"build":        true,
		"dist":         true,
		".next":        true,
		".cache":       true,
		"vendor":       true,
		"__pycache__":  true,
		".venv":        true,
		"venv":         true,
		".gradle":      true,
		"Pods":         true,
		".idea":        true,
		".vscode":      true,
		".expo":        true,
	}
	return skips[name]
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func renderArchitectureBox(components []DetectedComponent) {
	color.Cyan("╭─────────────────────────────────────────────────────────────╮")
	color.Cyan("│                DETECTED PROJECT ARCHITECTURE                │")
	color.Cyan("├───────────────────────┬─────────────────────────────────────┤")
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Subproject / Path", "Tech Stack & Manifests")
	color.Cyan("├───────────────────────┼─────────────────────────────────────┤")

	for _, c := range components {
		pathStr := c.RelativePath
		if len(pathStr) > 21 {
			pathStr = "..." + pathStr[len(pathStr)-18:]
		}

		stackStr := c.Stack
		if len(stackStr) > 35 {
			stackStr = stackStr[:32] + "..."
		}

		fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), color.HiYellowString(pathStr), color.HiGreenString(stackStr))
	}
	color.Cyan("╰───────────────────────┴─────────────────────────────────────╯\n")
}

func renderToolchainBox(tools []ToolStatus) {
	color.Cyan("╭─────────────────────────────────────────────────────────────╮")
	color.Cyan("│               REQUIRED TOOLCHAINS & RUNTIMES                │")
	color.Cyan("├──────────┬──────────────────┬──────────┬────────────────────┤")
	fmt.Printf(color.CyanString("│")+" %-8s "+color.CyanString("│")+" %-16s "+color.CyanString("│")+" %-8s "+color.CyanString("│")+" %-18s "+color.CyanString("│\n"), "Status", "Tool / Runtime", "Required", "Installed Version")
	color.Cyan("├──────────┼──────────────────┼──────────┼────────────────────┤")

	for _, t := range tools {
		statusStr := color.GreenString("  ✅    ")
		verStr := color.HiGreenString("%-18s", t.Version)
		if !t.Installed {
			statusStr = color.RedString("  ❌    ")
			verStr = color.HiRedString("%-18s", "[Not Installed]")
		}

		reqStr := "Yes"
		if !t.Required {
			reqStr = "Optional"
		}

		fmt.Printf(color.CyanString("│")+"%s"+color.CyanString("│")+" %-16s "+color.CyanString("│")+" %-8s "+color.CyanString("│")+" %-18s "+color.CyanString("│\n"),
			statusStr, color.HiWhiteString("%-16s", t.Name), reqStr, verStr)
	}
	color.Cyan("╰──────────┴──────────────────┴──────────┴────────────────────╯\n")
}

func renderVerdictBox(diag ProjectDiagnostic) {
	color.Cyan("╭─────────────────────────────────────────────────────────────╮")
	color.Cyan("│                DEVELOPER ENVIRONMENT STATUS                 │")
	color.Cyan("├─────────────────────────────────────────────────────────────┤")

	installedCount := 0
	totalCount := len(diag.Tools)
	for _, t := range diag.Tools {
		if t.Installed {
			installedCount++
		}
	}

	summaryMsg := fmt.Sprintf("System Status: %d of %d development tools detected on machine.", installedCount, totalCount)
	fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), color.HiWhiteString(summaryMsg))

	if len(diag.MissingReqs) > 0 {
		missingMsg := color.HiRedString("❌ Missing required tools: %s", strings.Join(diag.MissingReqs, ", "))
		fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), missingMsg)
		for _, t := range diag.Tools {
			if t.Required && !t.Installed {
				hintMsg := color.HiYellowString("   👉 Install %s: %s", t.Name, t.InstallHint)
				if len(hintMsg) > 59 {
					hintMsg = hintMsg[:56] + "..."
				}
				fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), hintMsg)
			}
		}
	}

	hasMissingDeps := false
	for _, c := range diag.Components {
		if !c.DepsOk {
			hasMissingDeps = true
			depMsg := color.YellowString("⚠️  %s: %s", c.RelativePath, c.DepsNotice)
			fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), depMsg)
		}
	}

	fmt.Printf(color.CyanString("│")+" %-60s "+color.CyanString("│\n"), "")
	if diag.AllReady {
		readyMsg := color.GreenString("🎉 All required toolchains and packages are ready to build!")
		fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), readyMsg)
	} else if len(diag.MissingReqs) == 0 && hasMissingDeps {
		readyMsg := color.YellowString("💡 All system tools are present! Install package dependencies to build.")
		fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), readyMsg)
	} else {
		readyMsg := color.RedString("⚠️  Environment setup required before you can build this codebase.")
		fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), readyMsg)
	}

	color.Cyan("╰─────────────────────────────────────────────────────────────╯\n")
}
