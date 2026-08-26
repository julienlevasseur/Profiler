package cmd

import (
	"fmt"
	"os"
	"slices"

	"github.com/julienlevasseur/profiler/config"
	"github.com/julienlevasseur/profiler/repository"
	"github.com/spf13/cobra"
)

// filterProfiles narrows profiles to the names in allowed, keeping the order
// the repository reported them in. An empty allow-list means no limit.
//
// A name in the allow-list that the repository does not hold is simply absent
// from the output, without a warning: one config file shared across machines
// that each hold a different subset of the profiles is the case these options
// are for.
func filterProfiles(allowed, profiles []string) []string {
	if len(allowed) == 0 {
		return profiles
	}

	kept := make([]string, 0, len(profiles))

	for _, profile := range profiles {
		if slices.Contains(allowed, profile) {
			kept = append(kept, profile)
		}
	}

	return kept
}

// appendRepoToProfileName suffixes each profile with the repository it came
// from, so a name held by several repositories stays distinguishable. Local
// profiles are never suffixed, and remote ones only when showRepoSource is
// set -- which mirrors the showRepoSourceInList config option.
func appendRepoToProfileName(
	repo string,
	showRepoSource bool,
	profiles []string,
) []string {
	if !showRepoSource || repo == "local" {
		return profiles
	}

	for i, profile := range profiles {
		profiles[i] = fmt.Sprintf("%s (%s)", profile, repo)
	}

	return profiles
}

// appendProfilesToOutput lists the profiles held by repo, narrowed to the
// allow-list cfg configures for it and suffixed with the repository name when
// cfg asks for it.
func appendProfilesToOutput(repo string, cfg config.Config) (profiles []string) {
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

		rp = filterProfiles(cfg.ProfileAllowList(repo), rp)

		profiles = append(
			profiles,
			appendRepoToProfileName(repo, cfg.ShowRepoSourceInList, rp)...,
		)
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

		cfg := config.Get()

		var profiles []string

		for _, r := range cfg.SupportedRepositories {
			profiles = append(
				profiles,
				appendProfilesToOutput(r, cfg)...,
			)
		}

		for _, p := range profiles {
			fmt.Println(p)
		}
	},
}

func init() {
	RootCmd.AddCommand(listCmd)
}
