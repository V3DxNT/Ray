package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/V3DxNT/Ray/internal/fsutil"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var castCmd = &cobra.Command{
	Use:   "cast [path]",
	Short: "Show a visual tree of folders and files",
	Long:  "Display a clean, color-coded visual tree of your folders and files. Ray automatically skips noisy folders like node_modules and .git by default so you can focus on your code.",
	Example: `  ray cast
  ray cast src/
  ray cast --output tree.txt
  ray cast --all
  ray cast --skip dist,coverage`,
	Run: runCast,
}

var skipDirs []string
var showAll bool = false
var outputFile string

func init() {
	RootCmd.AddCommand(castCmd)
	castCmd.Flags().BoolVarP(&showAll, "all", "a", false, "Show all files and folders without skipping anything")
	castCmd.Flags().StringSliceVarP(&skipDirs, "skip", "s", []string{
		"node_modules",
		".git",
		"build",
		"data",
	}, "Comma-separated list of folders or files to skip")
	castCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Save the directory tree to a text file instead of printing")
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
		if !showAll && fsutil.ShouldSkip(file.Name(), skipDirs) {
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
