package cmd

import (
	"fmt"
	"strings"

	"github.com/julienlevasseur/profiler/pkg/profile"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list profiles",
	Run: func(cmd *cobra.Command, args []string) {
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
	},
}

func init() {
	RootCmd.AddCommand(listCmd)
}
