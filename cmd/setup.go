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

	"github.com/V3DxNT/Ray/internal/detect"
	"github.com/V3DxNT/Ray/internal/fsutil"
	"github.com/V3DxNT/Ray/internal/system"
	"github.com/V3DxNT/Ray/internal/ui"
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
	if err != nil || !info.IsDir() {
		color.Red("Target path must be an existing directory")
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
		ui.PrintBoxTop()
		fmt.Printf(color.GreenString("│")+" %-59s "+color.GreenString("│\n"), "ALL SYSTEMS READY")
		ui.PrintBoxDivider()
		fmt.Printf(color.GreenString("│")+" %-69s "+color.GreenString("│\n"), "🎉 Codebase is fully provisioned! No missing tools or deps.")
		ui.PrintBoxBottom()
		fmt.Println()
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
	pkgMgr, pkgReady := system.DetectPackageManager(osName)

	var components []detect.Component
	detect.ScanSubprojects(root, root, 0, &components)

	if len(components) == 0 {
		components = append(components, detect.Component{
			RelativePath: "./",
			Stack:        "Generic Codebase",
			Anchor:       "Directory",
			DepsOk:       true,
		})
	}

	neededToolNames := make(map[string]string)

	if fsutil.DirExists(filepath.Join(root, ".git")) {
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
				if fsutil.FileExists(filepath.Join(subDir, "yarn.lock")) {
					neededToolNames["yarn"] = "Yarn Package Manager"
					cmdStr = "yarn install"
				} else if fsutil.FileExists(filepath.Join(subDir, "pnpm-lock.yaml")) {
					neededToolNames["pnpm"] = "pnpm Package Manager"
					cmdStr = "pnpm install"
				} else if fsutil.FileExists(filepath.Join(subDir, "bun.lockb")) || fsutil.FileExists(filepath.Join(subDir, "bun.lock")) {
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
			if fsutil.FileExists(reqFile) && !setupSystemOnly {
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

		if fsutil.DirExists(filepath.Join(subDir, "k8s")) || fsutil.DirExists(filepath.Join(subDir, "kubernetes")) || fsutil.DirExists(filepath.Join(subDir, "helm")) {
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
			ver = system.CleanVersion(string(out))
		} else {
			ver = "Installed"
		}
	}

	installCmd, estMB := system.GetInstallRecipe(name, osName, pkgMgr)

	return SetupTool{
		Name:        name,
		Installed:   installed,
		Version:     ver,
		InstallCmd:  installCmd,
		EstimatedMB: estMB,
		Category:    system.CategorizeTool(name),
		RequiredFor: context,
	}
}

func renderPrerequisitesBox(plan SetupPlan) {
	ui.PrintBoxTop()
	fmt.Printf(color.CyanString("│")+" %-59s "+color.CyanString("│\n"), color.HiWhiteString("DETECTED STACK & PREREQUISITES"))
	ui.PrintBoxDivider2ColWide()
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Host OS / Arch", color.HiYellowString(plan.OS+" / "+runtime.GOARCH))

	pkgStatus := color.HiGreenString(plan.PackageManager + " (Ready)")
	if !plan.PkgMgrReady {
		pkgStatus = color.HiRedString(plan.PackageManager + " [Not Installed]")
	}
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Package Manager", pkgStatus)

	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Installed Toolchains", color.HiWhiteString("%d Ready", len(plan.ReadyTools)))
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Missing Toolchains", color.HiRedString("%d Required", len(plan.MissingTools)))
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Pending Dep Tasks", color.HiMagentaString("%d Subproject(s)", len(plan.DepTasks)))
	ui.PrintBoxBottom()
	fmt.Println()
}

func renderSetupPlanBox(plan SetupPlan) {
	ui.PrintBoxTop()
	fmt.Printf(color.CyanString("│")+" %-59s "+color.CyanString("│\n"), color.HiWhiteString("SYSTEM REQUIREMENTS & AUTOMATION"))
	ui.PrintBoxDivider4Col()
	fmt.Printf(color.CyanString("│")+" %-8s "+color.CyanString("│")+" %-16s "+color.CyanString("│")+" %-8s "+color.CyanString("│")+" %-18s "+color.CyanString("│\n"), "Status", "Tool / Action", "Size (MB)", "Execution Recipe")
	ui.PrintBoxDivider4Col()

	for _, t := range plan.ReadyTools {
		statusStr := color.GreenString("  ✅    ")
		nameStr := color.HiWhiteString("%-16s", ui.Truncate(t.Name, 16))
		sizeStr := color.HiBlackString("%-8s", "Cached")
		recipeStr := color.HiBlackString("%-18s", ui.Truncate(t.Version, 18))
		fmt.Printf(color.CyanString("│")+"%s"+color.CyanString("│")+" %-16s "+color.CyanString("│")+" %-8s "+color.CyanString("│")+" %-18s "+color.CyanString("│\n"),
			statusStr, nameStr, sizeStr, recipeStr)
	}

	for _, t := range plan.MissingTools {
		statusStr := color.RedString("  ❌    ")
		nameStr := color.HiRedString("%-16s", ui.Truncate(t.Name, 16))
		sizeStr := color.YellowString("~%d MB", t.EstimatedMB)
		recipeStr := ui.Truncate(t.InstallCmd, 18)
		fmt.Printf(color.CyanString("│")+"%s"+color.CyanString("│")+" %-16s "+color.CyanString("│")+" %-8s "+color.CyanString("│")+" %-18s "+color.CyanString("│\n"),
			statusStr, nameStr, sizeStr, color.HiCyanString("%-18s", recipeStr))
	}

	for _, d := range plan.DepTasks {
		statusStr := color.YellowString("  📦    ")
		nameStr := color.HiYellowString("%-16s", ui.Truncate(d.Manager, 16))
		sizeStr := color.HiBlackString("%-8s", "Varies")
		recipeStr := ui.Truncate(d.Command, 18)
		fmt.Printf(color.CyanString("│")+"%s"+color.CyanString("│")+" %-16s "+color.CyanString("│")+" %-8s "+color.CyanString("│")+" %-18s "+color.CyanString("│\n"),
			statusStr, nameStr, sizeStr, color.HiWhiteString("%-18s", recipeStr))
	}

	ui.PrintBoxDivider()
	spaceStr := fmt.Sprintf("Estimated Total Download Space Required: ~%d MB", plan.TotalSpaceMB)
	fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), color.HiYellowString(spaceStr))
	ui.PrintBoxBottom()
	fmt.Println()
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
		fmt.Print("Install missing package manager now? [y/N]: ")
		in, err := reader.ReadString('\n')
		if err != nil {
			return false
		}
		in = strings.TrimSpace(strings.ToLower(in))
		proceed = in == "y" || in == "yes"
	}

	if !proceed {
		color.Red("Cannot install system toolchains without package manager.")
		return false
	}

	fmt.Printf("Running: %s\n", recipe)
	cmd := exec.Command("bash", "-c", recipe)
	if plan.OS == "windows" {
		cmd = exec.Command("powershell", "-c", recipe)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		color.Red("Failed to install package manager: %v", err)
		return false
	}

	plan.PkgMgrReady = true
	color.Green("✔ Package manager successfully installed!")
	return true
}

func getPackageManagerInstallRecipe(osName string) (string, string) {
	switch osName {
	case "darwin":
		return `/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`, "Homebrew"
	case "linux":
		return "sudo apt-get update && sudo apt-get install -y build-essential curl", "APT (Debian/Ubuntu)"
	case "windows":
		return "Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))", "Chocolatey"
	default:
		return "", "Package Manager"
	}
}

func renderSetupReportBox(plan SetupPlan) {
	ui.PrintBoxTop()
	fmt.Printf(color.CyanString("│")+" %-59s "+color.CyanString("│\n"), color.HiWhiteString("SETUP EXECUTION REPORT"))
	ui.PrintBoxDivider2ColWide()

	toolsSuccess := 0
	for _, t := range plan.MissingTools {
		if t.InstallState == "Installed" {
			toolsSuccess++
		}
	}

	depsSuccess := 0
	for _, d := range plan.DepTasks {
		if d.Success {
			depsSuccess++
		}
	}

	toolReport := fmt.Sprintf("%d of %d installed", toolsSuccess, len(plan.MissingTools))
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "System Toolchains", color.HiGreenString(toolReport))

	depReport := fmt.Sprintf("%d of %d synced", depsSuccess, len(plan.DepTasks))
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Dependency Tasks", color.HiGreenString(depReport))

	status := color.HiGreenString("Complete & Operational")
	if toolsSuccess < len(plan.MissingTools) || depsSuccess < len(plan.DepTasks) {
		status = color.HiYellowString("Completed with Warnings")
	}
	fmt.Printf(color.CyanString("│")+" %-21s "+color.CyanString("│")+" %-35s "+color.CyanString("│\n"), "Execution Status", status)
	ui.PrintBoxBottom()
	fmt.Println()
}
