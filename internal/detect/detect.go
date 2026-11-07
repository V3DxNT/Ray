package detect

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/V3DxNT/Ray/internal/fsutil"
)

type Component struct {
	RelativePath string `json:"path"`
	Stack        string `json:"stack"`
	Anchor       string `json:"anchor"`
	DepsOk       bool   `json:"dependencies_installed"`
	Notice       string `json:"dependency_notice,omitempty"`
}

func ScanSubprojects(root, current string, depth int, components *[]Component) {
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

	comp := DetectComponent(current, rel)
	if comp != nil {
		*components = append(*components, *comp)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if isSkippedDir(name) || (name == "app" && strings.HasSuffix(current, "android")) {
			continue
		}
		ScanSubprojects(root, filepath.Join(current, name), depth+1, components)
	}
}

func DetectComponent(dir, relPath string) *Component {
	hasGoMod := fsutil.FileExists(filepath.Join(dir, "go.mod"))
	hasPkgJson := fsutil.FileExists(filepath.Join(dir, "package.json"))
	hasNextConfig := fsutil.FileExists(filepath.Join(dir, "next.config.js")) || fsutil.FileExists(filepath.Join(dir, "next.config.ts")) || fsutil.FileExists(filepath.Join(dir, "next.config.mjs"))
	hasAppJson := fsutil.FileExists(filepath.Join(dir, "app.json")) || fsutil.FileExists(filepath.Join(dir, "metro.config.js"))
	hasDockerfile := fsutil.FileExists(filepath.Join(dir, "Dockerfile")) || fsutil.FileExists(filepath.Join(dir, "docker-compose.yml")) || fsutil.FileExists(filepath.Join(dir, "compose.yaml"))
	hasPy := fsutil.FileExists(filepath.Join(dir, "requirements.txt")) || fsutil.FileExists(filepath.Join(dir, "pyproject.toml"))
	hasCargo := fsutil.FileExists(filepath.Join(dir, "Cargo.toml"))
	hasJava := fsutil.FileExists(filepath.Join(dir, "pom.xml")) || fsutil.FileExists(filepath.Join(dir, "build.gradle"))
	hasCmake := fsutil.FileExists(filepath.Join(dir, "CMakeLists.txt"))
	hasPrisma := fsutil.FileExists(filepath.Join(dir, "schema.prisma")) || fsutil.FileExists(filepath.Join(dir, "prisma", "schema.prisma"))

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
		if !fsutil.FileExists(filepath.Join(dir, "go.sum")) {
			depsOk = false
			notice = "go.sum missing (run 'go mod tidy')"
		}
	}

	if hasAppJson {
		stackParts = append(stackParts, "React Native (Expo Mobile)")
		anchorParts = append(anchorParts, "app.json")
		if !fsutil.DirExists(filepath.Join(dir, "node_modules")) {
			depsOk = false
			notice = "node_modules missing (run 'npm install')"
		}
	} else if hasNextConfig {
		stackParts = append(stackParts, "Next.js Web Application")
		anchorParts = append(anchorParts, "next.config")
		if !fsutil.DirExists(filepath.Join(dir, "node_modules")) {
			depsOk = false
			notice = "node_modules missing (run 'npm install')"
		}
	} else if hasPkgJson {
		stackParts = append(stackParts, "Node.js / JavaScript")
		anchorParts = append(anchorParts, "package.json")
		if !fsutil.DirExists(filepath.Join(dir, "node_modules")) {
			depsOk = false
			notice = "node_modules missing (run 'npm install')"
		}
	}

	if hasPy {
		stackParts = append(stackParts, "Python Application")
		anchorParts = append(anchorParts, "requirements.txt")
	}

	if hasCargo {
		stackParts = append(stackParts, "Rust Application")
		anchorParts = append(anchorParts, "Cargo.toml")
	}

	if hasJava {
		stackParts = append(stackParts, "Java / JVM Application")
		anchorParts = append(anchorParts, "pom.xml")
	}

	if hasCmake {
		stackParts = append(stackParts, "C++ / CMake Project")
		anchorParts = append(anchorParts, "CMakeLists.txt")
	}

	if hasPrisma {
		stackParts = append(stackParts, "Prisma ORM")
		anchorParts = append(anchorParts, "schema.prisma")
	}

	if hasDockerfile && len(stackParts) == 0 {
		stackParts = append(stackParts, "Docker Containerization")
		anchorParts = append(anchorParts, "Dockerfile")
	} else if hasDockerfile {
		stackParts = append(stackParts, "Docker")
	}

	return &Component{
		RelativePath: relPath,
		Stack:        strings.Join(stackParts, " + "),
		Anchor:       strings.Join(anchorParts, ", "),
		DepsOk:       depsOk,
		Notice:       notice,
	}
}

func DetectStacks(root string) []string {
	stackSet := make(map[string]bool)

	var components []Component
	ScanSubprojects(root, root, 0, &components)

	for _, c := range components {
		st := strings.ToLower(c.Stack)
		if strings.Contains(st, "go lang") {
			stackSet["go"] = true
		}
		if strings.Contains(st, "node") || strings.Contains(st, "react") || strings.Contains(st, "next") || strings.Contains(st, "vite") {
			stackSet["node"] = true
		}
		if strings.Contains(st, "python") {
			stackSet["python"] = true
		}
		if strings.Contains(st, "rust") {
			stackSet["rust"] = true
		}
		if strings.Contains(st, "java") {
			stackSet["java"] = true
		}
		if strings.Contains(st, "c++") || strings.Contains(st, "cmake") {
			stackSet["cpp"] = true
		}
	}

	if fsutil.FileExists(filepath.Join(root, "go.mod")) {
		stackSet["go"] = true
	}
	if fsutil.FileExists(filepath.Join(root, "package.json")) {
		stackSet["node"] = true
	}
	if fsutil.FileExists(filepath.Join(root, "requirements.txt")) || fsutil.FileExists(filepath.Join(root, "pyproject.toml")) {
		stackSet["python"] = true
	}
	if fsutil.FileExists(filepath.Join(root, "Cargo.toml")) {
		stackSet["rust"] = true
	}
	if fsutil.FileExists(filepath.Join(root, "pom.xml")) || fsutil.FileExists(filepath.Join(root, "build.gradle")) {
		stackSet["java"] = true
	}
	if fsutil.FileExists(filepath.Join(root, "CMakeLists.txt")) {
		stackSet["cpp"] = true
	}

	var sorted []string
	for k := range stackSet {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)
	return sorted
}

func IdentifyFramework(filename string) string {
	switch filename {
	case "go.mod":
		return "Go Lang"
	case "next.config.js", "next.config.ts", "next.config.mjs":
		return "Next.js"
	case "app.json", "metro.config.js":
		return "React Native"
	case "requirements.txt", "pyproject.toml", "Pipfile":
		return "Python"
	case "CMakeLists.txt":
		return "C++"
	case "pom.xml", "build.gradle":
		return "Java"
	case "schema.prisma":
		return "Prisma DB"
	case "package.json":
		return "Node.js"
	default:
		return ""
	}
}

func isSkippedDir(name string) bool {
	skips := map[string]bool{
		".git":         true,
		"node_modules": true,
		".next":        true,
		"build":        true,
		"dist":         true,
		"data":         true,
		".idea":        true,
		".vscode":      true,
		"Pods":         true,
		".gradle":      true,
		"target":       true,
		".cache":       true,
		".turbo":       true,
		"vendor":       true,
		"__pycache__":  true,
		".venv":        true,
		"venv":         true,
	}
	return skips[name]
}

