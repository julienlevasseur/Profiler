package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	//"github.com/julienlevasseur/profiler/profile"
)

/*RootCmd root command*/
var RootCmd = &cobra.Command{
	Use:   "profiler",
	Short: "A tool to manage your env vars as profiles.",
	Long: `Profiler is simple tool that allow you to manage your
environment variables.`,
	Run: func(cmd *cobra.Command, args []string) {
		//profile.UseNoProfile()
	},
}

/*Execute is used in main.go*/
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
