package cmd

import (
	"github.com/kujtimiihoxha/kit/generator"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serviceCmd = &cobra.Command{
	Use:     "service",
	Short:   "Generate new service",
	Aliases: []string{"s"},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			logrus.Error("You must provide a name for the service")
			return
		}
		g := generator.NewNewService(args[0])
		if err := g.Generate(); err != nil {
			logrus.Error(err)
		}
	},
}

func init() {
	newCmd.AddCommand(serviceCmd)
	serviceCmd.Flags().StringP("module", "m", "", "The module name that you plan to set in the project")
	viper.BindPFlag("n_s_module", serviceCmd.Flags().Lookup("module"))
}
