package cmd

import (
	"github.com/kujtimiihoxha/kit/generator"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// middlewareCmd represents the middleware command
var middlewareCmd = &cobra.Command{
	Use:     "middleware",
	Aliases: []string{"m", "mdw"},
	Short:   "Generate middleware",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			logrus.Error("You must provide a name for the middleware")
			return
		}
		sn := viper.GetString("g_m_service")
		if sn == "" {
			logrus.Error("You must provide the name of the service")
			return
		}
		g := generator.NewGenerateMiddleware(
			args[0],
			sn,
			viper.GetBool("g_m_endpoint"),
		)
		if err := g.Generate(); err != nil {
			logrus.Error(err)
		}
		if viper.GetBool("g_m_endpoint") {
			logrus.Info("Do not forget to append your endpoint middleware to your service middlewares")
			logrus.Info("Add it to cmd/service/service.go#getEndpointMiddleware()")

		} else {

			logrus.Info("Do not forget to append your service middleware to your service middlewares")
			logrus.Info("Add it to cmd/service/service.go#getServiceMiddleware()")
		}
	},
}

func init() {
	generateCmd.AddCommand(middlewareCmd)
	middlewareCmd.Flags().StringP("service", "s", "",
		"Service name that the middleware will be created for")
	viper.BindPFlag("g_m_service", middlewareCmd.Flags().Lookup("service"))
	middlewareCmd.Flags().BoolP("endpoint", "e", false,
		"If set create endpoint middleware")
	viper.BindPFlag("g_m_endpoint", middlewareCmd.Flags().Lookup("endpoint"))
}
