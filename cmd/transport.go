package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/Sirupsen/logrus"
	"github.com/kujtimiihoxha/kit/generator"
)

var transportCmd = &cobra.Command{
	Use:   "transport",
	Short: "Add a new transport to the service",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			logrus.Error("You must specify the transport type")
			return
		}
		sn := viper.GetString("service")
		if sn == "" {
			logrus.Error("You must provide the name of the service")
			return
		}
		g := generator.NewGenerateTransport(
			sn,
			args[0],
			methods,
		)
		if err := g.Generate(); err != nil {
			logrus.Error(err)
		}
	},
}

func init() {
	addCmd.AddCommand(transportCmd)
	transportCmd.Flags().StringP("service", "s", "",
		"Service name that the transport will be created for")
	transportCmd.Flags().StringArrayVarP(&methods, "methods", "m", []string{}, "Specify methods to be generated")
	viper.BindPFlag("service", transportCmd.Flags().Lookup("service"))
}
