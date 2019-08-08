package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-services/code"
	"github.com/go-services/source"
	"github.com/spf13/afero"
	"kit/fs"
)

type Method struct {
	Code     *code.InterfaceMethod
	Request  *code.Struct
	Response *code.Struct
}

type Service struct {
	Interface *code.Interface
	Config    KitConfig
	Methods   []Method
	File      source.Source
	serviceFs afero.Fs
}

func Read(name string) (*Service, error) {
	rootFs := fs.AppFs()

	configData, err := fs.ReadFile("kit.json", rootFs)
	if err != nil {
		return nil, errors.New("not in a kit project, you need to be in a kit project to run this command")
	}
	var kitConfig KitConfig
	err = json.NewDecoder(bytes.NewBufferString(configData)).Decode(kitConfig)
	if err != nil {
		return nil, err
	}
	serviceFs := afero.NewBasePathFs(rootFs, name)
	serviceData, err := fs.ReadFile("service.go", serviceFs)
	if err != nil {
		return nil, fmt.Errorf("could not find service `%s`", name)
	}

	serviceFile, err := source.New(serviceData)
	if err != nil {
		return nil, err
	}

	serviceInterface := findServiceInterface(serviceFile)
	if serviceInterface == nil {
		return nil, errors.New("could not find the service interface, make sure you add @service() to your interface and export the interface")
	}

	return nil, nil
}

func findServiceInterface(src *source.Source) *code.Interface {
	for _, inf := range src.Interfaces() {
		for _, annotation := range inf.Annotations() {
			if annotation.Name == "service" && inf.Exported() {
				return inf.Code().(*code.Interface)
			}
		}
	}
	return nil
}

func (svc *Service) findServiceMethods() {
	for _, mth := range svc.Interface.Methods {
		if !IsExported(mth.Name) ||
			!HasCorrectParamNumber(mth.Params) ||
			!HasCorrectParamNumber(mth.Results) ||
			!HasCorrectParams(mth.Params) ||
			!HasCorrectResults(mth.Results) {
			continue
		}
		for _, param := range mth.Params {
			svc.fixMethodImport(&param)
		}
		for _, param := range mth.Results {
			svc.fixMethodImport(&param)
		}
	}
}
func (svc *Service) fixMethodImport(param *code.Parameter) {
	if param.Type.Import == nil && IsExported(param.Type.Qualifier) {
		param.Type.Import.Path = fmt.Sprintf("%s/%s", svc.Config.Module, svc.File.Package())
	}
}
