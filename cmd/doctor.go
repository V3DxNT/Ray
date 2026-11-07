package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/V3DxNT/Ray/internal/detect"
	"github.com/V3DxNT/Ray/internal/fsutil"
	"github.com/V3DxNT/Ray/internal/system"
	"github.com/V3DxNT/Ray/internal/ui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type ProjectDiagnostic struct {
	Components  []detect.Component  `json:"components"`
	Tools       []system.ToolStatus `json:"tools"`
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
	if err != nil || !info.IsDir() {
		color.Red("Target path must be an existing directory")
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
	var components []detect.Component
	toolMap := make(map[string]system.ToolStatus)

	detect.ScanSubprojects(root, root, 0, &components)

	if len(components) == 0 {
		components = append(components, detect.Component{
			RelativePath: "./",
			Stack:        "Generic Codebase",
			Anchor:       "Directory",
			DepsOk:       true,
		})
	}

	if fsutil.DirExists(filepath.Join(root, ".git")) {
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

	var tools []system.ToolStatus
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

func registerTool(m map[string]system.ToolStatus, name string, args []string, required bool, hint string) {
	if _, exists := m[name]; exists {
		return
	}
	m[name] = system.CheckTool(name, args, required, hint)
}

func renderArchitectureBox(components []detect.Component) {
	ui.PrintBoxTop()
	fmt.Printf(color.CyanString("│")+" %-59s "+color.CyanString("│\n"), color.HiWhiteString("DETECTED PROJECT ARCHITECTURE"))
	ui.PrintBoxDivider2ColWide()
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Subproject / Path", "Tech Stack & Manifests")
	ui.PrintBoxDivider2ColWide()

	for _, c := range components {
		pathStr := c.RelativePath
		if len(pathStr) > 21 {
			pathStr = pathStr[:18] + "..."
		}
		stackStr := c.Stack
		if len(stackStr) > 35 {
			stackStr = stackStr[:32] + "..."
		}
		fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), color.HiYellowString(pathStr), color.HiWhiteString(stackStr))
	}
	ui.PrintBoxBottom()
	fmt.Println()
}

func renderToolchainBox(tools []system.ToolStatus) {
	ui.PrintBoxTop()
	fmt.Printf(color.CyanString("│")+" %-59s "+color.CyanString("│\n"), color.HiWhiteString("REQUIRED TOOLCHAINS & RUNTIMES"))
	ui.PrintBoxDivider4Col()
	fmt.Printf(color.CyanString("│")+" %-8s "+color.CyanString("│")+" %-16s "+color.CyanString("│")+" %-8s "+color.CyanString("│")+" %-18s "+color.CyanString("│\n"), "Status", "Tool / Runtime", "Required", "Installed Version")
	ui.PrintBoxDivider4Col()

	for _, t := range tools {
		statusStr := color.GreenString("  ✅    ")
		versionStr := color.HiBlackString(t.Version)
		if !t.Installed {
			statusStr = color.RedString("  ❌    ")
			versionStr = color.RedString("Missing")
		} else {
			versionStr = color.HiGreenString(t.Version)
		}
		reqStr := "Optional"
		if t.Required {
			reqStr = "Yes"
		}

		nameStr := t.Name
		if len(nameStr) > 16 {
			nameStr = nameStr[:13] + "..."
		}
		if len(versionStr) > 18 {
			versionStr = versionStr[:15] + "..."
		}

		fmt.Printf(color.CyanString("│")+"%s"+color.CyanString("│")+" %-16s "+color.CyanString("│")+" %-8s "+color.CyanString("│")+" %-18s "+color.CyanString("│\n"),
			statusStr, nameStr, reqStr, versionStr)
	}
	ui.PrintBoxBottom()
	fmt.Println()
}

func renderVerdictBox(diag ProjectDiagnostic) {
	ui.PrintBoxTop()
	fmt.Printf(color.CyanString("│")+" %-59s "+color.CyanString("│\n"), color.HiWhiteString("DEVELOPER ENVIRONMENT STATUS"))
	ui.PrintBoxDivider()

	installedCount := 0
	for _, t := range diag.Tools {
		if t.Installed {
			installedCount++
		}
	}
	statusSummary := fmt.Sprintf("System Status: %d of %d development tools detected on machine.", installedCount, len(diag.Tools))
	fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), color.HiWhiteString(statusSummary))

	hasDepNotices := false
	for _, c := range diag.Components {
		if !c.DepsOk && c.Notice != "" {
			noticeStr := fmt.Sprintf("⚠️  %s: %s", c.RelativePath, c.Notice)
			if len(noticeStr) > 59 {
				noticeStr = noticeStr[:56] + "..."
			}
			fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), color.YellowString(noticeStr))
			hasDepNotices = true
		}
	}

	fmt.Printf(color.CyanString("│")+" %-59s "+color.CyanString("│\n"), "")
	if len(diag.MissingReqs) > 0 {
		missMsg := fmt.Sprintf("❌ Missing required system tools: %s. Run 'ray setup' to resolve.", strings.Join(diag.MissingReqs, ", "))
		if len(missMsg) > 59 {
			missMsg = missMsg[:56] + "..."
		}
		fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), color.HiRedString(missMsg))
	} else if hasDepNotices {
		fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), color.HiGreenString("💡 All system tools are present! Install package dependencies to build."))
	} else {
		fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), color.HiGreenString("🎉 Architecture & Toolchain fully verified. Ready to run and build!"))
	}
	ui.PrintBoxBottom()
	fmt.Println()
}
