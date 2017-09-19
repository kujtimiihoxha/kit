package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var RootCmd = &cobra.Command{
	Use:   "gk",
	Short: "Go-Kit CLI",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().BoolP("debug", "d", false, "If you want to se the debug logs.")
	RootCmd.PersistentFlags().BoolP("force", "f", false, "Force overide existing files without asking.")
	RootCmd.PersistentFlags().StringP("folder", "b", "", "If you want to specify the base folder of the project.")
	viper.BindPFlag("gk_folder", RootCmd.PersistentFlags().Lookup("folder"))
	viper.BindPFlag("gk_force", RootCmd.PersistentFlags().Lookup("force"))
	viper.BindPFlag("gk_debug", RootCmd.PersistentFlags().Lookup("debug"))
}
