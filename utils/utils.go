package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"fmt"

	"github.com/alioygur/godash"
	"github.com/spf13/viper"
	"golang.org/x/tools/imports"
)

// ToLowerFirstCamelCase returns the given string in camelcase formatted string
// but with the first letter being lowercase.
func ToLowerFirstCamelCase(s string) string {
	if s == "" {
		return s
	}
	if len(s) == 1 {
		return strings.ToLower(string(s[0]))
	}
	return strings.ToLower(string(s[0])) + godash.ToCamelCase(s)[1:]
}

// ToUpperFirst returns the given string with the first letter being uppercase.
func ToUpperFirst(s string) string {
	if s == "" {
		return s
	}
	if len(s) == 1 {
		return strings.ToLower(string(s[0]))
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}

// ToLowerSnakeCase the given string in snake-case format.
func ToLowerSnakeCase(s string) string {
	return strings.ToLower(godash.ToSnakeCase(s))
}

// ToCamelCase the given string in camelcase format.
func ToCamelCase(s string) string {
	return godash.ToCamelCase(s)
}

// GoImportsSource is used to format and optimize imports the
// given source.
func GoImportsSource(path string, s string) (string, error) {
	is, err := imports.Process(path, []byte(s), nil)
	return string(is), err
}

// GetServiceImportPath returns the import path of the service interface.
func GetServiceImportPath(name string) (string, error) {
	gosrc := GetGOPATH() + "/src/"
	gosrc = strings.Replace(gosrc, "\\", "/", -1)
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if viper.GetString("gk_folder") != "" {
		pwd += "/" + viper.GetString("gk_folder")
	}
	pwd = strings.Replace(pwd, "\\", "/", -1)
	projectPath := strings.Replace(pwd, gosrc, "", 1)
	svcPath := fmt.Sprintf(viper.GetString("gk_service_path_format"), ToLowerSnakeCase(name))

	svcPath = strings.Replace(svcPath, "\\", "/", -1)
	serviceImport := projectPath + "/" + svcPath
	return serviceImport, nil
}

// GetCmdServiceImportPath returns the import path of the cmd service (used by cmd/main.go).
func GetCmdServiceImportPath(name string) (string, error) {
	gosrc := GetGOPATH() + "/src/"
	gosrc = strings.Replace(gosrc, "\\", "/", -1)
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if viper.GetString("gk_folder") != "" {
		pwd += "/" + viper.GetString("gk_folder")
	}
	pwd = strings.Replace(pwd, "\\", "/", -1)
	projectPath := strings.Replace(pwd, gosrc, "", 1)
	svcPath := fmt.Sprintf(viper.GetString("gk_cmd_service_path_format"), ToLowerSnakeCase(name))

	svcPath = strings.Replace(svcPath, "\\", "/", -1)
	serviceImport := projectPath + "/" + svcPath
	return serviceImport, nil
}

// GetEndpointImportPath returns the import path of the service endpoints.
func GetEndpointImportPath(name string) (string, error) {
	gosrc := GetGOPATH() + "/src/"
	gosrc = strings.Replace(gosrc, "\\", "/", -1)
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if viper.GetString("gk_folder") != "" {
		pwd += "/" + viper.GetString("gk_folder")
	}
	pwd = strings.Replace(pwd, "\\", "/", -1)
	projectPath := strings.Replace(pwd, gosrc, "", 1)
	epPath := fmt.Sprintf(viper.GetString("gk_endpoint_path_format"), ToLowerSnakeCase(name))

	epPath = strings.Replace(epPath, "\\", "/", -1)
	endpointImport := projectPath + "/" + epPath
	return endpointImport, nil
}

// GetGRPCTransportImportPath returns the import path of the service grpc transport.
func GetGRPCTransportImportPath(name string) (string, error) {
	gosrc := GetGOPATH() + "/src/"
	gosrc = strings.Replace(gosrc, "\\", "/", -1)
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if viper.GetString("gk_folder") != "" {
		pwd += "/" + viper.GetString("gk_folder")
	}
	pwd = strings.Replace(pwd, "\\", "/", -1)
	projectPath := strings.Replace(pwd, gosrc, "", 1)
	epPath := fmt.Sprintf(viper.GetString("gk_grpc_path_format"), ToLowerSnakeCase(name))

	epPath = strings.Replace(epPath, "\\", "/", -1)
	endpointImport := projectPath + "/" + epPath
	return endpointImport, nil
}

// GetPbImportPath returns the import path of the generated service grpc pb.
func GetPbImportPath(name string) (string, error) {
	gosrc := GetGOPATH() + "/src/"
	gosrc = strings.Replace(gosrc, "\\", "/", -1)
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if viper.GetString("gk_folder") != "" {
		pwd += "/" + viper.GetString("gk_folder")
	}
	pwd = strings.Replace(pwd, "\\", "/", -1)
	projectPath := strings.Replace(pwd, gosrc, "", 1)
	epPath := fmt.Sprintf(viper.GetString("gk_grpc_pb_path_format"), ToLowerSnakeCase(name))

	epPath = strings.Replace(epPath, "\\", "/", -1)
	endpointImport := projectPath + "/" + epPath
	return endpointImport, nil
}

// GetHTTPTransportImportPath returns the import path of the service http transport.
func GetHTTPTransportImportPath(name string) (string, error) {
	gosrc := GetGOPATH() + "/src/"
	gosrc = strings.Replace(gosrc, "\\", "/", -1)
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if viper.GetString("gk_folder") != "" {
		pwd += "/" + viper.GetString("gk_folder")
	}
	pwd = strings.Replace(pwd, "\\", "/", -1)
	projectPath := strings.Replace(pwd, gosrc, "", 1)
	epPath := fmt.Sprintf(viper.GetString("gk_http_path_format"), ToLowerSnakeCase(name))

	epPath = strings.Replace(epPath, "\\", "/", -1)
	httpImports := projectPath + "/" + epPath
	return httpImports, nil
}

// GetDockerFileProjectPath returns the path of the project.
func GetDockerFileProjectPath() (string, error) {
	gosrc := GetGOPATH() + "/src/"
	gosrc = strings.Replace(gosrc, "\\", "/", -1)
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if viper.GetString("gk_folder") != "" {
		pwd += "/" + viper.GetString("gk_folder")
	}
	pwd = strings.Replace(pwd, "\\", "/", -1)
	projectPath := strings.Replace(pwd, gosrc, "", 1)
	return projectPath, nil
}

// GetGOPATH returns the gopath.
func GetGOPATH() string {
	if viper.GetString("GOPATH") != "" {
		return viper.GetString("GOPATH")
	}
	return defaultGOPATH()
}

func defaultGOPATH() string {
	env := "HOME"
	if runtime.GOOS == "windows" {
		env = "USERPROFILE"
	} else if runtime.GOOS == "plan9" {
		env = "home"
	}
	if home := os.Getenv(env); home != "" {
		def := filepath.Join(home, "go")
		if filepath.Clean(def) == filepath.Clean(runtime.GOROOT()) {
			// Don't set the default GOPATH to GOROOT,
			// as that will trigger warnings from the go tool.
			return ""
		}
		return def
	}
	return ""
}

// GetServiceInterfaceName returns the service interface name
func GetServiceInterfaceName(name string) string {
	format := viper.GetString("gk_service_interface_name")
	if format == "" {
		format = "%sService"
	}
	return fmt.Sprintf(format, ToCamelCase(name))
}

// GetProtoServiceName returns the protobuf service name
func GetProtoServiceName(name string) string {
	format := viper.GetString("gk_proto_service_name")
	if format == "" {
		format = "%s"
	}
	return ToCamelCase(fmt.Sprintf(format, name))
}

// GetProtoPackageName returns the protobuf package name
func GetProtoPackageName() string {
	format := viper.GetString("gk_proto_package_name")
	if format == "" {
		return "pb"
	}
	return fmt.Sprintf(format, "pb")
}
