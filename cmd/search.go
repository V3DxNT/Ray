package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search [filename...]",
	Short: "Hunt for exact and partial file matches across directories",
	Run:   search,
}

var targets []string
var searchOnlyFirst bool
var searchShowAll bool
var searchSkipDirs []string

var defaultSearchSkips = []string{
	".git",
	"node_modules",
}

func init() {
	searchCmd.Flags().BoolVarP(&searchOnlyFirst, "first", "f", false, "Stop search on first exact occurrence")
	searchCmd.Flags().BoolVarP(&searchShowAll, "all", "a", false, "Show all similar results without truncation")
	searchCmd.Flags().StringSliceVarP(&searchSkipDirs, "skip", "s", []string{}, "Folders or files to skip")
	RootCmd.AddCommand(searchCmd)
}

func search(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		color.Red("Search requires at least one filename target")
		return
	}

	currentDir, err := os.Getwd()
	if err != nil {
		color.Red("Error getting current directory: %v", err)
		return
	}

	targets = args
	var exactMatches []string
	var similarMatches []string
	seenExact := make(map[string]bool)
	seenSimilar := make(map[string]bool)

	fmt.Println()
	targetQuery := strings.Join(targets, ", ")
	fmt.Printf("🔍 Searching for: [%s] in %s\n\n", color.YellowString(targetQuery), color.CyanString(currentDir))

	hunt(currentDir, &exactMatches, &similarMatches, seenExact, seenSimilar)

	if len(exactMatches) == 0 && len(similarMatches) == 0 {
		renderSearchBox("SEARCH RESULTS (0 FOUND)", []string{}, color.RedString("❌ No exact or similar matches found for: %s", targetQuery))
		return
	}

	if len(exactMatches) > 0 {
		title := fmt.Sprintf("EXACT MATCHES (%d FOUND)", len(exactMatches))
		renderSearchBox(title, exactMatches, "")
	}

	if len(similarMatches) > 0 {
		title := fmt.Sprintf("SIMILAR / PARTIAL MATCHES (%d FOUND)", len(similarMatches))
		displayList := similarMatches
		extraNotice := ""

		if !searchShowAll && len(similarMatches) > 20 {
			extraCount := len(similarMatches) - 20
			displayList = similarMatches[:20]
			extraNotice = fmt.Sprintf("... and %d more similar matches (use -a to view all)", extraCount)
		}

		renderSearchBox(title, displayList, extraNotice)
	}
}

func renderSearchBox(title string, items []string, notice string) {
	padTotal := 61 - len(title)
	padLeft := padTotal / 2
	padRight := padTotal - padLeft
	if padLeft < 1 {
		padLeft = 1
	}
	if padRight < 1 {
		padRight = 1
	}

	color.Cyan("╭─────────────────────────────────────────────────────────────╮")
	fmt.Printf(color.CyanString("│")+"%s%s%s"+color.CyanString("│\n"), strings.Repeat(" ", padLeft), color.HiWhiteString(title), strings.Repeat(" ", padRight))
	color.Cyan("├─────────────────────────────────────────────────────────────┤")

	if notice != "" && len(items) == 0 {
		fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), notice)
		color.Cyan("╰─────────────────────────────────────────────────────────────╯\n")
		return
	}

	for i, m := range items {
		itemStr := m
		if len(itemStr) > 52 {
			itemStr = "..." + itemStr[len(itemStr)-49:]
		}
		line := fmt.Sprintf("  %2d. %s", i+1, itemStr)
		fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), color.HiGreenString(line))
	}

	if notice != "" {
		fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), color.HiBlackString("  "+notice))
	}

	color.Cyan("╰─────────────────────────────────────────────────────────────╯\n")
}

func hunt(currentDir string, exactMatches, similarMatches *[]string, seenExact, seenSimilar map[string]bool) {
	if len(targets) == 0 {
		return
	}

	dir, err := os.ReadDir(currentDir)
	if err != nil {
		return
	}

	for _, file := range dir {
		name := file.Name()
		if isSearchSkipped(name) {
			continue
		}

		fullPath := filepath.Join(currentDir, name)

		isExact, isSimilar := checkTargetMatch(name)
		if isExact {
			if !seenExact[fullPath] {
				seenExact[fullPath] = true
				*exactMatches = append(*exactMatches, fullPath)
			}
			if searchOnlyFirst {
				return
			}
		} else if isSimilar {
			if !seenSimilar[fullPath] && !seenExact[fullPath] {
				seenSimilar[fullPath] = true
				*similarMatches = append(*similarMatches, fullPath)
			}
		}

		if file.IsDir() {
			hunt(fullPath, exactMatches, similarMatches, seenExact, seenSimilar)
		}
	}
}

func checkTargetMatch(filename string) (bool, bool) {
	lowerFile := strings.ToLower(filename)
	for _, arg := range targets {
		lowerArg := strings.ToLower(arg)
		if strings.EqualFold(arg, filename) {
			return true, false
		}
		if strings.Contains(lowerFile, lowerArg) {
			return false, true
		}
	}
	return false, false
}

func isSearchSkipped(name string) bool {
	for _, s := range defaultSearchSkips {
		if s == name {
			return true
		}
	}
	for _, s := range searchSkipDirs {
		if s == name {
			return true
		}
	}
	return false
}
