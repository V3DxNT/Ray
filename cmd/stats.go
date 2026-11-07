package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/V3DxNT/Ray/internal/detect"
	"github.com/V3DxNT/Ray/internal/fsutil"
	"github.com/V3DxNT/Ray/internal/ui"
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
	if err != nil || !info.IsDir() {
		color.Red("Target path must be an existing directory")
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
		ui.PrintBoxTop()
		fmt.Printf(color.CyanString("│")+" %-59s "+color.CyanString("│\n"), color.HiWhiteString("CODEBASE SUMMARY DASHBOARD"))
		ui.PrintBoxDivider()
		fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), color.YellowString("No matching files found in target directory."))
		ui.PrintBoxBottom()
		fmt.Println()
		return
	}

	ui.PrintBoxTop()
	fmt.Printf(color.CyanString("│")+" %-59s "+color.CyanString("│\n"), color.HiWhiteString("CODEBASE SUMMARY DASHBOARD"))
	ui.PrintBoxDivider2Col()
	fmt.Printf(color.CyanString("│")+" %-28s "+color.CyanString("│")+" %-28s "+color.CyanString("│\n"), "Metric Category", "Scanned Quantity")
	ui.PrintBoxDivider2Col()
	fmt.Printf(color.CyanString("│")+" %-28s "+color.CyanString("│")+" %-37s "+color.CyanString("│\n"), "Total File System Entries", color.HiYellowString("%d", totalEntries))
	fmt.Printf(color.CyanString("│")+" Source & Data Files           "+color.CyanString("│")+" %-37s "+color.CyanString("│\n"), color.HiWhiteString("%d", totalFiles))
	fmt.Printf(color.CyanString("│")+" Directory Branches            "+color.CyanString("│")+" %-37s "+color.CyanString("│\n"), color.HiCyanString("%d", totalDir))
	ui.PrintBoxBottom()
	fmt.Println()

	var statsList []ExtCount
	for k, val := range extCounts {
		statsList = append(statsList, ExtCount{k, val})
	}
	sort.Slice(statsList, func(i, j int) bool {
		return statsList[i].Count > statsList[j].Count
	})

	ui.PrintBoxTop()
	fmt.Printf(color.CyanString("│")+" %-59s "+color.CyanString("│\n"), color.HiWhiteString("TOP FILE EXTENSIONS BREAKDOWN"))
	ui.PrintBoxDivider()
	for i, file := range statsList {
		if i == 10 {
			break
		}

		name := file.Name
		count := file.Count
		pct := (float64(count) / float64(totalFiles)) * 100
		bar := ui.RenderBar(pct, 18)
		lineFormatted := fmt.Sprintf("  %-12s %s %5.1f%% (%d files)", name, bar, pct, count)
		fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), lineFormatted)
	}
	ui.PrintBoxBottom()
	fmt.Println()

	if listFiles && len(detectedSubProjects) > 0 {
		ui.PrintBoxTop()
		fmt.Printf(color.CyanString("│")+" %-59s "+color.CyanString("│\n"), color.HiWhiteString("DETECTED ENVIRONMENTS"))
		ui.PrintBoxDivider2Col()
		fmt.Printf(color.CyanString("│")+" %-28s "+color.CyanString("│")+" %-28s "+color.CyanString("│\n"), "Directory Path", "Framework / Stack")
		ui.PrintBoxDivider2Col()
		for path, framework := range detectedSubProjects {
			relPath := path
			if r, err := filepath.Rel(absPath, path); err == nil && r != "." {
				relPath = "./" + r
			} else if r == "." {
				relPath = "./"
			}
			relPath = ui.Truncate(relPath, 28)
			fmt.Printf(color.CyanString("│")+" %-28s "+color.CyanString("│")+" %-37s "+color.CyanString("│\n"), relPath, color.HiGreenString(framework))
		}
		ui.PrintBoxBottom()
		fmt.Println()
	}
}

func analze(currentDir string) {
	sem <- struct{}{}
	defer func() { <-sem }()

	dirEntries, err := os.ReadDir(currentDir)
	if err != nil {
		return
	}

	for _, dirEntry := range dirEntries {
		if fsutil.ShouldSkip(dirEntry.Name(), skippedList) {
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

			fw := detect.IdentifyFramework(fileName)
			if fw != "" {
				if fileName != "package.json" || currentFramework == "" {
					detectedSubProjects[currentDir] = fw
				}
			}
		}
		totalEntries++
		mu.Unlock()
	}
}
