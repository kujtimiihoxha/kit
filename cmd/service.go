package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/kujtimiihoxha/kit/generator"
	"github.com/kujtimiihoxha/kit/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serviceCmd = &cobra.Command{
	Use:     "service",
	Short:   "Generate new service",
	Aliases: []string{"s"},
	Run: func(cmd *cobra.Command, args []string) {
		gosrc := strings.TrimSuffix(utils.GetGOPATH(), afero.FilePathSeparator) + afero.FilePathSeparator + "src" + afero.FilePathSeparator
		pwd, err := os.Getwd()
		if err != nil {
			logrus.Error(err)
			return
		}
		gosrc, err = filepath.EvalSymlinks(gosrc)
		if err != nil {
			logrus.Error(err)
			return
		}
		pwd, err = filepath.EvalSymlinks(pwd)
		if err != nil {
			logrus.Error(err)
			return
		}

		var modPath string
		modPath = viper.GetString("n_s_mod_module")

		if modPath == "" && !strings.HasPrefix(pwd, gosrc) {
			logrus.Error("The project must be in the $GOPATH/src folder for the generator to work, or generate project with --mod_module flag")
			return
		}

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
	serviceCmd.Flags().StringP("mod_module", "m", "", "The mod module name that you plan to set in the project")
	viper.BindPFlag("n_s_mod_module", serviceCmd.Flags().Lookup("mod_module"))
}
