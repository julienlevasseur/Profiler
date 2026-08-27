package cmd

import (
	"fmt"
	"os"

	"github.com/julienlevasseur/profiler/repository"
	"github.com/spf13/cobra"
)

const localRepo = "local"

var addCmd = &cobra.Command{
	Use:   "add [profile_name] [ENV_VAR] [value]",
	Short: "add the given profile (if only profile_name is provided) or the given env var to the profile",
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) == 0 {
			fmt.Printf(
				`You have to provide at least a profile name to create.\n
				By mentioning an existing profile, you can add new variable to
				it.\n`,
			)
		} else if args[0] == "help" || args[0] == "" {
			cmd.Help()
			os.Exit(0)
		} else {

			l, err := repository.GetRepository(localRepo)
			cmdErrorHandler(err)

			cmdErrorHandler(l.Add(args))
		}
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove [profile_name] [ENV_VAR]",
	Short: "remove the given profile or the given env var from the profile",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Printf(
				`You have to provide at least a profile name to remove.\n
				By just mentioning an profile name, the entire profile will be
				deleted.\n`,
			)
		} else if args[0] == "help" || args[0] == "" {
			cmd.Help()
			os.Exit(0)
		} else {
			l, err := repository.GetRepository(localRepo)
			cmdErrorHandler(err)

			// The repository decides what a bare profile name means: with no
			// variable after it, the whole profile goes.
			cmdErrorHandler(l.Remove(args))
		}
	},
}

var showCmd = &cobra.Command{
	Use:   "show [profile_name]",
	Short: "show given profile(s) variables name",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println(
				"You have to provide a profile name to show it's content. ",
				"You can pass multiple profiles.",
			)
		} else {
			l, err := repository.GetRepository(localRepo)
			cmdErrorHandler(err)

			for _, p := range args {
				vars, err := l.Show(p)
				cmdErrorHandler(err)

				// Display Profile's name:
				fmt.Printf("%s:\n", p)
				// Display each Profile's env var name:
				for _, v := range vars {
					fmt.Printf("- %s\n", v)
				}
				fmt.Printf("\n")
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(addCmd)
	RootCmd.AddCommand(removeCmd)
	RootCmd.AddCommand(showCmd)
}
