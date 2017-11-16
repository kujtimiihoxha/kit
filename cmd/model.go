package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/kujtimiihoxha/kit/generator"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// dockerCmd represents the docker command
var modelCmd = &cobra.Command{
	Use:     "model",
	Aliases: []string{"mdl"},
	Short:   "Generate model",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			logrus.Error("You must provide a name for the model")
			return
		}
		if viper.GetString("g_md_service") == "" {
			logrus.Error("You must provide the service name")
			return
		}
		g := generator.NewGenerateModel(viper.GetString("g_md_service"), args[0])
		if err := g.Generate(); err != nil {
			logrus.Error(err)
		}
	},
}

func init() {
	generateCmd.AddCommand(modelCmd)
	modelCmd.Flags().StringP("service", "s", "",
		"Service name that the middleware will be created for")
	viper.BindPFlag("g_md_service", modelCmd.Flags().Lookup("service"))
}
