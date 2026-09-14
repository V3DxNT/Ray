package cmd

import (
	"os"
	"path/filepath"
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

var skippedList []string
var defaultSkippedList []string = []string{
	"node_modules",
	".next",
	"build",
	".git",
}

func init() {
	statsCmd.Flags().StringSliceVarP(&skippedList, "skip", "s", []string{
		"node_modules",
		".next",
		"build",
		".git",
	}, "skipped files")
	RootCmd.AddCommand(statsCmd)
}

func stat(cmd *cobra.Command, args []string) {

	extCounts = make(map[string]int)

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
	wg.Add(1)
	go func() {
		defer wg.Done()
		analze(targetDir)
	}()
	wg.Wait()
}

func analze(currentDir string) {
	sem <- struct{}{}

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
				defer func() { <-sem }()
				analze(filepath.Join(currentDir, dirEntry.Name()))
			}()
		} else {
			extension := filepath.Ext(dirEntry.Name())
			extCounts[extension]++
			totalFiles++
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
