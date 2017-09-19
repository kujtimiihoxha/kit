package cmd

import (
	"github.com/spf13/cobra"
)

var transportCmd = &cobra.Command{
	Use:   "transport",
	Short: "Add a new transport to the service",
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	addCmd.AddCommand(transportCmd)
}
