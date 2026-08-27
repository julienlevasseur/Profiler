package cmd

import (
	"fmt"
	"os"

	"github.com/julienlevasseur/profiler/pkg/profile"
	"github.com/julienlevasseur/profiler/repository"
	"github.com/spf13/cobra"
)

const ssmRepo = "ssm"

var ssmCmd = &cobra.Command{
	Use:   "ssm",
	Short: "deal with remote profiles stored in AWS SSM",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 || args[0] == "help" {
			cmd.Help()
			os.Exit(0)
		}
	},
}

var ssmAddCmd = &cobra.Command{
	Use:   "add [profile_name] [ENV_VAR] [value]",
	Short: "add the given profile or the given env var to the SSM profile",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		s, err := repository.GetRepository(ssmRepo)
		cmdErrorHandler(err)

		cmdErrorHandler(s.Add(args))
	},
}

var ssmListCmd = &cobra.Command{
	Use:   "list",
	Short: "list remote profiles stored in AWS SSM",
	Run: func(cmd *cobra.Command, args []string) {
		s, err := repository.GetRepository(ssmRepo)
		cmdErrorHandler(err)

		profiles, err := s.List()
		cmdErrorHandler(err)

		for _, p := range profiles {
			fmt.Println(p)
		}
	},
}

var ssmRemoveCmd = &cobra.Command{
	Use:   "remove [profile_name] [ENV_VAR]",
	Short: "remove the given profile or the given env var from the remote profile stored in AWS SSM",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		s, err := repository.GetRepository(ssmRepo)
		cmdErrorHandler(err)

		cmdErrorHandler(s.Remove(args))
	},
}

var ssmShowCmd = &cobra.Command{
	Use:   "show [profile_name]",
	Short: "show given profile(s) variables name",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		s, err := repository.GetRepository(ssmRepo)
		cmdErrorHandler(err)

		for _, p := range args {
			vars, err := s.Show(p)
			cmdErrorHandler(err)

			// Display Profile's name:
			fmt.Printf("%s:\n", p)
			// Display each Profile's env var name:
			for _, v := range vars {
				fmt.Printf("- %s\n", v)
			}
			fmt.Printf("\n")
		}
	},
}

var ssmUseCmd = &cobra.Command{
	Use:   "use [profile_name]",
	Short: "use the given SSM profile",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		s, err := repository.GetRepository(ssmRepo)
		cmdErrorHandler(err)

		p, err := s.Get(args[0])
		cmdErrorHandler(err)

		// StackEnvironment ends in syscall.Exec, so on success it does not
		// return: this is the last thing `profiler ssm use` does.
		cmdErrorHandler(profile.StackEnvironment(p))
	},
}

func init() {
	ssmCmd.AddCommand(ssmAddCmd)
	ssmCmd.AddCommand(ssmListCmd)
	ssmCmd.AddCommand(ssmRemoveCmd)
	ssmCmd.AddCommand(ssmShowCmd)
	ssmCmd.AddCommand(ssmUseCmd)
	RootCmd.AddCommand(ssmCmd)
}
