package cmd

import (
	"github.com/kujtimiihoxha/kit/generator"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// dockerCmd represents the docker command
var dockerCmd = &cobra.Command{
	Use:     "docker",
	Aliases: []string{"d"},
	Short:   "Generate docker files",
	Run: func(cmd *cobra.Command, args []string) {
		g := generator.NewGenerateDocker(viper.GetBool("g_d_glide"))
		if err := g.Generate(); err != nil {
			logrus.Error(err)
		}
	},
}

func init() {
	generateCmd.AddCommand(dockerCmd)
	dockerCmd.Flags().Bool("glide", false, "Generate docker for project that uses glide package manager")
	viper.BindPFlag("g_d_glide", dockerCmd.Flags().Lookup("glide"))

}
