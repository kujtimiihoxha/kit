package generator

import (
	"path"

	"github.com/kujtimiihoxha/gk-cli/parser"
	"github.com/kujtimiihoxha/gk-cli/utils"
	"github.com/spf13/viper"
)

func setDefaults() {
	viper.SetDefault("gk_service_path_format", path.Join("%s", "pkg", "service"))
	viper.SetDefault("gk_cmd_path_format", path.Join("%s", "cmd", "service"))
	viper.SetDefault("gk_endpoint_path_format", path.Join("%s", "pkg", "endpoint"))
	viper.SetDefault("gk_http_path_format", path.Join("%s", "pkg", "http"))
	viper.SetDefault("gk_http_client_path_format", path.Join("%s", "client", "http"))
	viper.SetDefault("gk_client_cmd_path_format", path.Join("%s", "cmd", "client"))
	viper.SetDefault("gk_service_file_name", "service.go")
	viper.SetDefault("gk_service_middleware_file_name", "middleware.go")
	viper.SetDefault("gk_endpoint_file_name", "endpoint.go")
	viper.SetDefault("gk_endpoint_middleware_file_name", "middleware.go")
	viper.SetDefault("gk_http_file_name", "handler.go")
	viper.SetDefault("gk_service_struct_prefix", "basic")
	viper.Set("gk_testing", true)

}
func createTestMethod(name string, param []parser.NamedTypeValue, result []parser.NamedTypeValue) parser.Method {
	param = append(param, parser.NewNameType("ctx", "context.Context"))
	return parser.Method{
		Name:       name,
		Parameters: param,
		Results:    result,
	}
}
func getTestServiceInterface(name string) parser.Interface {
	n := utils.ToCamelCase(name + "_Service")
	return parser.NewInterface(n, []parser.Method{
		createTestMethod(
			"Foo",
			[]parser.NamedTypeValue{
				parser.NewNameType("s","string"),
			},
			[]parser.NamedTypeValue{
				parser.NewNameType("r","string"),
				parser.NewNameType("err","error"),
			},
		),
	})
}
