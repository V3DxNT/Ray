package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var castCmd = &cobra.Command{
	Use:   "cast [path]",
	Short: "Render a contextual, tech-aware ASCII directory tree",
	Run:   runCast,
}

var defaultSkips = []string{"node_modules", ".git", "data", "build", ".next"}
var skipDirs []string
var showAll bool = false
var outputFile string

func init() {
	RootCmd.AddCommand(castCmd)
	castCmd.Flags().BoolVarP(&showAll, "all", "a", false, "Ignore the skip list and show all files")
	castCmd.Flags().StringSliceVarP(&skipDirs, "skip", "s", []string{
		"node_modules",
		".git",
		"build",
		"data",
	}, "Comma-separated list of folders to skip")
	castCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path to save the directory tree")
}

func runCast(cmd *cobra.Command, args []string) {
	target := "."
	if len(args) > 0 {
		target = args[0]
	}

	var w io.Writer = os.Stdout
	var file *os.File
	var buf *bufio.Writer

	if outputFile != "" {
		dir := filepath.Dir(outputFile)
		if dir != "." && dir != "" {
			_ = os.MkdirAll(dir, 0755)
		}
		f, err := os.Create(outputFile)
		if err != nil {
			color.Red("Failed to create output file %s: %v", outputFile, err)
			return
		}
		file = f
		defer file.Close()
		buf = bufio.NewWriter(file)
		defer buf.Flush()
		w = buf
		if target == "." {
			color.Cyan("Generating directory tree...")
		} else {
			color.Cyan("Generating directory tree for %s...", target)
		}
	} else {
		color.Cyan("Printing Files")
	}

	walk(target, "", w)

	if outputFile != "" {
		if buf != nil {
			_ = buf.Flush()
		}
		color.Green("✔ Directory tree successfully written to %s", outputFile)
	}
}

func walk(targetPath string, prefix string, w io.Writer) {
	dir, err := os.ReadDir(targetPath)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	for i, file := range dir {
		if !showAll && shouldSkip(file.Name()) {
			continue
		}
		isLast := i == len(dir)-1
		pointer := ""
		if isLast {
			pointer = "└── "
		} else {
			pointer = "├── "
		}

		path := filepath.Join(targetPath, file.Name())

		fmt.Fprintln(w, prefix+pointer+file.Name())

		if file.IsDir() {
			if isLast {
				walk(path, prefix+"    ", w)
			} else {
				walk(path, prefix+"│   ", w)
			}
		}
	}
}

func shouldSkip(fileName string) bool {
	for _, s := range defaultSkips {
		if s == fileName {
			return true
		}
	}
	for _, s := range skipDirs {
		if s == fileName {
			return true
		}
	}
	return false
}
