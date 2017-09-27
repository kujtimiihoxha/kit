package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"fmt"

	"github.com/alioygur/godash"
	"github.com/dave/jennifer/jen"
	"github.com/spf13/viper"
	"golang.org/x/tools/imports"
)

func ToLowerFirstCamelCase(s string) string {
	if len(s) == 1 {
		return strings.ToLower(string(s[0]))
	}
	return strings.ToLower(string(s[0])) + godash.ToCamelCase(s)[1:]
}
func ToUpperFirst(s string) string {
	return strings.ToUpper(string(s[0])) + s[1:]
}
func ToLowerSnakeCase(s string) string {
	return strings.ToLower(godash.ToSnakeCase(s))
}

func ToCamelCase(s string) string {
	return godash.ToCamelCase(s)
}

func GoImportsSource(path string, s string) (string, error) {
	is, err := imports.Process(path, []byte(s), nil)
	return string(is), err
}

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

func GetHttpTransportImportPath(name string) (string, error) {
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

func GetGOPATH() string {
	if viper.GetString("GOPATH") != "" {
		return viper.GetString("GOPATH")
	}
	return defaultGOPATH()
}

func ToJenCodeArray(c jen.Code) []jen.Code {
	return []jen.Code{c}
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
