package cmd

import (
	"fmt"
	"os"

	"github.com/julienlevasseur/profiler/pkg/profile"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var addCmd = &cobra.Command{
	Use:   "add [profile_name] [ENV_VAR=value]",
	Short: "add the given profile or the given env var to the profile",
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
			filePath := viper.GetString("profilerFolder") + "/." + profileName + ".yml"

			// Local profile
			var key string
			if len(args) <= 2 {
				key = ""
			} else {
				key = args[1]
			}

			var value string
			if len(args) < 3 {
				value = ""
			} else {
				value = args[2]
			}

			alreadyExist, _, err := profile.FoundInfFile(
				filePath,
				key,
			)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			if alreadyExist {
				fmt.Fprintln(
					os.Stderr,
					fmt.Sprintf("The provided variable already exist in %s", profileName),
				)
				os.Exit(1)
			}

			err = profile.AppendToFile(
				filePath,
				profileName,
				key,
				value,
			)

			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(addCmd)
}
