package cmd

import (
	"fmt"
	"os"

	"github.com/julienlevasseur/profiler/pkg/consul"
	"github.com/julienlevasseur/profiler/pkg/failure"
	"github.com/julienlevasseur/profiler/pkg/profile"
	"github.com/spf13/cobra"
)

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
			profileName := args[0]

			p := profile.Profile{
				Name: profileName,
				Type: "consul",
			}

			err := profile.Create(p, args)
			if err != nil {
				failure.ExitOnError(err)
			}
		}
	},
}

var consulListCmd = &cobra.Command{
	Use:   "list",
	Short: "list remote profiles stored in Consul",
	Run: func(cmd *cobra.Command, args []string) {
		// List SSM Parameter Store Profiles
		profiles, err := consul.ListProfiles()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		for _, p := range profiles {
			fmt.Println(p)
		}
	},
}

// var consulRemoveCmd = &cobra.Command{
// 	Use:   "remove [profile_name] [ENV_VAR]",
// 	Short: "remove the given profile or the given env var from the remote profile stored in Consul",
// 	Run: func(cmd *cobra.Command, args []string) {
// 		// Check first if the given profile exists
// 		profileExist, err := consul.ProfileExist(args[0])
// 		if err != nil {
// 			fmt.Fprintln(os.Stderr, err)
// 			os.Exit(1)
// 		}

// 		if !profileExist {
// 			err := errors.New("The provided Profile does not exist")
// 			fmt.Fprintln(os.Stderr, err)
// 			os.Exit(1)
// 		} else {
// 			// check if a variable has been provided or just a profile name:
// 			if len(args) < 3 {
// 				// Only the profile name provided, delete all related params:
// 				err := consul.DeleteKey("/profiler/" + args[0])
// 				if err != nil {
// 					fmt.Fprintln(os.Stderr, err)
// 					os.Exit(1)
// 				}
// 			} else {
// 				profile, err := consul.GetKVPair("/profiler/" + args[0])
// 				if err != nil {
// 					fmt.Fprintln(os.Stderr, err)
// 					os.Exit(1)
// 				}
// 				fmt.Println(profile)
// 				//err := consul.DeleteKey(
// 				//	"/profiler/" + args[1] + "/" + args[2],
// 				//)
// 				//if err != nil {
// 				//	fmt.Fprintln(os.Stderr, err)
// 				//	os.Exit(1)
// 				//}
// 			}
// 		}
// 	},
// }

// var consulShowCmd = &cobra.Command{
// 	Use:   "show [profile_name]",
// 	Short: "show given profile(s) variables name",
// 	Run: func(cmd *cobra.Command, args []string) {
// 		for _, p := range args {
// 			vars, err := consul.ShowProfile(p)
// 			if err != nil {
// 				fmt.Fprintln(os.Stderr, err)
// 				os.Exit(1)
// 			}

// 			// Display Profile's name:
// 			fmt.Printf("%s:\n", p)
// 			// Display each Profile's env var name:
// 			for _, v := range vars {
// 				fmt.Printf("- %s\n", v)
// 			}
// 			fmt.Printf("\n")
// 		}
// 	},
// }

func init() {
	consulCmd.AddCommand(consulAddCmd)
	//consulCmd.AddCommand(consulListCmd)
	//consulCmd.AddCommand(consulRemoveCmd)
	//consulCmd.AddCommand(consulShowCmd)
	RootCmd.AddCommand(consulCmd)
}
