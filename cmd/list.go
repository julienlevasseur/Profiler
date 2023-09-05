package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/julienlevasseur/profiler/pkg/consul"
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

		if viper.GetString("consulAddress") != "" || viper.GetString("ssmRegion") != "" {
			fmt.Println("[Local Profiles]")
		}

		files = append(files, yamlFiles...)
		listLocalProfiles(files)

		// Checking for Consul config:
		if viper.GetString("consulAddress") != "" {
			consulProfiles, err := consul.ListProfiles()
			if err != nil {
				log.Printf("Error while listing Consul profiles: %w", err)
			}
			fmt.Println("\n[Consul Remote Profiles]")
			for _, profile := range consulProfiles {
				fmt.Println(profile)
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(listCmd)
}

func listLocalProfiles(files []string) {
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
}
