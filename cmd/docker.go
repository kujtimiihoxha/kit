package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/kujtimiihoxha/kit/generator"
	"github.com/spf13/cobra"
)

// dockerCmd represents the docker command
var dockerCmd = &cobra.Command{
	Use:     "docker",
	Aliases: []string{"d"},
	Short:   "Generate docker files",
	Run: func(cmd *cobra.Command, args []string) {
		g := generator.NewGenerateDocker()
		if err := g.Generate(); err != nil {
			logrus.Error(err)
		}
	},
}

func init() {
	generateCmd.AddCommand(dockerCmd)
}
