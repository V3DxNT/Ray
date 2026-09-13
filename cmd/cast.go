package cmd

import (
	"fmt"
	"os"
	"path/filepath"

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

var skipDirs []string

func init() {
	RootCmd.AddCommand(castCmd)
}

func run(cmd *cobra.Command, args []string) {
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
		isLast := i == len(dir)-1
		pointer := ""
		if isLast {
			pointer = "|   "
		} else {
			pointer = "-"
		}

		path := filepath.Join(targetPath, file.Name())

		fmt.Println(prefix + pointer + file.Name())

		if file.IsDir() {
			walk(path, prefix+"└── ")
		}

	}
}
