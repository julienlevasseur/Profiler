package cmd

import (
	"errors"
	"fmt"

	"github.com/julienlevasseur/profiler/pkg/consul"
	"github.com/julienlevasseur/profiler/pkg/failure"
	"github.com/julienlevasseur/profiler/pkg/local"
	"github.com/julienlevasseur/profiler/pkg/repository"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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
			for _, p := range args {
				repo := repository.Repository{
					Path: viper.GetString("repositoryPath"),
				}
				prof, err := repo.Get(p)
				if err != nil {
					failure.ExitOnError(err)
				}

				if prof.ProfileType == "consul" {
					consulCfg, err := consul.NewConsulConfig()
					if err != nil {
						failure.ExitOnError(err)
					}
					if consulCfg.Enabled {
						cp := consul.ConsulProfile{Name: p}
						cp.Show()
					}
				} else if prof.ProfileType == "local" {
					lp := local.LocalProfile{Name: p}
					lp.Show()
				} else {
					err := errors.New("no matching profile (locally or remotely)")
					failure.ExitOnError(err)
				}
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(showCmd)
}
