package cmd

import (
	"fmt"
	"os"

	"github.com/julienlevasseur/profiler/pkg/profile"
	"github.com/julienlevasseur/profiler/pkg/ssm"
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
			if args[0] == "ssm" {
				value, err := ssm.Get()
				if err != nil {
					panic(err)
				}

				fmt.Println(value)
			} else if args[0] == "consul" {
				// Adding a env var to a Consul profile
				fmt.Println("Not implemented yet")

			} else {
				// Local profile
				var value string
				if len(args) <= 2 {
					value = ""
				} else {
					value = args[1]
				}
				err := profile.AppendToFile(
					viper.GetString("profilerFolder")+"/."+args[0]+".yml",
					value+"\n",
				)

				if err != nil {
					panic(err)
				}
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(addCmd)
}
