package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

/*
This Deep-dives from the current directory to find specific files and prints their absolute paths.
*/

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search commands",
	Run:   search,
}

var targets []string
var onlyfirst bool

func init() {
	searchCmd.Flags().BoolVarP(&onlyfirst, "first", "f", false, "Search First Occurence")
	RootCmd.AddCommand(searchCmd)
}

func search(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		fmt.Println("Search needs at least one Argument")
		return
	}

	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		return
	}

	targets = args

	hunt(currentDir)
}

func hunt(currentDir string) {
	if len(targets) == 0 {
		return
	}
	dir, err := os.ReadDir(currentDir)
	if err != nil {
		return
	}

	for _, file := range dir {
		if isTarget(file.Name()) {
			fmt.Println(filepath.Join(currentDir, color.HiCyanString(file.Name())))
		}

		if file.IsDir() {
			hunt(filepath.Join(currentDir, file.Name()))
		}

	}
}

func isTarget(filename string) bool {
	for i, arg := range targets {
		if arg == filename {
			if onlyfirst {
				targets = append(targets[:i], targets[i+1:]...)
			}
			return true
		}
	}
	return false
}
