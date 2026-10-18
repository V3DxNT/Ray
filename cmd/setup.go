package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type SetupTool struct {
	Name         string `json:"name"`
	Installed    bool   `json:"installed"`
	Version      string `json:"version,omitempty"`
	InstallCmd   string `json:"install_command"`
	EstimatedMB  int    `json:"estimated_mb"`
	Category     string `json:"category"`
	RequiredFor  string `json:"required_for"`
	InstallState string `json:"install_state,omitempty"`
}

type ProjectDepTask struct {
	SubprojectPath string `json:"subproject_path"`
	Manager        string `json:"manager"`
	Command        string `json:"command"`
	Executed       bool   `json:"executed"`
	Success        bool   `json:"success"`
}

type SetupPlan struct {
	OS             string           `json:"operating_system"`
	PackageManager string           `json:"package_manager"`
	PkgMgrReady    bool             `json:"package_manager_ready"`
	MissingTools   []SetupTool      `json:"missing_tools"`
	ReadyTools     []SetupTool      `json:"ready_tools"`
	TotalSpaceMB   int              `json:"total_estimated_space_mb"`
	DepTasks       []ProjectDepTask `json:"dependency_tasks"`
}

var (
	setupAutoConfirm bool
	setupDryRun      bool
	setupSystemOnly  bool
	setupDepsOnly    bool
	setupJson        bool
)

var setupCmd = &cobra.Command{
	Use:   "setup [path]",
	Short: "Inspect requirements, install missing toolchains and bootstrap project dependencies",
	Run:   runSetup,
}

func init() {
	setupCmd.Flags().BoolVarP(&setupAutoConfirm, "yes", "y", false, "Auto-confirm all installation prompts")
	setupCmd.Flags().BoolVar(&setupDryRun, "dry-run", false, "Display setup execution plan without installing")
	setupCmd.Flags().BoolVar(&setupSystemOnly, "system-only", false, "Install missing system tools only, skipping package dependencies")
	setupCmd.Flags().BoolVar(&setupDepsOnly, "deps-only", false, "Install project package dependencies only, skipping system tools")
	setupCmd.Flags().BoolVar(&setupJson, "json", false, "Output setup plan and results in JSON format")
	RootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) {
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

	plan := buildSetupPlan(absPath)

	if setupJson {
		if !setupDryRun && (len(plan.MissingTools) > 0 || len(plan.DepTasks) > 0) && setupAutoConfirm {
			executeSetupPlan(&plan)
		}
		jsonBytes, err := json.MarshalIndent(plan, "", "  ")
		if err != nil {
			color.Red("Error marshaling json: %v", err)
			return
		}
		fmt.Println(string(jsonBytes))
		return
	}

	fmt.Println()
	fmt.Printf("🔍 Bootstrapping Setup: %s\n\n", color.YellowString(absPath))

	renderPrerequisitesBox(plan)
	renderSetupPlanBox(plan)

	if len(plan.MissingTools) == 0 && len(plan.DepTasks) == 0 {
		color.Green("╭─────────────────────────────────────────────────────────────╮")
		color.Green("│                    ALL SYSTEMS READY                        │")
		color.Green("├─────────────────────────────────────────────────────────────┤")
		fmt.Printf(color.GreenString("│")+" %-69s "+color.GreenString("│\n"), "🎉 Codebase is fully provisioned! No missing tools or deps.")
		color.Green("╰─────────────────────────────────────────────────────────────╯\n")
		return
	}

	if setupDryRun {
		color.Cyan("\n[Dry Run Active] No packages or tools were downloaded or installed.\n")
		return
	}

	shouldProceed := setupAutoConfirm
	if !shouldProceed {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Proceed with automated setup and installations? [y/N]: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		input = strings.TrimSpace(strings.ToLower(input))
		shouldProceed = input == "y" || input == "yes"
	}

	if !shouldProceed {
		color.Yellow("\nSetup cancelled by user.")
		return
	}

	executeSetupPlan(&plan)
	renderSetupReportBox(plan)
}

func buildSetupPlan(root string) SetupPlan {
	osName := runtime.GOOS
	pkgMgr, pkgReady := detectPackageManager(osName)

	var components []DetectedComponent
	scanSubprojects(root, root, 0, &components)

	if len(components) == 0 {
		components = append(components, DetectedComponent{
			RelativePath: "./",
			Stack:        "Generic Codebase",
			Anchor:       "Directory",
			DepsOk:       true,
		})
	}

	neededToolNames := make(map[string]string)

	if fileExists(filepath.Join(root, ".git")) {
		neededToolNames["git"] = "Version Control"
	}

	var depTasks []ProjectDepTask

	for _, c := range components {
		st := strings.ToLower(c.Stack)
		subDir := filepath.Join(root, strings.TrimPrefix(c.RelativePath, "./"))

		if strings.Contains(st, "go lang") {
			neededToolNames["go"] = "Go Backend"
			if !c.DepsOk && !setupSystemOnly {
				depTasks = append(depTasks, ProjectDepTask{
					SubprojectPath: c.RelativePath,
					Manager:        "go",
					Command:        "go mod tidy",
				})
			}
		}

		if strings.Contains(st, "node") || strings.Contains(st, "next") || strings.Contains(st, "react") || strings.Contains(st, "vite") {
			neededToolNames["node"] = "JS Runtime"
			neededToolNames["npm"] = "Node Package Manager"
			if !c.DepsOk && !setupSystemOnly {
				cmdStr := "npm install"
				if fileExists(filepath.Join(subDir, "yarn.lock")) {
					neededToolNames["yarn"] = "Yarn Package Manager"
					cmdStr = "yarn install"
				} else if fileExists(filepath.Join(subDir, "pnpm-lock.yaml")) {
					neededToolNames["pnpm"] = "pnpm Package Manager"
					cmdStr = "pnpm install"
				} else if fileExists(filepath.Join(subDir, "bun.lockb")) || fileExists(filepath.Join(subDir, "bun.lock")) {
					neededToolNames["bun"] = "Bun Runtime"
					cmdStr = "bun install"
				}
				depTasks = append(depTasks, ProjectDepTask{
					SubprojectPath: c.RelativePath,
					Manager:        "npm",
					Command:        cmdStr,
				})
			}
		}

		if strings.Contains(st, "expo") || strings.Contains(st, "react native") {
			neededToolNames["expo"] = "Expo CLI"
			neededToolNames["adb"] = "Android Debug Bridge"
		}

		if strings.Contains(st, "docker") {
			neededToolNames["docker"] = "Container Engine"
		}

		if strings.Contains(st, "python") {
			neededToolNames["python3"] = "Python Runtime"
			neededToolNames["pip3"] = "Python Package Installer"
			reqFile := filepath.Join(subDir, "requirements.txt")
			if fileExists(reqFile) && !setupSystemOnly {
				depTasks = append(depTasks, ProjectDepTask{
					SubprojectPath: c.RelativePath,
					Manager:        "pip",
					Command:        "pip3 install -r requirements.txt",
				})
			}
		}

		if strings.Contains(st, "rust") {
			neededToolNames["cargo"] = "Rust Package Manager"
			neededToolNames["rustc"] = "Rust Compiler"
		}

		if strings.Contains(st, "java") {
			neededToolNames["java"] = "Java Development Kit"
		}

		if strings.Contains(st, "c++") || strings.Contains(st, "cmake") {
			neededToolNames["cmake"] = "CMake Build System"
			neededToolNames["make"] = "Make Tool"
		}

		if strings.Contains(st, "prisma") {
			neededToolNames["npx"] = "Node Runner"
			if !setupSystemOnly {
				depTasks = append(depTasks, ProjectDepTask{
					SubprojectPath: c.RelativePath,
					Manager:        "prisma",
					Command:        "npx prisma generate",
				})
			}
		}

		if fileExists(filepath.Join(subDir, "k8s")) || fileExists(filepath.Join(subDir, "kubernetes")) || fileExists(filepath.Join(subDir, "helm")) {
			neededToolNames["kubectl"] = "Kubernetes CLI"
		}
	}

	var missing []SetupTool
	var ready []SetupTool
	totalSpace := 0

	for tool, context := range neededToolNames {
		info := inspectSetupTool(tool, context, osName, pkgMgr)
		if info.Installed {
			ready = append(ready, info)
		} else {
			if !setupDepsOnly {
				missing = append(missing, info)
				totalSpace += info.EstimatedMB
			}
		}
	}

	return SetupPlan{
		OS:             osName,
		PackageManager: pkgMgr,
		PkgMgrReady:    pkgReady,
		MissingTools:   missing,
		ReadyTools:     ready,
		TotalSpaceMB:   totalSpace,
		DepTasks:       depTasks,
	}
}

func detectPackageManager(osName string) (string, bool) {
	switch osName {
	case "darwin":
		if _, err := exec.LookPath("brew"); err == nil {
			return "Homebrew (brew)", true
		}
		return "Homebrew", false
	case "linux":
		if _, err := exec.LookPath("apt-get"); err == nil {
			return "APT (apt-get)", true
		}
		if _, err := exec.LookPath("dnf"); err == nil {
			return "DNF", true
		}
		if _, err := exec.LookPath("pacman"); err == nil {
			return "Pacman", true
		}
		if _, err := exec.LookPath("yum"); err == nil {
			return "YUM", true
		}
		return "APT", false
	case "windows":
		if _, err := exec.LookPath("winget"); err == nil {
			return "WinGet", true
		}
		if _, err := exec.LookPath("choco"); err == nil {
			return "Chocolatey", true
		}
		if _, err := exec.LookPath("scoop"); err == nil {
			return "Scoop", true
		}
		return "WinGet / Chocolatey", false
	default:
		return "Package Manager", true
	}
}

func inspectSetupTool(name, context, osName, pkgMgr string) SetupTool {
	cmdName := name
	if strings.Contains(name, " ") {
		cmdName = strings.Split(name, " ")[0]
	}

	path, err := exec.LookPath(cmdName)
	installed := err == nil
	ver := ""

	if installed {
		args := []string{"--version"}
		if name == "go" {
			args = []string{"version"}
		} else if name == "java" {
			args = []string{"-version"}
		}
		out, err := exec.Command(path, args...).CombinedOutput()
		if err == nil {
			ver = cleanVersionOutput(string(out))
		} else {
			ver = "Installed"
		}
	}

	installCmd, estMB := getInstallRecipe(name, osName, pkgMgr)

	return SetupTool{
		Name:        name,
		Installed:   installed,
		Version:     ver,
		InstallCmd:  installCmd,
		EstimatedMB: estMB,
		Category:    categorizeTool(name),
		RequiredFor: context,
	}
}

func categorizeTool(name string) string {
	switch name {
	case "go", "node", "python3", "java", "rustc":
		return "Compiler / Runtime"
	case "npm", "yarn", "pnpm", "bun", "pip3", "cargo":
		return "Package Manager"
	case "docker", "docker compose":
		return "Container System"
	case "git", "kubectl", "cmake", "make", "expo", "adb", "npx":
		return "Developer CLI"
	default:
		return "Utility"
	}
}

func getInstallRecipe(name, osName, pkgMgr string) (string, int) {
	if osName == "darwin" {
		switch name {
		case "go":
			return "brew install go", 250
		case "node", "npm", "npx":
			return "brew install node", 120
		case "yarn":
			return "npm install -g yarn", 15
		case "pnpm":
			return "brew install pnpm", 25
		case "bun":
			return "curl -fsSL https://bun.sh/install | bash", 40
		case "docker", "docker compose":
			return "brew install --cask docker", 1200
		case "python3", "pip3":
			return "brew install python", 180
		case "rustc", "cargo":
			return "brew install rust", 320
		case "java":
			return "brew install openjdk", 350
		case "cmake":
			return "brew install cmake", 85
		case "make":
			return "xcode-select --install", 200
		case "git":
			return "brew install git", 65
		case "kubectl":
			return "brew install kubectl", 60
		case "expo":
			return "npm install -g expo-cli", 45
		case "adb":
			return "brew install --cask android-platform-tools", 95
		default:
			return "brew install " + name, 50
		}
	} else if osName == "linux" {
		switch name {
		case "go":
			return "sudo apt-get install -y golang-go", 240
		case "node", "npm", "npx":
			return "sudo apt-get install -y nodejs npm", 110
		case "yarn":
			return "npm install -g yarn", 15
		case "pnpm":
			return "npm install -g pnpm", 25
		case "bun":
			return "curl -fsSL https://bun.sh/install | bash", 40
		case "docker", "docker compose":
			return "sudo apt-get install -y docker.io docker-compose", 900
		case "python3", "pip3":
			return "sudo apt-get install -y python3 python3-pip", 140
		case "rustc", "cargo":
			return "curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh", 300
		case "java":
			return "sudo apt-get install -y default-jdk", 320
		case "cmake":
			return "sudo apt-get install -y cmake", 75
		case "make":
			return "sudo apt-get install -y build-essential", 180
		case "git":
			return "sudo apt-get install -y git", 50
		case "kubectl":
			return "sudo apt-get install -y kubectl", 55
		case "expo":
			return "npm install -g expo-cli", 45
		case "adb":
			return "sudo apt-get install -y adb", 80
		default:
			return "sudo apt-get install -y " + name, 50
		}
	} else {
		switch name {
		case "go":
			return "winget install GoLang.Go", 250
		case "node", "npm", "npx":
			return "winget install OpenJS.NodeJS", 120
		case "docker", "docker compose":
			return "winget install Docker.DockerDesktop", 1500
		case "python3", "pip3":
			return "winget install Python.Python.3", 160
		case "git":
			return "winget install Git.Git", 70
		default:
			return "winget install " + name, 50
		}
	}
}

func renderPrerequisitesBox(plan SetupPlan) {
	color.Cyan("╭─────────────────────────────────────────────────────────────╮")
	color.Cyan("│                DETECTED STACK & PREREQUISITES               │")
	color.Cyan("├───────────────────────┬─────────────────────────────────────┤")
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Host OS / Arch", color.HiYellowString(plan.OS+" / "+runtime.GOARCH))

	pkgStatus := color.HiGreenString(plan.PackageManager + " (Ready)")
	if !plan.PkgMgrReady {
		pkgStatus = color.HiRedString(plan.PackageManager + " [Not Installed]")
	}
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Package Manager", pkgStatus)

	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Installed Toolchains", color.HiWhiteString("%d Ready", len(plan.ReadyTools)))
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Missing Toolchains", color.HiRedString("%d Required", len(plan.MissingTools)))
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Pending Dep Tasks", color.HiMagentaString("%d Subproject(s)", len(plan.DepTasks)))
	color.Cyan("╰───────────────────────┴─────────────────────────────────────╯\n")
}

func renderSetupPlanBox(plan SetupPlan) {
	color.Cyan("╭─────────────────────────────────────────────────────────────╮")
	color.Cyan("│               SYSTEM REQUIREMENTS & AUTOMATION              │")
	color.Cyan("├──────────┬──────────────────┬──────────┬────────────────────┤")
	fmt.Printf(color.CyanString("│")+" %-8s "+color.CyanString("│")+" %-16s "+color.CyanString("│")+" %-8s "+color.CyanString("│")+" %-18s "+color.CyanString("│\n"), "Status", "Tool / Action", "Size (MB)", "Execution Recipe")
	color.Cyan("├──────────┼──────────────────┼──────────┼────────────────────┤")

	for _, t := range plan.ReadyTools {
		statusStr := color.GreenString("  ✅    ")
		nameStr := color.HiWhiteString("%-16s", t.Name)
		sizeStr := color.HiBlackString("%-8s", "Cached")
		recipeStr := color.HiBlackString("%-18s", t.Version)
		fmt.Printf(color.CyanString("│")+"%s"+color.CyanString("│")+" %-16s "+color.CyanString("│")+" %-8s "+color.CyanString("│")+" %-18s "+color.CyanString("│\n"),
			statusStr, nameStr, sizeStr, recipeStr)
	}

	for _, t := range plan.MissingTools {
		statusStr := color.RedString("  ❌    ")
		nameStr := color.HiRedString("%-16s", t.Name)
		sizeStr := color.YellowString("~%d MB", t.EstimatedMB)
		recipeStr := t.InstallCmd
		if len(recipeStr) > 18 {
			recipeStr = recipeStr[:15] + "..."
		}
		fmt.Printf(color.CyanString("│")+"%s"+color.CyanString("│")+" %-16s "+color.CyanString("│")+" %-8s "+color.CyanString("│")+" %-18s "+color.CyanString("│\n"),
			statusStr, nameStr, sizeStr, color.HiCyanString("%-18s", recipeStr))
	}

	for _, d := range plan.DepTasks {
		statusStr := color.YellowString("  📦    ")
		nameStr := color.HiYellowString("%-16s", d.Manager)
		sizeStr := color.HiBlackString("%-8s", "Varies")
		recipeStr := d.Command
		if len(recipeStr) > 18 {
			recipeStr = recipeStr[:15] + "..."
		}
		fmt.Printf(color.CyanString("│")+"%s"+color.CyanString("│")+" %-16s "+color.CyanString("│")+" %-8s "+color.CyanString("│")+" %-18s "+color.CyanString("│\n"),
			statusStr, nameStr, sizeStr, color.HiWhiteString("%-18s", recipeStr))
	}

	color.Cyan("├─────────────────────────────────────────────────────────────┤")
	spaceStr := fmt.Sprintf("Estimated Total Download Space Required: ~%d MB", plan.TotalSpaceMB)
	fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), color.HiYellowString(spaceStr))
	color.Cyan("╰─────────────────────────────────────────────────────────────╯\n")
}

func executeSetupPlan(plan *SetupPlan) {
	fmt.Println()
	color.Cyan("⚙  Starting automated toolchain and dependency bootstrap...")

	if len(plan.MissingTools) > 0 {
		ensurePackageManagerReady(plan)
	}

	for i := range plan.MissingTools {
		t := &plan.MissingTools[i]
		fmt.Printf("\n▶ Installing %s via '%s'...\n", color.YellowString(t.Name), color.HiCyanString(t.InstallCmd))

		parts := strings.Fields(t.InstallCmd)
		if len(parts) == 0 {
			t.InstallState = "Skipped"
			continue
		}

		cmd := exec.Command(parts[0], parts[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			t.InstallState = "Failed"
			color.Red("✖ Installation failed for %s: %v", t.Name, err)
		} else {
			t.InstallState = "Installed"
			color.Green("✔ %s successfully installed!", t.Name)
		}
	}

	for i := range plan.DepTasks {
		d := &plan.DepTasks[i]
		fmt.Printf("\n▶ Bootstrapping dependencies in %s ('%s')...\n", color.YellowString(d.SubprojectPath), color.HiCyanString(d.Command))
		parts := strings.Fields(d.Command)
		if len(parts) == 0 {
			continue
		}

		cmd := exec.Command(parts[0], parts[1:]...)
		cmd.Dir = d.SubprojectPath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		d.Executed = true
		if err != nil {
			d.Success = false
			color.Red("✖ Dependency install failed in %s: %v", d.SubprojectPath, err)
		} else {
			d.Success = true
			color.Green("✔ Dependencies successfully bootstrapped in %s!", d.SubprojectPath)
		}
	}
	fmt.Println()
}

func ensurePackageManagerReady(plan *SetupPlan) bool {
	if plan.PkgMgrReady {
		return true
	}

	recipe, desc := getPackageManagerInstallRecipe(plan.OS)
	if recipe == "" {
		return false
	}

	color.Yellow("\n⚠️  Package manager (%s) is required to install toolchains but is not present.", desc)
	fmt.Printf("Automated installation command: %s\n", color.CyanString(recipe))

	proceed := setupAutoConfirm
	if !proceed {
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Install %s now using the CLI? [y/N]: ", desc)
		input, err := reader.ReadString('\n')
		if err != nil {
			return false
		}
		input = strings.TrimSpace(strings.ToLower(input))
		proceed = input == "y" || input == "yes"
	}

	if !proceed {
		color.Yellow("Skipped installing %s. Tool installations that rely on it may fail.", desc)
		return false
	}

	color.Cyan("▶ Installing %s...", desc)
	var cmd *exec.Cmd
	if plan.OS == "darwin" {
		cmd = exec.Command("/bin/bash", "-c", recipe)
	} else if plan.OS == "windows" {
		cmd = exec.Command("powershell", "-Command", recipe)
	} else {
		parts := strings.Fields(recipe)
		cmd = exec.Command(parts[0], parts[1:]...)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err != nil {
		color.Red("✖ Failed to install package manager: %v", err)
		return false
	}

	if plan.OS == "darwin" {
		_ = os.Setenv("PATH", os.Getenv("PATH")+":/opt/homebrew/bin:/usr/local/bin")
	}

	plan.PkgMgrReady = true
	color.Green("✔ %s successfully installed!", desc)
	return true
}

func getPackageManagerInstallRecipe(osName string) (string, string) {
	switch osName {
	case "darwin":
		return `/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`, "Homebrew"
	case "windows":
		return `Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))`, "Chocolatey"
	case "linux":
		return "sudo apt-get update && sudo apt-get install -y software-properties-common", "APT"
	default:
		return "", ""
	}
}

func renderSetupReportBox(plan SetupPlan) {
	color.Cyan("╭─────────────────────────────────────────────────────────────╮")
	color.Cyan("│                  SETUP INSTALLATION REPORT                  │")
	color.Cyan("├───────────────────────┬──────────┬──────────────────────────┤")
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-8s "+color.CyanString("│")+" %-24s "+color.CyanString("│\n"), "Component / Tool", "Status", "Result Details")
	color.Cyan("├───────────────────────┼──────────┼──────────────────────────┤")

	for _, t := range plan.MissingTools {
		statusStr := color.GreenString("  ✅    ")
		detailStr := color.HiGreenString("Successfully Installed")
		if t.InstallState == "Failed" {
			statusStr = color.RedString("  ❌    ")
			detailStr = color.HiRedString("Installation Failed")
		} else if t.InstallState == "Skipped" {
			statusStr = color.YellowString("  ↷    ")
			detailStr = color.YellowString("Skipped")
		}
		fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+"%s"+color.CyanString("│")+" %-33s "+color.CyanString("│\n"),
			color.HiWhiteString(t.Name), statusStr, detailStr)
	}

	for _, d := range plan.DepTasks {
		statusStr := color.GreenString("  ✅    ")
		detailStr := color.HiGreenString("Dependencies Synced")
		if !d.Success {
			statusStr = color.RedString("  ❌    ")
			detailStr = color.HiRedString("Failed: " + d.Command)
		}
		taskName := d.Manager + " (" + d.SubprojectPath + ")"
		if len(taskName) > 21 {
			taskName = taskName[:18] + "..."
		}
		fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+"%s"+color.CyanString("│")+" %-33s "+color.CyanString("│\n"),
			color.HiWhiteString(taskName), statusStr, detailStr)
	}

	color.Cyan("├─────────────────────────────────────────────────────────────┤")
	fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), color.GreenString("🎉 Setup cycle complete. Run 'ray doctor' to confirm readiness!"))
	color.Cyan("╰─────────────────────────────────────────────────────────────╯\n")
}
