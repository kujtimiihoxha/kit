package main

import (
	"path"
	"runtime"

	"github.com/kujtimiihoxha/kit/cmd"
	"github.com/spf13/viper"
)

func main() {
	setDefaults()
	viper.AutomaticEnv()
	cmd.Execute()
}

func setDefaults() {
	viper.SetDefault("gk_service_path_format", path.Join("%s", "pkg", "service"))
	viper.SetDefault("gk_cmd_service_path_format", path.Join("%s", "cmd", "service"))
	viper.SetDefault("gk_cmd_path_format", path.Join("%s", "cmd"))
	viper.SetDefault("gk_endpoint_path_format", path.Join("%s", "pkg", "endpoint"))
	viper.SetDefault("gk_http_path_format", path.Join("%s", "pkg", "http"))
	viper.SetDefault("gk_http_client_path_format", path.Join("%s", "client", "http"))
	viper.SetDefault("gk_grpc_client_path_format", path.Join("%s", "client", "grpc"))
	viper.SetDefault("gk_client_cmd_path_format", path.Join("%s", "cmd", "client"))
	viper.SetDefault("gk_grpc_path_format", path.Join("%s", "pkg", "grpc"))
	viper.SetDefault("gk_grpc_pb_path_format", path.Join("%s", "pkg", "grpc", "pb"))

	viper.SetDefault("gk_service_file_name", "service.go")
	viper.SetDefault("gk_service_middleware_file_name", "middleware.go")
	viper.SetDefault("gk_endpoint_base_file_name", "endpoint_gen.go")
	viper.SetDefault("gk_endpoint_file_name", "endpoint.go")
	viper.SetDefault("gk_endpoint_middleware_file_name", "middleware.go")
	viper.SetDefault("gk_http_file_name", "handler.go")
	viper.SetDefault("gk_http_base_file_name", "handler_gen.go")
	viper.SetDefault("gk_cmd_base_file_name", "service_gen.go")
	viper.SetDefault("gk_cmd_svc_file_name", "service.go")
	viper.SetDefault("gk_http_client_file_name", "http.go")
	viper.SetDefault("gk_grpc_client_file_name", "grpc.go")
	viper.SetDefault("gk_grpc_pb_file_name", "%s.proto")
	viper.SetDefault("gk_grpc_base_file_name", "handler_gen.go")
	viper.SetDefault("gk_grpc_file_name", "handler.go")
	if runtime.GOOS == "windows" {
		viper.SetDefault("gk_grpc_compile_file_name", "compile.bat")
	} else {
		viper.SetDefault("gk_grpc_compile_file_name", "compile.sh")
	}
	viper.SetDefault("gk_service_struct_prefix", "basic")

}
