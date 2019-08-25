package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-services/annotation"
	"github.com/go-services/code"
	"github.com/go-services/source"
	"github.com/ozgio/strutil"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"io/ioutil"
	"kit/config"
	"kit/fs"
	"kit/template"
	"os"
	"path"
)

type Method struct {
	Code       code.InterfaceMethod
	Transports []MethodTransport
	Request    *code.Struct
	Response   *code.Struct
}

type Service struct {
	Name       string
	Package    string
	Interface  *source.Interface
	StubStruct *source.Structure
	Config     config.KitConfig
	Methods    []Method
	File       source.Source
	serviceFs  afero.Fs
}

func Read(name string) (*Service, error) {
	rootFs := fs.AppFs()

	configData, err := fs.ReadFile("kit.json", rootFs)
	if err != nil {
		return nil, errors.New("not in a kit project, you need to be in a kit project to run this command")
	}
	var kitConfig config.KitConfig
	err = json.NewDecoder(bytes.NewBufferString(configData)).Decode(&kitConfig)
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
	serviceInterface, err := findServiceInterface(serviceFile)
	if err != nil {
		return nil, err
	}
	serviceMethods, err := findServiceMethods(&serviceInterface, name, kitConfig.Module, serviceFile.Package())
	if err != nil {
		return nil, err
	}
	svc := &Service{
		Name:       name,
		Package:    serviceFile.Package(),
		StubStruct: findServiceStub(serviceFile),
		Interface:  &serviceInterface,
		Config:     kitConfig,
		Methods:    serviceMethods,
		File:       *serviceFile,
		serviceFs:  serviceFs,
	}
	return svc, nil
}

func (s Service) Generate() error {
	err := s.serviceFs.RemoveAll("gen")
	if err != nil {
		return err
	}
	err = fs.CreateFolder("gen", s.serviceFs)
	if err != nil {
		return err
	}
	genFs := afero.NewBasePathFs(s.serviceFs, "gen")
	err = s.generateGen(genFs)
	if err != nil {
		return err
	}
	err = s.generateEndpoints(genFs)
	if err != nil {
		return err
	}
	err = s.generateTransports(genFs)
	if err != nil {
		return err
	}
	return s.generateCmd(genFs)
}

func (s Service) generateGen(genFs afero.Fs) error {
	genCode, err := template.CompileGoFromPath(
		"/assets/templates/service/gen/gen.go.gotmpl",
		s,
	)
	if err != nil {
		return err
	}
	err = fs.WriteFile("gen.go", genCode, genFs)
	if err != nil {
		return err
	}
	return nil
}

func (s Service) generateEndpoints(genFs afero.Fs) error {
	err := fs.CreateFolder("endpoint", genFs)
	if err != nil {
		return err
	}
	genFs = afero.NewBasePathFs(genFs, "endpoint")

	allEndpointsCode, err := template.CompileGoFromPath(
		"/assets/templates/service/gen/endpoint/endpoint.go.gotmpl",
		s,
	)
	if err != nil {
		return err
	}
	err = fs.WriteFile("endpoint.go", allEndpointsCode, genFs)
	for _, method := range s.Methods {
		err := method.generateEndpoint(s, genFs)
		if err != nil {
			return err
		}
	}
	return err
}
func (s Service) generateTransports(genFs afero.Fs) error {
	err := fs.CreateFolder("transport", genFs)
	if err != nil {
		return err
	}

	genFs = afero.NewBasePathFs(genFs, "transport")

	transportsCode, err := template.CompileGoFromPath(
		"/assets/templates/service/gen/transport/transport.go.gotmpl",
		s,
	)
	if err != nil {
		return err
	}
	err = fs.WriteFile("transports.go", transportsCode, genFs)
	if err != nil {
		return err
	}

	err = fs.CreateFolder("http", genFs)
	if err != nil {
		return err
	}

	httpFs := afero.NewBasePathFs(genFs, "http")

	allHttpCode, err := template.CompileGoFromPath(
		"/assets/templates/service/gen/transport/http/http.go.gotmpl",
		s,
	)
	if err != nil {
		return err
	}

	err = fs.WriteFile("http.go", allHttpCode, httpFs)
	if err != nil {
		return err
	}
	for _, method := range s.Methods {
		err := method.generateTransports(s, genFs)
		if err != nil {
			return err
		}
	}
	return nil
}
func (s Service) generateCmd(genFs afero.Fs) error {
	err := fs.CreateFolder("cmd", genFs)
	if err != nil {
		return err
	}

	genFs = afero.NewBasePathFs(genFs, "cmd")

	genCode, err := template.CompileGoFromPath(
		"/assets/templates/service/gen/cmd/cmd.go.gotmpl",
		s,
	)
	if err != nil {
		return err
	}
	return fs.WriteFile("cmd.go", genCode, genFs)
}

func (s Service) hasMethodImplementation(method Method, stubName string) bool {
	for _, fn := range s.File.Functions() {
		if fn.Receiver() != nil && fn.Receiver().Type.Qualifier == stubName && fn.Name() == method.Code.Name {
			return true
		}
	}
	return false
}

func (s Service) hasNewMethod() bool {
	for _, fn := range s.File.Functions() {
		if fn.Name() == "New" && fn.Receiver() == nil {
			return true
		}
	}
	return false
}

func (m Method) generateEndpoint(svc Service, endpointFs afero.Fs) error {
	methodEndpointCode, err := template.CompileGoFromPath(
		"/assets/templates/service/gen/endpoint/method.go.gotmpl",
		map[string]interface{}{
			"Service": svc,
			"Method":  m,
		},
	)
	if err != nil {
		return err
	}
	return fs.WriteFile(
		fmt.Sprintf("%s.go", strutil.ToSnakeCase(m.Code.Name)),
		methodEndpointCode,
		endpointFs,
	)
}
func (m Method) generateTransports(svc Service, transportsFs afero.Fs) error {
	for _, transport := range m.Transports {
		err := transport.Generate(svc, m, transportsFs)
		if err != nil {
			return err
		}
	}
	return nil
}

func findServiceInterface(src *source.Source) (source.Interface, error) {
	for _, inf := range src.Interfaces() {
		for _, ann := range inf.Annotations() {
			if ann.Name == "service" && inf.Exported() {
				return inf, nil
			}
		}
	}
	return source.Interface{}, errors.New(
		"could not find the service interface, make sure you add @service() to your interface and export the interface",
	)
}

func findServiceStub(src *source.Source) *source.Structure {
	for _, strc := range src.Structures() {
		for _, ann := range strc.Annotations() {
			if ann.Name == "stub" {
				return &strc
			}
		}
	}
	return nil
}
func findServiceMethods(inf *source.Interface, serviceName, module, pkg string) (methods []Method, err error) {
	for mthInx, mth := range inf.Code().(*code.Interface).Methods {
		hasCorrectParams, err := HasCorrectParams(mth.Params)
		if err != nil {
			return nil, err
		}
		hasCorrectResults, err := HasCorrectResults(mth.Results)
		if err != nil {
			return nil, err
		}
		if !IsExported(mth.Name) ||
			!HasCorrectParamNumber(mth.Params) ||
			!HasCorrectParamNumber(mth.Results) ||
			!hasCorrectParams ||
			!hasCorrectResults {
			continue
		}
		for paramInx, param := range mth.Params {
			inf.Code().(*code.Interface).Methods[mthInx].Params[paramInx] = fixMethodImport(param, serviceName, module, pkg)
		}
		for paramInx, param := range mth.Results {
			inf.Code().(*code.Interface).Methods[mthInx].Results[paramInx] = fixMethodImport(param, serviceName, module, pkg)
		}
		var reqStruct *code.Struct
		if len(mth.Params) == 2 {
			reqStruct, err = findStruct(mth.Params[1])
			if err != nil {
				return nil, err
			}
		}
		var respStruct *code.Struct
		if len(mth.Results) == 2 {
			respStruct, err = findStruct(mth.Results[0])
			if err != nil {
				return nil, err
			}
		}
		mth.Params = fixParameterParams(mth.Params)
		methods = append(methods, Method{
			Code:       mth,
			Transports: findMethodTransports(inf.Methods()[mthInx]),
			Request:    reqStruct,
			Response:   respStruct,
		})
	}
	return
}

func findMethodTransports(method source.InterfaceMethod) (transports []MethodTransport) {
	var httpAnnotations []annotation.Annotation
	for _, ann := range method.Annotations() {
		if ann.Name == "http" {
			httpAnnotations = append(httpAnnotations, ann)
		}
	}
	if httpAnnotations != nil {
		transports = append(transports, NewHTTPTransport(httpAnnotations))
	}
	return
}
func fixMethodImport(param code.Parameter, serviceName, module, pkg string) code.Parameter {
	if param.Type.Import == nil && IsExported(param.Type.Qualifier) {
		currentPath, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		if viper.GetString("testPath") != "" {
			currentPath = path.Join(currentPath, viper.GetString("testPath"))
		}
		param.Type.Import = code.NewImportWithFilePath(
			"service",
			fmt.Sprintf("%s/%s", module, pkg),
			path.Join(currentPath, serviceName),
		)
	}
	return param
}

func findStruct(param code.Parameter) (*code.Struct, error) {
	notFoundErr := errors.New(
		"could not find structure, make sure that you are using a structure as request/response parameters",
	)
	if param.Type.Import.FilePath == "" {
		return nil, notFoundErr
	}
	fls, err := ioutil.ReadDir(param.Type.Import.FilePath)
	if err != nil {
		panic(err)
	}
	if fls == nil {
		return nil, notFoundErr
	}
	for _, file := range fls {
		if file.IsDir() {
			continue
		}
		data, err := ioutil.ReadFile(path.Join(param.Type.Import.FilePath, file.Name()))
		if err != nil {
			panic(err)
		}
		dataFile, err := source.New(string(data))
		if err != nil {
			panic(err)
		}
		var strc *code.Struct
		for _, structure := range dataFile.Structures() {
			if structure.Name() == param.Type.Qualifier {
				strc = structure.Code().(*code.Struct)
				break
			}
		}
		if strc != nil {
			return strc, nil
		}
	}
	return nil, notFoundErr
}
func fixParameterParams(params []code.Parameter) []code.Parameter {
	if len(params) == 1 && params[0].Name == "" {
		params[0].Name = "ctx"
	} else if len(params) == 2 {
		if params[0].Name == "" {
			params[0].Name = "ctx"
		}
		if params[1].Name == "" {
			params[1].Name = "request"
		}
	}
	return params
}
