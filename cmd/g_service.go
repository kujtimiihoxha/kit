package cmd

import (
	"os/exec"
	"runtime"

	"github.com/Sirupsen/logrus"
	"github.com/kujtimiihoxha/kit/generator"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var methods []string
var initserviceCmd = &cobra.Command{
	Use:     "service",
	Short:   "Initiate a service",
	Aliases: []string{"s"},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			logrus.Error("You must provide a name for the service")
			return
		}
		if viper.GetString("transport") == "grpc" {
			p := exec.Command("protoc")
			if p.Run() != nil {
				logrus.Error("Please install protoc first and than rerun the command")
				if runtime.GOOS == "windows" {
					logrus.Info(
						"Install proto3.",
						"https://github.com/google/protobuf/releases",
						"Update protoc Go bindings via",
						"go get -u github.com/golang/protobuf/protoc-gen-go",
						"",
						"See also",
						"https://github.com/grpc/grpc-go/tree/master/examples",
					)
				}
				logrus.Info(
					"Install proto3 from source macOS only.",
					"brew install autoconf automake libtool",
					"git clone https://github.com/google/protobuf",
					" ./autogen.sh ; ./configure ; make ; make install",
					"",
					"Update protoc Go bindings via",
					"go get -u github.com/golang/protobuf/{proto,protoc-gen-go}",
					"",
					"See also",
					"https://github.com/grpc/grpc-go/tree/master/examples",
				)
			}
		}
		emw := false
		smw := false
		if viper.GetBool("dmw") {
			emw = true
			smw = true
		} else {
			emw = viper.GetBool("endpoint-mdw")
			smw = viper.GetBool("svc-mdw")
		}
		g := generator.NewGenerateService(
			args[0],
			viper.GetString("transport"),
			smw,
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
	initserviceCmd.Flags().Bool("dmw", false, "Generate default middleware for service and endpoint")
	initserviceCmd.Flags().StringArrayVarP(&methods, "methods", "m", []string{}, "Specify methods to be generated")
	initserviceCmd.Flags().Bool("svc-mdw", false, "If set a default Logging and Instrumental middleware will be created and attached to the service")
	initserviceCmd.Flags().Bool("endpoint-mdw", false, "If set a default Logging and Tracking middleware will be created and attached to the endpoint")
	viper.BindPFlag("transport", initserviceCmd.Flags().Lookup("transport"))
	viper.BindPFlag("dmw", initserviceCmd.Flags().Lookup("dmw"))
	viper.BindPFlag("svc-mdw", initserviceCmd.Flags().Lookup("svc-mdw"))
	viper.BindPFlag("endpoint-mdw", initserviceCmd.Flags().Lookup("endpoint-mdw"))
}
