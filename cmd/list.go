package cmd

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"

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

// getRepository is repository.GetRepository, as a variable so the listing can
// be exercised against repositories that do not need a Consul, a Vault and an
// AWS account to answer.
var getRepository = repository.GetRepository

// appendProfilesToOutput lists the profiles held by repo, narrowed to the
// allow-list cfg configures for it and suffixed with the repository name when
// cfg asks for it. A repository that is not configured holds nothing, and is
// not an error: that is what lets `profiler list` work on a machine with no
// Consul, no Vault and no AWS.
func appendProfilesToOutput(
	repo string,
	cfg config.Config,
) ([]string, error) {
	r, err := getRepository(repo)
	if err != nil {
		return nil, err
	}

	if !r.IsConfigured() {
		return nil, nil
	}

	profiles, err := r.List()
	if err != nil {
		// The repository is named because several are listed at once, so
		// the bare error would not say which one failed.
		return nil, fmt.Errorf("%s: %w", repo, err)
	}

	profiles = filterProfiles(cfg.ProfileAllowList(repo), profiles)

	return appendRepoToProfileName(repo, cfg.ShowRepoSourceInList, profiles), nil
}

// listRepositories asks every repository for its profiles at once, and returns
// what each one holds, in the order they were given.
//
// The repositories are asked in parallel because most of them answer over the
// network: an SSM listing is a TLS handshake plus a request a page (~200ms to
// begin with), and Consul and Vault cost a round trip each. Asked one after
// another, `profiler list` waited for the sum of all of them -- and printed
// nothing at all until the slowest had answered, local profiles included.
// Asked together, it waits for the slowest alone.
func listRepositories(repos []string, cfg config.Config) ([][]string, error) {
	listed := make([][]string, len(repos))
	errs := make([]error, len(repos))

	var wg sync.WaitGroup

	for i, repo := range repos {
		wg.Add(1)

		go func() {
			defer wg.Done()

			// Each goroutine writes its own slot, so no lock is needed
			// here. config.Get is the shared state, and it takes one
			// of its own.
			listed[i], errs[i] = appendProfilesToOutput(repo, cfg)
		}()
	}

	wg.Wait()

	// Joined in the order the repositories were given rather than the order
	// they failed in, so the output does not depend on which goroutine lost
	// the race.
	return listed, errors.Join(errs...)
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

		listed, err := listRepositories(cfg.SupportedRepositories, cfg)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		for _, profiles := range listed {
			for _, profile := range profiles {
				fmt.Println(profile)
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(listCmd)
}
