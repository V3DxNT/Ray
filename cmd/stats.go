package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats [path]",
	Short: "Generate a codebase dashboard (tech stack, lines of code, sizes)",
	Run:   stat,
}

var wg sync.WaitGroup
var mu sync.Mutex
var sem = make(chan struct{}, 100)

var extCounts map[string]int
var totalEntries int
var totalDir int
var totalFiles int
var listFiles bool

type ExtCount struct {
	Name  string
	Count int
}

var skippedList []string
var defaultSkippedList = []string{
	"node_modules",
	".next",
	"build",
	"dist",
	".git",
}

var detectedSubProjects map[string]string

func init() {
	statsCmd.Flags().StringSliceVarP(&skippedList, "skip", "s", []string{}, "Folders or files to skip")
	statsCmd.Flags().BoolVarP(&listFiles, "list", "l", false, "List detected subproject environments")
	RootCmd.AddCommand(statsCmd)
}

func stat(cmd *cobra.Command, args []string) {
	mu.Lock()
	extCounts = make(map[string]int)
	detectedSubProjects = make(map[string]string)
	totalEntries = 0
	totalDir = 0
	totalFiles = 0
	mu.Unlock()

	var targetDir string
	if len(args) == 0 {
		pwd, err := os.Getwd()
		if err != nil {
			color.Red("Error getting current directory: %v", err)
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

	sem <- struct{}{}
	wg.Add(1)
	go func() {
		defer func() { <-sem }()
		defer wg.Done()
		analze(absPath)
	}()
	wg.Wait()

	fmt.Println()
	fmt.Printf("🔍 Target: %s\n\n", color.YellowString(absPath))

	if totalFiles == 0 {
		color.Cyan("╭─────────────────────────────────────────────────────────────╮")
		color.Cyan("│                 CODEBASE SUMMARY DASHBOARD                  │")
		color.Cyan("├─────────────────────────────────────────────────────────────┤")
		fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), color.YellowString("No matching files found in target directory."))
		color.Cyan("╰─────────────────────────────────────────────────────────────╯\n")
		return
	}

	color.Cyan("╭─────────────────────────────────────────────────────────────╮")
	color.Cyan("│                 CODEBASE SUMMARY DASHBOARD                  │")
	color.Cyan("├──────────────────────────────┬──────────────────────────────┤")
	fmt.Printf(color.CyanString("│")+" %-28s "+color.CyanString("│")+" %-28s "+color.CyanString("│\n"), "Metric Category", "Scanned Quantity")
	color.Cyan("├──────────────────────────────┼──────────────────────────────┤")
	fmt.Printf(color.CyanString("│")+" %-28s "+color.CyanString("│")+" %-37s "+color.CyanString("│\n"), "Total File System Entries", color.HiYellowString("%d", totalEntries))
	fmt.Printf(color.CyanString("│")+" Source & Data Files           "+color.CyanString("│")+" %-37s "+color.CyanString("│\n"), color.HiWhiteString("%d", totalFiles))
	fmt.Printf(color.CyanString("│")+" Directory Branches            "+color.CyanString("│")+" %-37s "+color.CyanString("│\n"), color.HiCyanString("%d", totalDir))
	color.Cyan("╰──────────────────────────────┴──────────────────────────────╯\n")

	var statsList []ExtCount
	for k, val := range extCounts {
		statsList = append(statsList, ExtCount{k, val})
	}
	sort.Slice(statsList, func(i, j int) bool {
		return statsList[i].Count > statsList[j].Count
	})

	color.Cyan("╭─────────────────────────────────────────────────────────────╮")
	color.Cyan("│               TOP FILE EXTENSIONS BREAKDOWN                 │")
	color.Cyan("├─────────────────────────────────────────────────────────────┤")
	for i, file := range statsList {
		if i == 10 {
			break
		}

		name := file.Name
		count := file.Count
		pct := (float64(count) / float64(totalFiles)) * 100
		bar := renderStatsBar(pct)
		lineFormatted := fmt.Sprintf("  %-12s %s %5.1f%% (%d files)", name, bar, pct, count)
		fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), lineFormatted)
	}
	color.Cyan("╰─────────────────────────────────────────────────────────────╯\n")

	if listFiles && len(detectedSubProjects) > 0 {
		color.Cyan("╭─────────────────────────────────────────────────────────────╮")
		color.Cyan("│                    DETECTED ENVIRONMENTS                    │")
		color.Cyan("├──────────────────────────────┬──────────────────────────────┤")
		fmt.Printf(color.CyanString("│")+" %-28s "+color.CyanString("│")+" %-28s "+color.CyanString("│\n"), "Directory Path", "Framework / Stack")
		color.Cyan("├──────────────────────────────┼──────────────────────────────┤")
		for path, framework := range detectedSubProjects {
			relPath := path
			if r, err := filepath.Rel(absPath, path); err == nil && r != "." {
				relPath = "./" + r
			} else if r == "." {
				relPath = "./"
			}
			if len(relPath) > 28 {
				relPath = relPath[:25] + "..."
			}
			fmt.Printf(color.CyanString("│")+" %-28s "+color.CyanString("│")+" %-37s "+color.CyanString("│\n"), relPath, color.HiGreenString(framework))
		}
		color.Cyan("╰──────────────────────────────┴──────────────────────────────╯\n")
	}
}

func renderStatsBar(pct float64) string {
	filled := int((pct / 100) * 18)
	if filled > 18 {
		filled = 18
	}
	if filled < 1 && pct > 0 {
		filled = 1
	}
	return color.GreenString(strings.Repeat("█", filled)) + color.HiBlackString(strings.Repeat("░", 18-filled))
}

func analze(currentDir string) {
	sem <- struct{}{}
	defer func() { <-sem }()

	dirEntries, err := os.ReadDir(currentDir)
	if err != nil {
		return
	}

	for _, dirEntry := range dirEntries {
		if shoudldSkip(dirEntry.Name()) {
			continue
		}

		mu.Lock()
		if dirEntry.IsDir() {
			totalDir++
			wg.Add(1)
			go func(p string) {
				defer wg.Done()
				analze(p)
			}(filepath.Join(currentDir, dirEntry.Name()))
		} else {
			extension := filepath.Ext(dirEntry.Name())
			if extension == "" {
				extension = "Binaries"
			}
			extCounts[extension]++
			totalFiles++
			fileName := dirEntry.Name()
			currentFramework := detectedSubProjects[currentDir]

			switch fileName {
			case "go.mod":
				detectedSubProjects[currentDir] = "Go Lang"
			case "next.config.js", "next.config.ts", "next.config.mjs":
				detectedSubProjects[currentDir] = "Next.js"
			case "app.json", "metro.config.js":
				detectedSubProjects[currentDir] = "React Native"
			case "requirements.txt", "pyproject.toml", "Pipfile":
				detectedSubProjects[currentDir] = "Python"
			case "CMakeLists.txt":
				detectedSubProjects[currentDir] = "C++"
			case "pom.xml", "build.gradle":
				detectedSubProjects[currentDir] = "Java"
			case "schema.prisma":
				detectedSubProjects[currentDir] = "Prisma DB"
			case "package.json":
				if currentFramework == "" {
					detectedSubProjects[currentDir] = "Node.js"
				}
			}
		}
		totalEntries++
		mu.Unlock()
	}
}

func shoudldSkip(fileName string) bool {
	for _, skipped := range skippedList {
		if fileName == skipped {
			return true
		}
	}

	for _, skipped := range defaultSkippedList {
		if fileName == skipped {
			return true
		}
	}
	return false
}
