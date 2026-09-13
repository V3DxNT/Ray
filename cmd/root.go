package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "ray",
	Short: "Ray CLI",
	Long:  long,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Ray CLI v" + cmd.Version)
	},
	Version: "1.0.0",
}

var long string = "Long: `Ray is a high-performance, cross-platform CLI tool engineered for deep codebase introspection.\n" +
	"It maps out project architectures, hunts down buried files, and audits system health instantly." +
	"\n\nCore Commands:" +
	"\n  cast      Render a contextual, tech-aware ASCII directory tree." +
	"\n  search    Exhaustively hunt for a specific file across all directories." +
	"\n  stats     Generate a codebase dashboard (tech stack, lines of code, sizes)." +
	"\n  audit     Scan the repository for exposed secrets and missing standard files." +
	"\n  dedup     Find identical files hidden across massive repositories using SHA-256." +
	"\n  upgrade   Seamlessly self-update the Ray binary to the latest version.`,"

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
