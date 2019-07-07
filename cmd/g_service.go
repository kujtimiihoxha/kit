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

var methods []string
var initserviceCmd = &cobra.Command{
	Use:     "service",
	Short:   "Initiate a service",
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

		if modPath == "" && !strings.HasPrefix(pwd, gosrc) {
			logrus.Error("The project must be in the $GOPATH/src folder for the generator to work.")
			return
		}

		if len(args) == 0 {
			logrus.Error("You must provide a name for the service")
			return
		}
		if viper.GetString("g_s_transport") == "grpc" {
			if !checkProtoc() {
				return
			}
		}
		var emw, smw bool
		if viper.GetBool("g_s_dmw") {
			emw = true
			smw = true
		} else {
			emw = viper.GetBool("g_s_endpoint_mdw")
			smw = viper.GetBool("g_s_svc_mdw")
		}
		g := generator.NewGenerateService(
			args[0],
			viper.GetString("g_s_transport"),
			smw,
			viper.GetBool("g_s_gorilla"),
			emw,
			methods,
		)
		if err := g.Generate(); err != nil {
			logrus.Error(err)
		}
	},
}

func init() {
	generateCmd.AddCommand(initserviceCmd)
	initserviceCmd.Flags().StringP("transport", "t", "http", "The transport you want your service to be initiated with")
	initserviceCmd.Flags().BoolP("dmw", "w", false, "Generate default middleware for service and endpoint")
	initserviceCmd.Flags().Bool("gorilla", false, "Generate http using gorilla mux")
	initserviceCmd.Flags().StringArrayVarP(&methods, "methods", "m", []string{}, "Specify methods to be generated")
	initserviceCmd.Flags().Bool("svc-mdw", false, "If set a default Logging and Instrumental middleware will be created and attached to the service")
	initserviceCmd.Flags().Bool("endpoint-mdw", false, "If set a default Logging and Tracking middleware will be created and attached to the endpoint")
	viper.BindPFlag("g_s_transport", initserviceCmd.Flags().Lookup("transport"))
	viper.BindPFlag("g_s_dmw", initserviceCmd.Flags().Lookup("dmw"))
	viper.BindPFlag("g_s_gorilla", initserviceCmd.Flags().Lookup("gorilla"))
	viper.BindPFlag("g_s_svc_mdw", initserviceCmd.Flags().Lookup("svc-mdw"))
	viper.BindPFlag("g_s_endpoint_mdw", initserviceCmd.Flags().Lookup("endpoint-mdw"))
}
