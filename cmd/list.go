package cmd

import (
	"fmt"
	"os"

	"github.com/julienlevasseur/profiler/repository"
	"github.com/spf13/cobra"
)

var supportedRepositories = []string{
	"local",
	"consul",
	"ssm",
	"vault",
}

func appendRepoToProfileName(repo string, profiles []string) []string {
	for i, profile := range profiles {
		profiles[i] = fmt.Sprintf("%s (%s)", profile, repo)
	}

	return profiles
}

func appendProfilesToOutput(repo string) (profiles []string) {
	r, err := repository.GetRepository(repo)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if r.IsConfigured() {
		rp, err := r.List()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if repo != "local" {
			rp = appendRepoToProfileName(repo, rp)
		}
		profiles = append(profiles, rp...)
	}

	return profiles
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list profiles",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 0 && args[0] == "help" {
			cmd.Help()
			os.Exit(0)
		}

		var profiles []string

		for _, r := range supportedRepositories {
			profiles = append(profiles, appendProfilesToOutput(r)...)
		}

		for _, p := range profiles {
			fmt.Println(p)
		}
	},
}

func init() {
	RootCmd.AddCommand(listCmd)
}
