package cmd

import (
	"os"
	"os/exec"
	"runtime"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// RootCmd is the root command of kit
var RootCmd = &cobra.Command{
	Use:   "kit",
	Short: "Go-Kit CLI",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// Execute runs the root command
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().BoolP("debug", "d", false, "If you want to se the debug logs.")
	RootCmd.PersistentFlags().BoolP("force", "f", false, "Force overide existing files without asking.")
	RootCmd.PersistentFlags().StringP("folder", "b", "", "If you want to specify the base folder of the project.")
	RootCmd.PersistentFlags().String("mod_module", "", "The mod module path you plan to set in the project")

	viper.BindPFlag("gk_folder", RootCmd.PersistentFlags().Lookup("folder"))
	viper.BindPFlag("gk_force", RootCmd.PersistentFlags().Lookup("force"))
	viper.BindPFlag("gk_debug", RootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("g_s_mod_module", RootCmd.PersistentFlags().Lookup("mod_module"))
}

func checkProtoc() bool {
	p := exec.Command("protoc")
	if p.Run() != nil {
		logrus.Error("Please install protoc first and than rerun the command")
		if runtime.GOOS == "windows" {
			logrus.Info(
				`Install proto3.
https://github.com/google/protobuf/releases
Update protoc Go bindings via
> go get -u github.com/golang/protobuf/proto
> go get -u github.com/golang/protobuf/protoc-gen-go

See also
https://github.com/grpc/grpc-go/tree/master/examples`,
			)
		} else if runtime.GOOS == "darwin" {
			logrus.Info(
				`Install proto3 from source macOS only.
> brew install autoconf automake libtool
> git clone https://github.com/google/protobuf
> ./autogen.sh ; ./configure ; make ; make install

Update protoc Go bindings via
> go get -u github.com/golang/protobuf/{proto,protoc-gen-go}

See also
https://github.com/grpc/grpc-go/tree/master/examples`,
			)
		} else {
			logrus.Info(`Install proto3
sudo apt-get install -y git autoconf automake libtool curl make g++ unzip
git clone https://github.com/google/protobuf.git
cd protobuf/
./autogen.sh
./configure
make
make check
sudo make install
sudo ldconfig # refresh shared library cache.`)
		}
		return false
	}
	return true
}
