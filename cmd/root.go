package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "ray",
	Short: "A developer toolkit to explore, check, clean, and organize your code",
	Long:  long,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Ray CLI v" + cmd.Version)
	},
	Version: "3.0.0",
}

var long string = "Ray is a friendly command-line tool that helps you understand, clean up,\n" +
	"and manage your projects.\n\n" +
	"It gives you a visual directory tree, checks what tools and packages are installed,\n" +
	"scans for leaked secrets and missing environment variables, cleans up heavy build\n" +
	"cache folders, finds duplicate files, and creates new project structures.\n\n" +
	"Core Commands:\n" +
	"  cast      Show a visual tree of folders and files\n" +
	"  stats     Show code languages, file counts, and sizes\n" +
	"  search    Find files by name across your project\n" +
	"  doctor    Check if required runtimes, tools, and compilers are installed\n" +
	"  setup     Install missing tools and set up project dependencies\n" +
	"  clean     Find and remove build cache, temporary files, and heavy folders\n" +
	"  dedup     Find identical duplicate files across your project\n" +
	"  audit     Check for leaked secrets, standard files, and code metrics\n" +
	"  env       Find environment variables used in code and check against .env\n" +
	"  ignore    Generate .gitignore and .dockerignore files for your project\n" +
	"  scaffold  Create project folders and starter files from a blueprint or interactively"

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
