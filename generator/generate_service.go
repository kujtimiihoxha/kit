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

	err = generateEndpoints(
		serviceInterface,
		afero.NewBasePathFs(appFs, "gen"),
		kitConfig.Module,
		serviceSource.Package(),
	)
	if err != nil {
		return err
	}
	//err = generateTransports(
	//	serviceInterface,
	//	afero.NewBasePathFs(appFs, "generated"),
	//	goModule.Name,
	//	serviceSource.Package(),
	//)
	//if err != nil {
	//	return err
	//}
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

//
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
		methodEndpointCode, err :=  template.CompileGoFromPath(
			"/assets/templates/service/gen/endpoint/method.go.gotmpl",
			map[string]interface{}{
				"ServiceInterface": svcInterface,
				"ServiceMethod": &mth,
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

//
//func generateTransports(svcInterface *source.Interface, generatedFs afero.Fs, moduleName, packageName string) error {
//	err := fs.CreateFolder("transport", generatedFs)
//	if err != nil {
//		return err
//	}
//
//	generatedFs = afero.NewBasePathFs(generatedFs, "transport")
//
//	transportsCode, err := templates.TransportsFile(moduleName, packageName)
//	if err != nil {
//		return err
//	}
//	err = fs.CreateFile("transports.go", transportsCode, generatedFs)
//	if err != nil {
//		return err
//	}
//
//	err = fs.CreateFolder("http", generatedFs)
//	if err != nil {
//		return err
//	}
//
//	generatedFs = afero.NewBasePathFs(generatedFs, "http")
//
//	allHttpCode, err := templates.AllHttpFile(svcInterface, moduleName, packageName)
//	if err != nil {
//		return err
//	}
//
//	err = fs.CreateFile("http.go", allHttpCode, generatedFs)
//	if err != nil {
//		return err
//	}
//
//	for _, mth := range svcInterface.Methods() {
//		methodHttpCode, err := templates.MethodHttpFile(mth, moduleName, packageName)
//		if err != nil {
//			return err
//		}
//		err = fs.CreateFile(
//			fmt.Sprintf("%s.go", strutil.ToSnakeCase(mth.Name())),
//			methodHttpCode,
//			generatedFs,
//		)
//		if err != nil {
//			return err
//		}
//	}
//	return nil
//}
