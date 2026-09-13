package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

/*
This is for showing the ASCII tree
*/

var castCmd = &cobra.Command{
	Use:   "cast",
	Short: "Cast commands",
	Run:   run,
}

var skipDirs = []string{"node_modules", ".git", "data", "build"}
var showAll bool = false
var castHelpMessage = ""

func init() {
	RootCmd.AddCommand(castCmd)
	castCmd.Flags().BoolVarP(&showAll, "all", "a", false,
		"Ignore the skip list and show all files")

	castCmd.Flags().StringSliceVarP(&skipDirs, "skip", "s", []string{
		"node_modules",
		".git",
		"build",
		"data",
	},
		"Comma-separated list of folders to skip",
	)
}

func run(cmd *cobra.Command, args []string) {
	color.Cyan("Printing Files")
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(pwd)

	if len(args) == 0 {
		walk(".", "")
	} else {
		walk(args[0], "")
	}

}

func walk(targetPath string, prefix string) {

	dir, err := os.ReadDir(targetPath)
	if err != nil {
		fmt.Println(err)
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

		fmt.Println(prefix + pointer + file.Name())

		if file.IsDir() {
			if isLast {
				walk(path, prefix+"    ")
			} else {
				walk(path, prefix+"│   ")
			}
		}

	}
}

func shouldSkip(fileName string) bool {
	for _, s := range skipDirs {
		if s == fileName {
			return true
		}
	}
	return false
}
