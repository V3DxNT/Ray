package system

import (
	"os/exec"
	"regexp"
	"strings"
)

type ToolStatus struct {
	Name        string `json:"name"`
	Installed   bool   `json:"installed"`
	Version     string `json:"version,omitempty"`
	Required    bool   `json:"required"`
	InstallHint string `json:"install_hint"`
}

func CheckTool(name string, args []string, required bool, hint string) ToolStatus {
	path, err := exec.LookPath(name)
	if err != nil {
		return ToolStatus{
			Name:        name,
			Installed:   false,
			Required:    required,
			InstallHint: hint,
		}
	}

	ver := "Installed"
	if len(args) > 0 {
		out, err := exec.Command(path, args...).CombinedOutput()
		if err == nil {
			ver = CleanVersion(string(out))
		}
	}

	return ToolStatus{
		Name:        name,
		Installed:   true,
		Version:     ver,
		Required:    required,
		InstallHint: hint,
	}
}

func CleanVersion(raw string) string {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	firstLine := ""
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" {
			firstLine = trimmed
			break
		}
	}

	reSemver := regexp.MustCompile(`v?[0-9]+(\.[0-9]+)+(-[a-zA-Z0-9.]+)?`)
	match := reSemver.FindString(firstLine)
	if match != "" {
		return match
	}

	if len(firstLine) > 20 {
		return firstLine[:17] + "..."
	}
	return firstLine
}

func DetectPackageManager(osName string) (string, bool) {
	switch osName {
	case "darwin":
		_, err := exec.LookPath("brew")
		return "Homebrew (brew)", err == nil
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
		return "APT / Package Manager", false
	case "windows":
		if _, err := exec.LookPath("winget"); err == nil {
			return "Windows Package Manager (winget)", true
		}
		if _, err := exec.LookPath("choco"); err == nil {
			return "Chocolatey (choco)", true
		}
		return "winget / Chocolatey", false
	default:
		return "System Package Manager", false
	}
}

func CategorizeTool(name string) string {
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

func GetInstallRecipe(name, osName, pkgMgr string) (string, int) {
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
			return "sudo npm install -g pnpm", 25
		case "bun":
			return "curl -fsSL https://bun.sh/install | bash", 40
		case "docker", "docker compose":
			return "sudo apt-get install -y docker.io docker-compose", 800
		case "python3", "pip3":
			return "sudo apt-get install -y python3 python3-pip", 160
		case "rustc", "cargo":
			return "curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh", 300
		case "java":
			return "sudo apt-get install -y default-jdk", 320
		case "cmake":
			return "sudo apt-get install -y cmake", 80
		case "make":
			return "sudo apt-get install -y build-essential", 150
		case "git":
			return "sudo apt-get install -y git", 60
		case "kubectl":
			return "sudo apt-get install -y kubectl", 55
		case "expo":
			return "npm install -g expo-cli", 45
		case "adb":
			return "sudo apt-get install -y adb", 90
		default:
			return "sudo apt-get install -y " + name, 50
		}
	} else {
		switch name {
		case "go":
			return "winget install GoLang.Go", 250
		case "node", "npm", "npx":
			return "winget install OpenJS.NodeJS", 130
		case "yarn":
			return "npm install -g yarn", 15
		case "pnpm":
			return "npm install -g pnpm", 25
		case "bun":
			return "powershell -c \"irm bun.sh/install.ps1 | iex\"", 40
		case "docker", "docker compose":
			return "winget install Docker.DockerDesktop", 1500
		case "python3", "pip3":
			return "winget install Python.Python.3.11", 170
		case "rustc", "cargo":
			return "winget install Rustlang.Rustup", 310
		case "java":
			return "winget install Oracle.JDK.21", 340
		case "cmake":
			return "winget install Kitware.CMake", 90
		case "git":
			return "winget install Git.Git", 70
		case "kubectl":
			return "winget install Kubernetes.kubectl", 60
		default:
			return "winget install " + name, 50
		}
	}
}

