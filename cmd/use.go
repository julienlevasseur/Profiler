package cmd

import (
	"fmt"
	"os"
	"slices"

	"github.com/julienlevasseur/profiler/pkg/local"
	"github.com/julienlevasseur/profiler/pkg/profile"
	"github.com/julienlevasseur/profiler/repository"
	"github.com/spf13/cobra"
)

// remoteRepos are the repositories addressed as `profiler use <repo> <name>`.
// Anything else in args[0] is taken to be a local profile name.
var remoteRepos = []string{"consul", "ssm", "vault"}

// resolveTarget works out which repository to ask and which profile to ask it
// for, from the arguments `profiler use` was given.
func resolveTarget(args []string) (repo, profileName string, err error) {
	if !slices.Contains(remoteRepos, args[0]) {
		return "local", args[0], nil
	}

	// A remote repository is named by args[0] and takes the profile name in
	// args[1].
	if len(args) < 2 {
		return "", "", fmt.Errorf(
			"missing profile name: profiler use %s <profile_name>",
			args[0],
		)
	}

	return args[0], args[1], nil
}

var useCmd = &cobra.Command{
	Use:   "use [profile_name]",
	Short: "use the given profile (if no profile specified, profiler will load .profiler file if found)",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			useNoProfile()
			return
		}

		if args[0] == "help" {
			cmd.Help()
			os.Exit(0)
		}

		repo, profileName, err := resolveTarget(args)
		cmdErrorHandler(err)

		r, err := repository.GetRepository(repo)
		cmdErrorHandler(err)

		p, err := r.Get(profileName)
		cmdErrorHandler(err)

		// SetEnvironment ends in syscall.Exec, so on success it does not
		// return: this is the last thing `profiler use` does.
		cmdErrorHandler(profile.SetEnvironment(p))
	},
}

// useNoProfile sources the local env files (.profiler, any *.env, .env.yml
// and .envrc, in that precedence order) with no named profile. Shared by bare
// `profiler` and by `profiler use` with no argument, which are documented as
// the same operation.
func useNoProfile() {
	p, err := local.GetDotProfiler()
	cmdErrorHandler(err)

	err = profile.SetEnvironment(p)
	cmdErrorHandler(err)
}

func init() {
	RootCmd.AddCommand(useCmd)
}
