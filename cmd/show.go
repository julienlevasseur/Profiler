package cmd

import (
	"fmt"

	"github.com/julienlevasseur/profiler/pkg/profile"
	"github.com/julienlevasseur/profiler/pkg/ssm"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var showCmd = &cobra.Command{
	Use:   "show [profile_name]",
	Short: "show given profile(s) variables name",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println(
				"You have to provide a profile name to show it's content. ",
				"You can pass multiple profiles.",
			)
		} else {
			if args[0] == "ssm" {
				/* Poping out first element of the args slice because it is the
				"ssm" argument:*/
				args = args[1:]

				for _, p := range args {
					vars, err := ssm.ShowProfile(p)
					if err != nil {
						panic(err)
					}

					displayProfileVarNames(p, vars)
				}
			} else {
				for _, p := range args {
					vars := profile.ShowProfile(
						viper.GetString("profilerFolder"),
						p,
					)

					displayProfileVarNames(p, vars)
				}
			}
		}
	},
}

func displayProfileVarNames(profileName string, vars []string) {
	// Display Profile's name:
	fmt.Printf("%s:\n", profileName)
	// Display each Profile's env var name:
	for _, v := range vars {
		fmt.Printf("- %s\n", v)
	}
	fmt.Printf("\n")
}

func init() {
	RootCmd.AddCommand(showCmd)
}
