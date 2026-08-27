package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "current profile status",
	Run: func(cmd *cobra.Command, args []string) {
		p := os.Getenv("profile_name")

		if p == "" {
			fmt.Println("No profile currently in use")
		} else {
			fmt.Println(p)
		}
	},
}

func init() {
	RootCmd.AddCommand(statusCmd)
}
