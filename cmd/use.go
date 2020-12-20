package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/julienlevasseur/profiler/pkg/profile"
)

var useCmd = &cobra.Command{
	Use:   "use [profile_name]",
	Short: "use the given profile",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			profile.UseNoProfile()
		} else if args[0] == "help" {
			cmd.Help()
			os.Exit(0)
		} else {
			profile.Use(
				viper.GetString("profilerFolder"),
				args[0],
			)
		}
	},
}

func init() {
	RootCmd.AddCommand(useCmd)
}
