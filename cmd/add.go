package cmd

import (
	"fmt"
	"os"

	"github.com/julienlevasseur/profiler/pkg/failure"
	"github.com/julienlevasseur/profiler/pkg/profile"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add [profile_name] [ENV_VAR] [value]",
	Short: "add the given profile (if only profile_name is provided) or the given env var to the profile",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Printf(
				`You have to provide at least a profile name to create.\n
				By mentioning an existing profile, you can add new variable to 
				it.\n`,
			)
		} else if args[0] == "help" || args[0] == "" {
			cmd.Help()
			os.Exit(0)
		} else {
			profileName := args[0]

			p := profile.Profile{
				Name: profileName,
				Type: "local",
			}

			err := profile.Create(p, args)
			if err != nil {
				failure.ExitOnError(err)
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(addCmd)
}
