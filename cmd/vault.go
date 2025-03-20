package cmd

import (
	"fmt"
	"os"

	"github.com/julienlevasseur/profiler/pkg/profile"
	"github.com/julienlevasseur/profiler/repository"
	"github.com/spf13/cobra"
)

const vaultRepo = "vault"

var vaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "deal with remote profiles stored in Hashicorp Vault",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 || args[0] == "help" {
			cmd.Help()
			os.Exit(0)
		}
	},
}

var vaultAddCmd = &cobra.Command{
	Use:   "add [profile_name] [ENV_VAR=value]",
	Short: "add the given profile or the given env var to the vault profile",
	Run: func(cmd *cobra.Command, args []string) {

		c, err := repository.GetRepository(vaultRepo)
		cmdErrorHandler(err)

		c.Add(args)
	},
}

var vaultListCmd = &cobra.Command{
	Use:   "list",
	Short: "list remote profiles stored in Vault",
	Run: func(cmd *cobra.Command, args []string) {
		v, err := repository.GetRepository(vaultRepo)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		profiles, err := v.List()
		cmdErrorHandler(err)

		for _, p := range profiles {
			fmt.Println(p)
		}
	},
}

var vaultRemoveCmd = &cobra.Command{
	Use:   "remove [profile_name] [ENV_VAR]",
	Short: "remove the given profile or the given env var from the remote profile stored in Vault",
	Run: func(cmd *cobra.Command, args []string) {

		v, err := repository.GetRepository(vaultRepo)
		cmdErrorHandler(err)

		v.Remove(args)
	},
}

var vaultShowCmd = &cobra.Command{
	Use:   "show [profile_name]",
	Short: "show given profile(s) variables name",
	Run: func(cmd *cobra.Command, args []string) {
		v, err := repository.GetRepository(vaultRepo)
		cmdErrorHandler(err)

		for _, p := range args {
			vars, err := v.Show(p)
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

var vaultUseCmd = &cobra.Command{
	Use:   "use [profile_name]",
	Short: "use the given Vault profile",
	Run: func(cmd *cobra.Command, args []string) {
		for _, p := range args {
			v, err := repository.GetRepository(vaultRepo)
			cmdErrorHandler(err)

			prof, err := v.Get(p)
			cmdErrorHandler(err)

			err = profile.SetEnvironment(prof)
			cmdErrorHandler(err)
		}
	},
}

func init() {
	vaultCmd.AddCommand(vaultAddCmd)
	vaultCmd.AddCommand(vaultListCmd)
	vaultCmd.AddCommand(vaultRemoveCmd)
	vaultCmd.AddCommand(vaultShowCmd)
	vaultCmd.AddCommand(vaultUseCmd)
	RootCmd.AddCommand(vaultCmd)
}
