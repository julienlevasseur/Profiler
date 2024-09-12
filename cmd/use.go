package cmd

import (
	"errors"
	"os"

	"github.com/julienlevasseur/profiler/pkg/consul"
	"github.com/julienlevasseur/profiler/pkg/failure"
	"github.com/julienlevasseur/profiler/pkg/local"
	"github.com/julienlevasseur/profiler/pkg/repository"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var useCmd = &cobra.Command{
	Use:   "use [profile_name]",
	Short: "use the given profile",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			local.UseNoProfile()
		} else if args[0] == "help" {
			cmd.Help()
			os.Exit(0)
		} else {
			repo := repository.Repository{
				Path: viper.GetString("repositoryPath"),
			}
			prof, err := repo.Get(args[0])
			if err != nil {
				failure.ExitOnError(err)
			}

			if prof.ProfileType == "consul" {
				cp := consul.ConsulProfile{Name: args[0]}
				cp.Use()

			} else if prof.ProfileType == "local" {
				lp := local.LocalProfile{Name: args[0]}
				lp.Use()
			} else {
				err := errors.New("no matching profile (locally or remotely)")
				failure.ExitOnError(err)
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(useCmd)
}
