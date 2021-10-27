package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/julienlevasseur/profiler/pkg/profile"
	"github.com/julienlevasseur/profiler/pkg/ssm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list profiles",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			files := profile.ListFiles(
				viper.GetString("profilerFolder"),
				".*.yml",
			)

			for _, file := range files {
				fmt.Println(
					strings.Split(
						strings.Split(
							file,
							fmt.Sprintf(
								"%s/.",
								viper.GetString("profilerFolder"),
							),
						)[1], ".yml",
					)[0],
				)
			}
		} else {
			if args[0] == "ssm" {
				// List SSM Parameter Store Profiles
				profiles, err := ssm.ListProfiles()
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}

				for _, p := range profiles {
					fmt.Println(p)
				}
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(listCmd)
}
