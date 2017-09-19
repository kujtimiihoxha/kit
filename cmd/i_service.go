package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/kujtimiihoxha/gk-cli/generator"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var initserviceCmd = &cobra.Command{
	Use:     "service",
	Short:   "Initiate a service",
	Aliases: []string{"s"},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			logrus.Error("You must provide a name for the service")
			return
		}
		g := generator.NewInitService(
			args[0],
			viper.GetString("transport"),
			viper.GetBool("no-svc-middleware"),
			viper.GetBool("no-endpoint-middleware"),
		)
		if err := g.Generate(); err != nil {
			logrus.Error(err)
		}
	},
}

func init() {
	initCmd.AddCommand(initserviceCmd)
	initserviceCmd.Flags().StringP("transport", "t", "http", "The transport you want your service to be initiated with")
	initserviceCmd.Flags().Bool("no-svc-middleware", false, "If set no default service middleware will be created")
	initserviceCmd.Flags().Bool("no-endpoint-middleware", false, "If set no default endpoint middleware will be created")
	viper.BindPFlag("transport", initserviceCmd.Flags().Lookup("transport"))
	viper.BindPFlag("no-svc-middleware", initserviceCmd.Flags().Lookup("no-svc-middleware"))
	viper.BindPFlag("no-endpoint-middleware", initserviceCmd.Flags().Lookup("no-endpoint-middleware"))
}
