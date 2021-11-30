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

		yamlFiles := profile.ListFiles(
			viper.GetString("profilerFolder"),
			"*.yaml",
		)

		files = append(files, yamlFiles...)

		for _, file := range files {
			fmt.Println(
				strings.Split(
					strings.Split(
						file,
						fmt.Sprintf(
							"%s/.",
							viper.GetString("profilerFolder"),
						),
					)[1], ".y", // The separator here is '.y' to support both .yml and .yaml files
				)[0],
			)
		}
	},
}

func init() {
	RootCmd.AddCommand(listCmd)
}
