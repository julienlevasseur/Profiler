package cmd

import (
	"fmt"
	"os"

	"github.com/julienlevasseur/profiler/config"
	"github.com/spf13/cobra"
)

/* RootCmd root command */
var RootCmd = &cobra.Command{
	Use:   "profiler",
	Short: "A tool to manage your env vars as profiles.",
	Long: `Profiler is simple tool that allow you to manage your
environment variables.`,
	Run: func(cmd *cobra.Command, args []string) {},
}

/* Execute is used in main.go */
func Execute() {
	err := RootCmd.Execute()
	cmdErrorHandler(err)
}

func init() {
	cobra.OnInitialize(config.InitCfg)
}

func cmdErrorHandler(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
