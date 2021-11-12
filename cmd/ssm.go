package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/julienlevasseur/profiler/pkg/ssm"
	"github.com/spf13/cobra"
)

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
	Use:   "add [profile_name] [ENV_VAR=value]",
	Short: "add the given profile or the given env var to the SSM profile",
	Run: func(cmd *cobra.Command, args []string) {
		// Check first if the given profile exists
		profileExist, err := ssm.ProfileExist(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if !profileExist {
			err = ssm.AddParameter(args[0]+"/profile_name", args[0])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}

		if len(args) > 1 {
			if len(args) < 3 { // If no value provided:
				err := errors.New("Please provide a value for the variable")
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			err = ssm.AddParameter(args[0]+"/"+args[1], args[2])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
	},
}

var ssmListCmd = &cobra.Command{
	Use:   "list",
	Short: "list remote profiles stored in AWS SSM",
	Run: func(cmd *cobra.Command, args []string) {
		// List SSM Parameter Store Profiles
		profiles, err := ssm.ListProfiles()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		for _, p := range profiles {
			fmt.Println(p)
		}
	},
}

var ssmRemoveCmd = &cobra.Command{
	Use:   "remove [profile_name] [ENV_VAR]",
	Short: "remove the given profile or the given env var from the remote profile stored in AWS SSM",
	Run: func(cmd *cobra.Command, args []string) {
		// Check first if the given profile exists
		profileExist, err := ssm.ProfileExist(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if !profileExist {
			err := errors.New("The provided Profile does not exist")
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		} else {
			// check if a variable has been provided or just a profile name:
			if len(args) < 3 {
				// Only the profile name provided, delete all related params:
				params, err := ssm.ShowProfile(args[1])
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}

				for _, param := range params {
					err := ssm.RemoveParameter(
						"/profiler/" + args[1] + "/" + param,
					)
					if err != nil {
						fmt.Fprintln(os.Stderr, err)
						os.Exit(1)
					}
				}
			} else {
				err := ssm.RemoveParameter(
					"/profiler/" + args[1] + "/" + args[2],
				)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
			}
		}
	},
}

var ssmShowCmd = &cobra.Command{
	Use:   "show [profile_name]",
	Short: "show given profile(s) variables name",
	Run: func(cmd *cobra.Command, args []string) {
		for _, p := range args {
			vars, err := ssm.ShowProfile(p)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

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

func init() {
	ssmCmd.AddCommand(ssmAddCmd)
	ssmCmd.AddCommand(ssmListCmd)
	ssmCmd.AddCommand(ssmRemoveCmd)
	ssmCmd.AddCommand(ssmShowCmd)
	RootCmd.AddCommand(ssmCmd)
}
