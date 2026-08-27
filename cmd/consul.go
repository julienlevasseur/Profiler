package cmd

import (
	"fmt"
	"os"

	"github.com/julienlevasseur/profiler/pkg/profile"
	"github.com/julienlevasseur/profiler/repository"
	"github.com/spf13/cobra"
)

const consulRepo = "consul"

var consulCmd = &cobra.Command{
	Use:   "consul",
	Short: "deal with remote profiles stored in Consul",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 || args[0] == "help" {
			cmd.Help()
			os.Exit(0)
		}
	},
}

var consulAddCmd = &cobra.Command{
	Use:   "add [profile_name] [ENV_VAR=value]",
	Short: "add the given profile or the given env var to the consul profile",
	Run: func(cmd *cobra.Command, args []string) {

		c, err := repository.GetRepository(consulRepo)
		cmdErrorHandler(err)

		c.Add(args)
	},
}

var consulListCmd = &cobra.Command{
	Use:   "list",
	Short: "list remote profiles stored in Consul",
	Run: func(cmd *cobra.Command, args []string) {
		c, err := repository.GetRepository(consulRepo)
		cmdErrorHandler(err)

		profiles, err := c.List()

		for _, p := range profiles {
			fmt.Println(p)
		}
	},
}

var consulRemoveCmd = &cobra.Command{
	Use:   "remove [profile_name] [ENV_VAR]",
	Short: "remove the given profile or the given env var from the remote profile stored in Consul",
	Run: func(cmd *cobra.Command, args []string) {

		c, err := repository.GetRepository(consulRepo)
		cmdErrorHandler(err)

		c.Remove(args)
	},
}

var consulShowCmd = &cobra.Command{
	Use:   "show [profile_name]",
	Short: "show given profile(s) variables name",
	Run: func(cmd *cobra.Command, args []string) {
		c, err := repository.GetRepository(consulRepo)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		for _, p := range args {
			vars, err := c.Show(p)
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

var consulUseCmd = &cobra.Command{
	Use:   "use [profile_name]",
	Short: "use the given Vault profile",
	Run: func(cmd *cobra.Command, args []string) {

		c, err := repository.GetRepository(consulRepo)
		cmdErrorHandler(err)

		for _, p := range args {
			// prof, err := consul.GetProfile(p)
			prof, err := c.Get(p)
			cmdErrorHandler(err)

			err = profile.StackEnvironment(prof)
			cmdErrorHandler(err)
		}
	},
}

func init() {
	consulCmd.AddCommand(consulAddCmd)
	consulCmd.AddCommand(consulListCmd)
	consulCmd.AddCommand(consulRemoveCmd)
	consulCmd.AddCommand(consulShowCmd)
	consulCmd.AddCommand(consulUseCmd)
	RootCmd.AddCommand(consulCmd)
}
