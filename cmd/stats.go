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

/*
This Aggregates total lines of code, file counts,
heaviest directories, and primary tech stack breakdown.
*/

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show stats",
	Run:   stat,
}

var wg sync.WaitGroup
var mu sync.Mutex
var sem = make(chan struct{}, 100)

var extCounts map[string]int
var frameworks []string
var totalEntries int
var totalDir int
var totalFiles int
var listFiles bool

type ExtCount struct {
	Name  string
	Count int
}

var skippedList []string
var defaultSkippedList []string = []string{
	"node_modules",
	".next",
	"build",
	".git",
}

type SubProjects struct {
	Path      string
	Framework string
}

var detectedSubProjects map[string]string

func init() {
	statsCmd.Flags().StringSliceVarP(&skippedList, "skip", "s", []string{}, "skipped files")

	statsCmd.Flags().BoolVarP(&listFiles, "list", "l", false, "framework files")
	RootCmd.AddCommand(statsCmd)
}

func stat(cmd *cobra.Command, args []string) {

	extCounts = make(map[string]int)
	detectedSubProjects = make(map[string]string)

	if len(args) > 1 {
		color.Red("Too many arguments")
		return
	}
	var targetDir string
	if len(args) == 0 {
		pwd, _ := os.Getwd()
		targetDir = pwd
	} else {
		targetDir = args[0]
	}

	sem <- struct{}{}
	defer func() { <-sem }()
	wg.Add(1)
	go func() {
		defer wg.Done()
		analze(targetDir)
	}()
	wg.Wait()

	if totalFiles == 0 {
		fmt.Println("No files found")
		return
	}

	var statsList []ExtCount
	for k, val := range extCounts {
		statsList = append(statsList, ExtCount{k, val})
	}
	sort.Slice(statsList, func(i, j int) bool {
		return statsList[i].Count > statsList[j].Count
	})

	fmt.Println()
	color.Cyan("╭──────────────────────────────────────╮")
	fmt.Printf(color.CyanString("│")+" %-20s %-15d "+color.CyanString("│\n"), "Total", totalEntries)
	fmt.Printf(color.CyanString("│")+" %-20s %-15d "+color.CyanString("│\n"), "Files", totalFiles)
	fmt.Printf(color.CyanString("│")+" %-20s %-15d "+color.CyanString("│\n"), "Directories", totalDir)
	color.Cyan("╰──────────────────────────────────────╯")
	fmt.Println()

	if listFiles && len(detectedSubProjects) > 0 {
		fmt.Println("\n" + color.GreenString("🎯 Detected Environments:"))
		for path, framwork := range detectedSubProjects {
			fmt.Printf("  ├── %-30s [%s]\n", path, color.YellowString(framwork))
		}

	}

	for i, file := range statsList {

		if i == 10 {
			break
		}

		name := file.Name
		count := file.Count
		pct := (float64(count) / float64(totalFiles)) * 100
		filled := int((pct / 100) * 20)
		bar := strings.Repeat("█", filled) + strings.Repeat("░", 20-filled)
		fmt.Printf("%-10s %s %.1f%% : %d Files \n", name, bar, pct, count)
	}
}

func analze(currentDir string) {
	sem <- struct{}{}
	defer func() { <-sem }()

	dirEntries, err := os.ReadDir(currentDir)
	if err != nil {
		color.Red(err.Error())
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
			go func() {
				defer wg.Done()
				analze(filepath.Join(currentDir, dirEntry.Name()))
			}()
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

			case "next.config.js", "next.config.ts":
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
