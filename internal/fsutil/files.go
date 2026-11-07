package fsutil

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

var DefaultSkips = []string{
	".git",
	"node_modules",
	"build",
	"dist",
	".next",
	".nuxt",
	".turbo",
	".cache",
	"vendor",
	"__pycache__",
	".venv",
	"venv",
	"target",
	"bin",
	"obj",
	".gradle",
	"Pods",
	".idea",
	".vscode",
	".terraform",
}

var LockFileNames = map[string]bool{
	"go.sum":            true,
	"package-lock.json": true,
	"yarn.lock":         true,
	"pnpm-lock.yaml":    true,
	"Cargo.lock":        true,
	"composer.lock":     true,
	"poetry.lock":       true,
	"flake.lock":        true,
}

var binaryExtensions = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".ico": true,
	".svg": true, ".pdf": true, ".zip": true, ".tar": true, ".gz": true,
	".exe": true, ".bin": true, ".iso": true, ".dll": true, ".so": true,
	".dylib": true, ".woff": true, ".woff2": true, ".ttf": true, ".eot": true,
	".mp4": true, ".mp3": true, ".wav": true, ".db": true, ".sqlite": true,
	".wasm": true,
}

var codeExtensions = map[string]bool{
	".js": true, ".jsx": true, ".ts": true, ".tsx": true, ".mjs": true, ".cjs": true,
	".go": true,
	".py": true,
	".rs": true,
	".java": true, ".kt": true,
	".php": true,
	".rb":  true,
}

func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func ShouldSkip(name string, userSkips []string) bool {
	if LockFileNames[name] {
		return true
	}
	for _, s := range userSkips {
		if s == name {
			return true
		}
	}
	for _, s := range DefaultSkips {
		if s == name {
			return true
		}
	}
	return false
}

func IsCandidateTextFile(name, path string) bool {
	if LockFileNames[name] {
		return false
	}

	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".min.js") || strings.HasSuffix(lower, ".min.css") || strings.HasSuffix(lower, ".map") {
		return false
	}

	ext := strings.ToLower(filepath.Ext(name))
	if binaryExtensions[ext] {
		return false
	}

	info, err := os.Stat(path)
	if err != nil || info.Size() > 2*1024*1024 {
		return false
	}

	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return false
	}

	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return false
		}
	}
	return true
}

func IsCandidateCodeFile(name, path string) bool {
	if LockFileNames[name] {
		return false
	}

	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".min.js") || strings.HasSuffix(lower, ".min.css") || strings.HasSuffix(lower, ".map") {
		return false
	}

	ext := strings.ToLower(filepath.Ext(name))
	if !codeExtensions[ext] {
		return false
	}

	info, err := os.Stat(path)
	if err != nil || info.Size() > 2*1024*1024 {
		return false
	}

	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return false
	}

	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return false
		}
	}
	return true
}

func CalculateDirSize(dirPath string) int64 {
	var total int64
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return 0
	}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if entry.IsDir() {
			total += CalculateDirSize(filepath.Join(dirPath, entry.Name()))
		} else {
			total += info.Size()
		}
	}
	return total
}

