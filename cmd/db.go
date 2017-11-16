package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/kujtimiihoxha/kit/generator"
	"github.com/spf13/cobra"
)

// dockerCmd represents the docker command
var dbCmd = &cobra.Command{
	Use:     "db",
	Aliases: []string{},
	Short:   "Generate db",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			logrus.Error("You must provide a name for the service")
			return
		}
		g := generator.NewGenerateDB(args[0])
		if err := g.Generate(); err != nil {
			logrus.Error(err)
		}
	},
}

func init() {
	generateCmd.AddCommand(dbCmd)
}
