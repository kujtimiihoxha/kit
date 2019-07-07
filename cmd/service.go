package cmd

import (
	"github.com/kujtimiihoxha/kit/generator"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strings"
	"github.com/kujtimiihoxha/kit/utils"
	"github.com/spf13/afero"
)

var serviceCmd = &cobra.Command{
	Use:     "service",
	Short:   "Generate new service",
	Aliases: []string{"s"},
	Run: func(cmd *cobra.Command, args []string) {
		gosrc := strings.TrimSuffix(utils.GetGOPATH(), afero.FilePathSeparator ) + afero.FilePathSeparator + "src" + afero.FilePathSeparator
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
		modPath = viper.GetString("g_s_mod_module")
		if modPath != "" && strings.HasPrefix(pwd, gosrc) {
			logrus.Error("The project in the $GOPATH/src folder for the generator to work do not need to set --mod_module.")
			return
		}

		if modPath != "" && !strings.HasPrefix(pwd, gosrc) {  //modPath is complete project path, such as github.com/groupame/projectname; modPath is a projectname directory
			modPath = strings.Replace(modPath, "\\", "/", -1)
			modPathArr := strings.Split(modPath, "/")
			pwdArr := strings.Split(pwd, "/")
			if len(pwdArr) < len(modPathArr) {
				logrus.Error("The mod_module path invalid, and your mod_module path must be under the " + pwd)
				return
			}
			j := len(pwdArr)
			if len(modPathArr) > 1 { //only consider complete project path
				for i := len(modPathArr) - 2 ; i >=0 && j > 0 ; i-- {
					if modPathArr[i] != pwdArr[j - 1] {
						logrus.Error("The mod_module path invalid, and your mod_module path must be under the " + pwd)
						return
					}
					j --
				}
			}
		}

		if modPath == "" && !strings.HasPrefix(pwd, gosrc) {
			logrus.Error("The project must be in the $GOPATH/src folder for the generator to work, or generate project with —mod_module flag")
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
}
