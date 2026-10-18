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
	Short: "Exhaustively hunt for a specific file across all directories",
	Run:   search,
}

var targets []string
var onlyfirst bool

func init() {
	searchCmd.Flags().BoolVarP(&onlyfirst, "first", "f", false, "Stop search on first occurrence")
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
	var matches []string

	fmt.Println()
	targetQuery := strings.Join(targets, ", ")
	fmt.Printf("🔍 Searching for: [%s] in %s\n\n", color.YellowString(targetQuery), color.CyanString(currentDir))

	hunt(currentDir, &matches)

	title := fmt.Sprintf("SEARCH RESULTS (%d FOUND)", len(matches))
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

	if len(matches) == 0 {
		fmt.Printf(color.CyanString("│")+" %-69s "+color.CyanString("│\n"), color.RedString("❌ No matching files found for: %s", targetQuery))
		color.Cyan("╰─────────────────────────────────────────────────────────────╯\n")
		return
	}

	for i, m := range matches {
		line := fmt.Sprintf("  %d. %s", i+1, m)
		if len(line) > 59 {
			line = line[:56] + "..."
		}
		fmt.Printf(color.CyanString("│")+" %-59s "+color.CyanString("│\n"), color.HiGreenString(line))
	}

	color.Cyan("╰─────────────────────────────────────────────────────────────╯\n")
}

func hunt(currentDir string, matches *[]string) {
	if len(targets) == 0 {
		return
	}

	dir, err := os.ReadDir(currentDir)
	if err != nil {
		return
	}

	for _, file := range dir {
		name := file.Name()
		if name == ".git" {
			continue
		}

		fullPath := filepath.Join(currentDir, name)

		if isTarget(name) {
			*matches = append(*matches, fullPath)
		}

		if file.IsDir() {
			hunt(fullPath, matches)
		}
	}
}

func isTarget(filename string) bool {
	for i, arg := range targets {
		if strings.EqualFold(arg, filename) {
			if onlyfirst {
				targets = append(targets[:i], targets[i+1:]...)
			}
			return true
		}
	}
	return false
}
