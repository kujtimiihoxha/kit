package generator

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-services/source"
	"github.com/ozgio/strutil"
	"github.com/spf13/afero"
	"kit/fs"
	"kit/template"
	"strings"
)

func GenerateService(name string) error {
	appFs := fs.AppFs()
	folderName := strings.ReplaceAll(strutil.ToSnakeCase(name), "_", "")

	b, err := afero.Exists(appFs, folderName)

	if err != nil {
		return err
	} else if !b {
		return errors.New("service folder does not exist")
	}

	b, err = afero.Exists(appFs, "kit.json")

	if err != nil {
		return err
	} else if !b {
		return errors.New("not in a kit project, you need to be in a project to run this command")
	}

	// go module details
	kitConf, err := afero.ReadFile(appFs, "kit.json")
	if err != nil {
		return err
	}
	var kitConfig KitConfig
	err = json.NewDecoder(bytes.NewBuffer(kitConf)).Decode(&kitConfig)

	appFs = afero.NewBasePathFs(appFs, folderName)
	data, err := afero.ReadFile(appFs, "service.go")
	if err != nil {
		return err
	}

	serviceSource, err := source.New(string(data))
	if err != nil {
		return err
	}
	var serviceInterface *source.Interface
	for _, inf := range serviceSource.Interfaces() {
		if isServiceInterface(inf) {
			serviceInterface = &inf
			break
		}
	}
	if serviceInterface == nil {
		return errors.New("service interface not found")
	}

	// clean generated folder
	err = appFs.RemoveAll("gen")
	if err != nil {
		return err
	}
	err = fs.CreateFolder("gen", appFs)
	if err != nil {
		return err
	}
	genFs := afero.NewBasePathFs(appFs, "gen")
	err = generateEndpoints(
		serviceInterface,
		genFs,
		kitConfig.Module,
		serviceSource.Package(),
	)
	if err != nil {
		return err
	}
	err = generateTransports(
		serviceInterface,
		genFs,
		kitConfig.Module,
		serviceSource.Package(),
	)
	if err != nil {
		return err
	}
	err = generateGen(
		serviceInterface,
		genFs,
		kitConfig.Module,
		serviceSource.Package(),
	)
	if err != nil {
		return err
	}
	err = generateCmd(
		genFs,
		kitConfig.Module,
		serviceSource.Package(),
	)
	if err != nil {
		return err
	}
	return nil
}
func isServiceInterface(inf source.Interface) bool {
	for _, annotation := range inf.Annotations() {
		if annotation.Name == "service" && inf.Exported() {
			return true
		}
	}
	return false
}

func generateCmd(generatedFs afero.Fs, moduleName, packageName string) error {
	err := fs.CreateFolder("cmd", generatedFs)
	if err != nil {
		return err
	}

	generatedFs = afero.NewBasePathFs(generatedFs, "cmd")

	genCode, err := template.CompileGoFromPath(
		"/assets/templates/service/gen/cmd/cmd.go.gotmpl",
		map[string]interface{}{
			"ProjectModule":  moduleName,
			"ServicePackage": packageName,
		},
	)
	if err != nil {
		return err
	}
	err = fs.CreateFile("cmd.go", genCode, generatedFs)
	if err != nil {
		return err
	}
	return nil
}
func generateGen(svcInterface *source.Interface, generatedFs afero.Fs, moduleName, packageName string) error {
	genCode, err := template.CompileGoFromPath(
		"/assets/templates/service/gen/gen.go.gotmpl",
		map[string]interface{}{
			"ServiceInterface": svcInterface,
			"ProjectModule":    moduleName,
			"ServicePackage":   packageName,
		},
	)
	if err != nil {
		return err
	}
	err = fs.CreateFile("gen.go", genCode, generatedFs)
	if err != nil {
		return err
	}
	return nil
}
func generateEndpoints(svcInterface *source.Interface, generatedFs afero.Fs, moduleName, packageName string) error {
	err := fs.CreateFolder("endpoint", generatedFs)
	if err != nil {
		return err
	}

	generatedFs = afero.NewBasePathFs(generatedFs, "endpoint")

	allEndpointsCode, err := template.CompileGoFromPath(
		"/assets/templates/service/gen/endpoint/endpoint.go.gotmpl",
		map[string]interface{}{
			"ServiceInterface": svcInterface,
			"ProjectModule":    moduleName,
			"ServicePackage":   packageName,
		},
	)
	if err != nil {
		return err
	}
	err = fs.CreateFile("endpoint.go", allEndpointsCode, generatedFs)
	if err != nil {
		return err
	}

	for _, mth := range svcInterface.Methods() {
		fixMethodTypes(&mth)
		for _, p := range mth.Params() {
			fmt.Println(p.Name, p.Type.Import.FilePath)
		}
		methodEndpointCode, err := template.CompileGoFromPath(
			"/assets/templates/service/gen/endpoint/method.go.gotmpl",
			map[string]interface{}{
				"ServiceInterface": svcInterface,
				"ServiceMethod":    &mth,
				"ProjectModule":    moduleName,
				"ServicePackage":   packageName,
			},
		)
		if err != nil {
			return err
		}
		err = fs.CreateFile(
			fmt.Sprintf("%s.go", strutil.ToSnakeCase(mth.Name())),
			methodEndpointCode,
			generatedFs,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func generateTransports(svcInterface *source.Interface, generatedFs afero.Fs, moduleName, packageName string) error {
	err := fs.CreateFolder("transport", generatedFs)
	if err != nil {
		return err
	}

	generatedFs = afero.NewBasePathFs(generatedFs, "transport")

	transportsCode, err := template.CompileGoFromPath(
		"/assets/templates/service/gen/transport/transport.go.gotmpl",
		map[string]interface{}{
			"ProjectModule":  moduleName,
			"ServicePackage": packageName,
		},
	)
	if err != nil {
		return err
	}
	err = fs.CreateFile("transports.go", transportsCode, generatedFs)
	if err != nil {
		return err
	}

	err = fs.CreateFolder("http", generatedFs)
	if err != nil {
		return err
	}

	generatedFs = afero.NewBasePathFs(generatedFs, "http")

	allHttpCode, err := template.CompileGoFromPath(
		"/assets/templates/service/gen/transport/http/http.go.gotmpl",
		map[string]interface{}{
			"ServiceInterface": svcInterface,
			"ProjectModule":    moduleName,
			"ServicePackage":   packageName,
		},
	)
	if err != nil {
		return err
	}

	err = fs.CreateFile("http.go", allHttpCode, generatedFs)
	if err != nil {
		return err
	}

	for _, mth := range svcInterface.Methods() {
		fixMethodTypes(&mth)
		methodHttpCode, err := template.CompileGoFromPath(
			"/assets/templates/service/gen/transport/http/method.go.gotmpl",
			map[string]interface{}{
				"ServiceInterface": svcInterface,
				"ServiceMethod":    &mth,
				"MethodRoutes":     findMethodRoutes(mth),
				"ProjectModule":    moduleName,
				"ServicePackage":   packageName,
			},
		)
		if err != nil {
			return err
		}
		err = fs.CreateFile(
			fmt.Sprintf("%s.go", strutil.ToSnakeCase(mth.Name())),
			methodHttpCode,
			generatedFs,
		)
		if err != nil {
			return err
		}
	}
	return nil
}
