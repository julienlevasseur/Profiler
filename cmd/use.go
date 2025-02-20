package cmd

import (
	"fmt"
	"os"

	"github.com/julienlevasseur/profiler/profile"
	"github.com/julienlevasseur/profiler/repository"
	"github.com/julienlevasseur/profiler/repository/local"
	"github.com/spf13/cobra"
)

var useCmd = &cobra.Command{
	Use:   "use [profile_name]",
	Short: "use the given profile (if no profile specified, profiler will load .profiler file if found)",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			p, err := local.GetDotProfiler()
			cmdErrorHandler(err)

			err = profile.SetEnvironment(p)
			cmdErrorHandler(err)

		} else if args[0] == "help" {
			cmd.Help()
			os.Exit(0)
		} else {
			p := profile.New()

			repo := args[0]
			if repo == "consul" {
				c, err := repository.GetRepository("consul")
				cmdErrorHandler(err)

				p, err = c.Get(args[1])
				cmdErrorHandler(err)

			} else if repo == "ssm" {
				s, err := repository.GetRepository("ssm")
				cmdErrorHandler(err)
				p, err = s.Get(args[1])
				cmdErrorHandler(err)

			} else if repo == "vault" {
				v, err := repository.GetRepository("vault")
				cmdErrorHandler(err)

				p, err := v.Get(args[1])
				cmdErrorHandler(err)

				fmt.Println(p)
				// for _, pName := range p {
				// fmt.Println(pName)
				// }
			} else {
				// if no repo is specified as first arg, we assume the local
				// repo is meant to be used:
				l, err := repository.GetRepository("local")
				cmdErrorHandler(err)

				p, err := l.Get(args[0])
				cmdErrorHandler(err)

				err = profile.SetEnvironment(p)
				cmdErrorHandler(err)
			}

			err := profile.SetEnvironment(p)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(useCmd)
}
