package cmd

import (
	"fmt"

	"github.com/julienlevasseur/profiler/pkg/failure"
	"github.com/julienlevasseur/profiler/pkg/repository"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list profiles",
	Run: func(cmd *cobra.Command, args []string) {
		profiles, err := listProfiles()
		if err != nil {
			failure.ExitOnError(err)
		}

		for profileType, profiles := range profiles {
			fmt.Printf("[%v]\n", profileType)
			for _, profileName := range profiles {
				fmt.Printf("  %v\n", profileName)
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(listCmd)
}

func listProfiles() (map[string][]string, error) {
	repo := repository.Repository{
		Path: viper.GetString("repositoryPath"),
	}

	profiles := make(map[string][]string)

	prof, err := repo.List()
	if err != nil {
		return map[string][]string{}, err
	}

	for _, p := range prof {
		profiles[p.ProfileType] = append(profiles[p.ProfileType], p.Name)
	}

	return profiles, nil
}
