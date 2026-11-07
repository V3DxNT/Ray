package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/V3DxNT/Ray/internal/fsutil"
	"github.com/V3DxNT/Ray/internal/ui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type CleanCategory string

const (
	CategoryJunkDir   CleanCategory = "Junk Dir"
	CategoryJunkFile  CleanCategory = "Junk File"
	CategoryHeavyFile CleanCategory = "Heavy File"
	CategoryDocker    CleanCategory = "Docker"
)

type CleanTarget struct {
	Path        string
	Name        string
	Size        int64
	Category    CleanCategory
	Description string
	IsDir       bool
}

var cleanThreshold int64
var cleanSkip []string
var cleanListOnly bool
var cleanForce bool
var cleanIncludeDocker bool

var cleanCmd = &cobra.Command{
	Use:   "clean [path]",
	Short: "Clean junk directories, heavy files, and system cache",
	Run:   runClean,
}

func init() {
	cleanCmd.Flags().Int64VarP(&cleanThreshold, "threshold", "t", 10, "Threshold in MB for heavy files and min directory size")
	cleanCmd.Flags().StringSliceVarP(&cleanSkip, "skip", "s", []string{}, "Folders or files to skip")
	cleanCmd.Flags().BoolVarP(&cleanListOnly, "list", "l", false, "List targets only without interactive deletion")
	cleanCmd.Flags().BoolVarP(&cleanForce, "force", "f", false, "Delete all targets without confirmation")
	cleanCmd.Flags().BoolVarP(&cleanIncludeDocker, "docker", "d", false, "Include Docker dangling images and build cache")
	RootCmd.AddCommand(cleanCmd)
}

func runClean(cmd *cobra.Command, args []string) {
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

	dirThresholdBytes := int64(0)
	if cmd.Flags().Changed("threshold") {
		dirThresholdBytes = cleanThreshold * 1024 * 1024
	}
	fileThresholdBytes := cleanThreshold * 1024 * 1024

	var targets []CleanTarget
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 100)

	printCleanHeader(absPath)

	sem <- struct{}{}
	wg.Add(1)
	go func() {
		defer func() { <-sem }()
		defer wg.Done()
		scanCleanTargets(absPath, dirThresholdBytes, fileThresholdBytes, &targets, &mu, &wg, sem, cleanSkip)
	}()
	wg.Wait()

	if cleanIncludeDocker {
		dockerTarget := checkDockerReclaimable()
		if dockerTarget != nil {
			targets = append(targets, *dockerTarget)
		}
	}

	if len(targets) == 0 {
		color.Green("✔ Codebase is pristine! No junk directories or heavy files found exceeding criteria.\n")
		return
	}

	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Size > targets[j].Size
	})

	var totalReclaimable int64
	for _, t := range targets {
		totalReclaimable += t.Size
	}

	renderCleanDashboard(targets, totalReclaimable)

	if cleanListOnly {
		color.Cyan("\n[List Mode Active] No deletion performed.\n")
		return
	}

	executeInteractiveClean(targets)
}

func printCleanHeader(path string) {
	fmt.Println()
	fmt.Printf("🔍 Scanning: %s\n", color.YellowString(path))
	fmt.Printf("⚙  Threshold: %s (Heavy files: %d MB | Junk dirs: dynamic)\n\n", color.GreenString("%d MB", cleanThreshold), cleanThreshold)
}

func renderCleanDashboard(targets []CleanTarget, totalSize int64) {
	ui.PrintBoxTop()
	fmt.Printf(color.CyanString("│")+" %-59s "+color.CyanString("│\n"), color.HiWhiteString("DETECTED CLEANUP CANDIDATES"))
	ui.PrintBoxDivider()
	summaryLine := fmt.Sprintf("Total Candidates: %d  │  Total Reclaimable: %s", len(targets), ui.FormatBytes(totalSize))
	fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), color.HiMagentaString(summaryLine))
	ui.PrintBoxDivider()

	for i, t := range targets {
		catTag := color.RedString("[DIR] ")
		if t.Category == CategoryJunkFile {
			catTag = color.YellowString("[JUNK]")
		} else if t.Category == CategoryHeavyFile {
			catTag = color.MagentaString("[FILE]")
		} else if t.Category == CategoryDocker {
			catTag = color.CyanString("[DOCK]")
		}

		sizeStr := color.HiYellowString("%9s", ui.FormatBytes(t.Size))
		bar := renderCleanBar(t.Size, totalSize)

		displayPath := t.Path
		if len(displayPath) > 34 {
			displayPath = "..." + displayPath[len(displayPath)-31:]
		}

		fmt.Printf(color.CyanString("│")+"  %2d.  %s %s %s  %-34s "+color.CyanString("│\n"),
			i+1, catTag, sizeStr, bar, color.HiWhiteString(displayPath))

		descStr := fmt.Sprintf("└── %s", t.Description)
		if len(descStr) > 59 {
			descStr = descStr[:56] + "..."
		}
		fmt.Printf(color.CyanString("│")+"     %-65s "+color.CyanString("│\n"), color.HiBlackString(descStr))
	}

	ui.PrintBoxBottom()
}

func renderCleanBar(size, total int64) string {
	if total <= 0 {
		return color.HiBlackString("░░░░░░░░")
	}
	pct := float64(size) / float64(total)
	filled := int(pct * 10)
	if filled < 1 && size > 0 {
		filled = 1
	}
	if filled > 10 {
		filled = 10
	}
	return color.GreenString(strings.Repeat("█", filled)) + color.HiBlackString(strings.Repeat("░", 10-filled))
}

func executeInteractiveClean(targets []CleanTarget) {
	reader := bufio.NewReader(os.Stdin)
	var deleted []CleanTarget
	var totalFreed int64

	fmt.Println()
	color.Yellow("Commands: [y]es delete, [n]o cancel, [a]ll delete all remaining, [s]kip")
	fmt.Println()

	for i, item := range targets {
		if cleanForce {
			freed := performDeletion(item)
			if freed > 0 {
				deleted = append(deleted, item)
				totalFreed += freed
			}
			continue
		}

		fmt.Printf("[%d/%d] Delete %s %s (%s)? [y/n/a/s]: ",
			i+1, len(targets), item.Category, color.YellowString(item.Path), color.HiCyanString(ui.FormatBytes(item.Size)))

		input, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "y" || input == "yes" {
			freed := performDeletion(item)
			if freed > 0 {
				deleted = append(deleted, item)
				totalFreed += freed
			}
		} else if input == "a" || input == "all" {
			for j := i; j < len(targets); j++ {
				rem := targets[j]
				freed := performDeletion(rem)
				if freed > 0 {
					deleted = append(deleted, rem)
					totalFreed += freed
				}
			}
			break
		} else if input == "s" || input == "skip" {
			color.HiBlack("  ↷ Skipped %s", item.Path)
			continue
		} else if input == "n" || input == "no" {
			color.Yellow("\nSweeper interrupted by user.")
			break
		} else {
			color.HiBlack("  ↷ Skipped %s", item.Path)
		}
	}

	fmt.Println()
	if len(deleted) == 0 {
		color.Cyan("No targets were deleted.")
		return
	}

	ui.PrintBoxTop()
	fmt.Printf(color.GreenString("│")+" %-59s "+color.GreenString("│\n"), "CLEANUP COMPLETE")
	ui.PrintBoxBottom()
	for _, item := range deleted {
		fmt.Printf("  ✔ Deleted: %s (%s)\n", item.Path, color.HiMagentaString(ui.FormatBytes(item.Size)))
	}
	fmt.Printf("\n🎉 Reclaimed Space: %s\n\n", color.HiGreenString(ui.FormatBytes(totalFreed)))
}

func performDeletion(item CleanTarget) int64 {
	if item.Category == CategoryDocker {
		return cleanDockerResources()
	}

	err := os.RemoveAll(item.Path)
	if err != nil {
		color.Red("  ✖ Failed to remove %s: %v", item.Path, err)
		return 0
	}
	color.Green("  ✔ Removed %s", item.Path)
	return item.Size
}

func scanCleanTargets(currentDir string, dirThreshold, fileThreshold int64, targets *[]CleanTarget, mu *sync.Mutex, wg *sync.WaitGroup, sem chan struct{}, skips []string) {
	entries, err := os.ReadDir(currentDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		if name == ".git" {
			continue
		}
		for _, s := range skips {
			if s == name {
				continue
			}
		}

		fullPath := filepath.Join(currentDir, name)

		if entry.IsDir() {
			desc, isJunk := getJunkDirMetadata(name)
			if isJunk {
				dirSize := fsutil.CalculateDirSize(fullPath)
				if dirSize >= dirThreshold {
					mu.Lock()
					*targets = append(*targets, CleanTarget{
						Path:        fullPath,
						Name:        name,
						Size:        dirSize,
						Category:    CategoryJunkDir,
						Description: desc,
						IsDir:       true,
					})
					mu.Unlock()
				}
				continue
			}

			select {
			case sem <- struct{}{}:
				wg.Add(1)
				go func(p string) {
					defer func() { <-sem }()
					defer wg.Done()
					scanCleanTargets(p, dirThreshold, fileThreshold, targets, mu, wg, sem, skips)
				}(fullPath)
			default:
				scanCleanTargets(fullPath, dirThreshold, fileThreshold, targets, mu, wg, sem, skips)
			}
		} else {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			fSize := info.Size()

			if desc, isJunk := getJunkFileMetadata(name); isJunk {
				mu.Lock()
				*targets = append(*targets, CleanTarget{
					Path:        fullPath,
					Name:        name,
					Size:        fSize,
					Category:    CategoryJunkFile,
					Description: desc,
					IsDir:       false,
				})
				mu.Unlock()
			} else if fSize >= fileThreshold {
				mu.Lock()
				*targets = append(*targets, CleanTarget{
					Path:        fullPath,
					Name:        name,
					Size:        fSize,
					Category:    CategoryHeavyFile,
					Description: "Heavy repository file",
					IsDir:       false,
				})
				mu.Unlock()
			}
		}
	}
}

func getJunkDirMetadata(name string) (string, bool) {
	junkMap := map[string]string{
		"node_modules":  "Node.js package dependencies",
		".next":         "Next.js build cache and pages",
		".nuxt":         "Nuxt.js build artifacts",
		".turbo":        "Turborepo remote/local cache",
		".cache":        "General build and tool cache",
		"build":         "Compiled build artifacts",
		"dist":          "Distribution release output",
		"target":        "Rust / Java target build folder",
		"bin":           "Binary executable artifacts",
		"obj":           ".NET build intermediate objects",
		"vendor":        "Vendored module dependencies",
		"__pycache__":   "Python bytecode cache",
		".pytest_cache": "PyTest run cache",
		".venv":         "Python virtual environment",
		"venv":          "Python virtual environment",
		"env":           "Virtual environment",
		".gradle":       "Gradle build daemon & wrapper cache",
		"Pods":          "Cocoapods iOS dependencies",
		".dart_tool":    "Flutter / Dart build configuration",
		".parcel-cache": "Parcel bundler cache",
		".svelte-kit":   "SvelteKit development output",
		".terraform":    "Terraform provider plugins & modules",
		".serverless":   "Serverless framework packaging",
		".yarn":         "Yarn local cache & releases",
		"coverage":      "Test coverage reports",
		".nyc_output":   "NYC coverage instrumentation",
	}

	desc, ok := junkMap[name]
	return desc, ok
}

func getJunkFileMetadata(name string) (string, bool) {
	lower := strings.ToLower(name)

	if lower == ".ds_store" || lower == "thumbs.db" {
		return "OS directory metadata artifact", true
	}
	if strings.HasSuffix(lower, ".tsbuildinfo") {
		return "TypeScript incremental build state", true
	}
	if strings.HasPrefix(lower, "npm-debug.log") || strings.HasPrefix(lower, "yarn-error.log") || strings.HasPrefix(lower, "pnpm-debug.log") {
		return "Package manager error log dump", true
	}
	if strings.HasSuffix(lower, ".hprof") || strings.HasPrefix(lower, "core.") || strings.HasSuffix(lower, ".dump") {
		return "Process crash or heap dump file", true
	}
	if strings.HasSuffix(lower, ".log") && !strings.Contains(lower, "changelog") {
		return "Application runtime log file", true
	}
	if strings.HasSuffix(lower, ".tmp") || strings.HasSuffix(lower, ".swp") || strings.HasSuffix(lower, "~") {
		return "Temporary swap or backup file", true
	}

	return "", false
}

func checkDockerReclaimable() *CleanTarget {
	_, err := exec.LookPath("docker")
	if err != nil {
		return nil
	}

	out, err := exec.Command("docker", "system", "df", "--format", "{{.Type}}|{{.Reclaimable}}").Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var totalReclaimableBytes int64

	for _, line := range lines {
		parts := strings.Split(line, "|")
		if len(parts) >= 2 {
			recStr := parts[1]
			if idx := strings.Index(recStr, "("); idx != -1 {
				recStr = strings.TrimSpace(recStr[:idx])
			}
			b := parseDockerByteString(recStr)
			totalReclaimableBytes += b
		}
	}

	if totalReclaimableBytes > 0 {
		return &CleanTarget{
			Path:        "docker://system/cache",
			Name:        "Docker System Cache",
			Size:        totalReclaimableBytes,
			Category:    CategoryDocker,
			Description: "Dangling images, stopped containers & build cache",
			IsDir:       false,
		}
	}

	return nil
}

func parseDockerByteString(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "0B" || s == "0" {
		return 0
	}

	var val float64
	var unit string
	_, err := fmt.Sscanf(s, "%f%s", &val, &unit)
	if err != nil {
		return 0
	}

	unit = strings.ToUpper(unit)
	switch {
	case strings.HasPrefix(unit, "GB"):
		return int64(val * 1024 * 1024 * 1024)
	case strings.HasPrefix(unit, "MB"):
		return int64(val * 1024 * 1024)
	case strings.HasPrefix(unit, "KB"):
		return int64(val * 1024)
	case strings.HasPrefix(unit, "B"):
		return int64(val)
	default:
		return 0
	}
}

func cleanDockerResources() int64 {
	color.Cyan("  🐳 Pruning Docker dangling images and build cache...")
	cmdImg := exec.Command("docker", "image", "prune", "-f")
	_ = cmdImg.Run()

	cmdBuild := exec.Command("docker", "builder", "prune", "-f")
	_ = cmdBuild.Run()

	color.Green("  ✔ Docker cleanup complete.")
	return 1
}
