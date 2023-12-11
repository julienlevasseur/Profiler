package cmd

// import (
// 	"fmt"
// 	"os"

// 	"github.com/julienlevasseur/profiler/pkg/profile"
// 	"github.com/spf13/cobra"
// 	"github.com/spf13/viper"
// )

// var removeCmd = &cobra.Command{
// 	Use:   "remove [profile_name] [ENV_VAR]",
// 	Short: "remove the given profile or the given env var from the profile",
// 	Run: func(cmd *cobra.Command, args []string) {
// 		if len(args) == 0 {
// 			fmt.Printf(
// 				`You have to provide at least a profile name to remove.\n
// 				By just mentioning an profile name, the entire profile will be
// 				deleted.\n`,
// 			)
// 		} else if args[0] == "help" || args[0] == "" {
// 			cmd.Help()
// 			os.Exit(0)
// 		} else {
// 			// check if a variable has been provided or just a profile name:
// 			if len(args) < 2 {
// 				// Only the profile name provided, delete the file:
// 				err := os.Remove(
// 					viper.GetString("profilesFolder") + "/." + args[0] + ".yml",
// 				)
// 				if err != nil {
// 					fmt.Fprintln(os.Stderr, err)
// 					os.Exit(1)
// 				}
// 			} else {
// 				profile.RemoveFromFile(
// 					viper.GetString("profilesFolder")+"/."+args[0]+".yml",
// 					args[1],
// 				)
// 			}
// 		}
// 	},
// }

// func init() {
// 	RootCmd.AddCommand(removeCmd)
// }
