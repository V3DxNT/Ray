package cmd

import (
	"os"
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
var framewords []string
var totalFiles int

func init() {
	extCounts = make(map[string]int)
	statsCmd.AddCommand(statsCmd)
}

func stat(cmd *cobra.Command, args []string) {

	if len(args) > 1 {
		color.Red("Too many arguments")
		return
	}

	if len(args) == 0 {
		pwd, _ := os.Getwd()
		analze(pwd)
	} else {
		analze(args[0])
	}
}

func analze(currentDir string) {

}
