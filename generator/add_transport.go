package generator

import (
	"fmt"
	"path"
	"strings"

	"bytes"

	"github.com/Sirupsen/logrus"
	"github.com/dave/jennifer/jen"
	"github.com/emicklei/proto"
	"github.com/kujtimiihoxha/kit/fs"
	"github.com/kujtimiihoxha/kit/parser"
	"github.com/kujtimiihoxha/kit/utils"
	"github.com/spf13/viper"
)

type GenerateTransport struct {
	BaseGenerator
	name              string
	transport         string
	interfaceName     string
	serviceStructName string
	destPath          string
	methods           []string
	filePath          string
	file              *parser.File
	serviceInterface  parser.Interface
}

func NewGenerateTransport(name string, transport string, methods []string) Gen {
	i := &GenerateTransport{
		name:          name,
		interfaceName: utils.ToCamelCase(name + "Service"),
		destPath:      fmt.Sprintf(viper.GetString("gk_service_path_format"), utils.ToLowerSnakeCase(name)),
		methods:       methods,
	}
	i.filePath = path.Join(i.destPath, viper.GetString("gk_service_file_name"))
	i.serviceStructName = utils.ToLowerFirstCamelCase(viper.GetString("gk_service_struct_prefix") + "-" + i.interfaceName)
	i.transport = transport
	// Not used.
	i.srcFile = jen.NewFilePath("")
	i.InitPg()
	//
	i.fs = fs.Get()
	return i
}

func (g *GenerateTransport) Generate() (err error) {
	for n, v := range SUPPORTED_TRANSPORTS {
		if v == g.transport {
			break
		} else if n == len(SUPPORTED_TRANSPORTS)-1 {
			logrus.Errorf("Transport `%s` not supported", g.transport)
			return
		}
	}
	if b, err := g.fs.Exists(g.filePath); err != nil {
		return err
	} else {
		if !b {
			logrus.Errorf("Service %s was not found", g.name)
			return nil
		}
	}
	svcSrc, err := g.fs.ReadFile(g.filePath)
	if err != nil {
		return err
	}
	g.file, err = parser.NewFileParser().Parse([]byte(svcSrc))
	if !g.serviceFound() {
		return
	}
	g.removeBadMethods()
	if len(g.serviceInterface.Methods) == 0 {
		logrus.Error("The service has no suitable methods please implement the interface methods")
		return
	}
	switch g.transport {
	case "http":
		tbG := newGenerateHttpTransportBase(g.name, g.serviceInterface, g.methods)
		err = tbG.Generate()
		if err != nil {
			return err
		}
		tG := newGenerateHttpTransport(g.name, g.serviceInterface, g.methods)
		err = tG.Generate()
		if err != nil {
			return err
		}
	case "grpc":
		gp := newGenerateGRPCTransportProto(g.name, g.serviceInterface, g.methods)
		err = gp.Generate()
		if err != nil {
			return err
		}
	default:
		logrus.Warn("This transport type is not yet implemented")
	}
	return
}
func (g *GenerateTransport) serviceFound() bool {
	for n, v := range g.file.Interfaces {
		if v.Name == g.interfaceName {
			g.serviceInterface = v
			return true
		} else if n == len(g.file.Interfaces)-1 {
			logrus.Errorf("Could not find the service interface in `%s`", g.name)
			return false
		}
	}
	return false
}
func (g *GenerateTransport) removeBadMethods() {
	keepMethods := []parser.Method{}
	for _, v := range g.serviceInterface.Methods {
		if len(g.methods) > 0 {
			notFound := true
			for _, m := range g.methods {
				if v.Name == m {
					notFound = false
					break
				}
			}
			if notFound {
				continue
			}
		}
		if string(v.Name[0]) == strings.ToLower(string(v.Name[0])) {
			logrus.Warnf("The method '%s' is private and will be ignored", v.Name)
			continue
		}
		if len(v.Results) == 0 {
			logrus.Warnf("The method '%s' does not have any return value and will be ignored", v.Name)
			continue
		}
		for n, p := range v.Parameters {
			if p.Type == "context.Context" {
				keepMethods = append(keepMethods, v)
				break
			} else if n == len(v.Parameters)-1 {
				logrus.Warnf("The method '%s' does not have a context and will be ignored", v.Name)
				continue
			}
		}

	}
	g.serviceInterface.Methods = keepMethods
}

type generateHttpTransport struct {
	BaseGenerator
	name              string
	methods           []string
	interfaceName     string
	destPath          string
	generateFirstTime bool
	file              *parser.File
	filePath          string
	serviceInterface  parser.Interface
}

func newGenerateHttpTransport(name string, serviceInterface parser.Interface, methods []string) Gen {
	t := &generateHttpTransport{
		name:             name,
		methods:          methods,
		interfaceName:    utils.ToCamelCase(name + "Service"),
		destPath:         fmt.Sprintf(viper.GetString("gk_http_path_format"), utils.ToLowerSnakeCase(name)),
		serviceInterface: serviceInterface,
	}
	t.filePath = path.Join(t.destPath, viper.GetString("gk_http_file_name"))
	t.srcFile = jen.NewFilePath(t.destPath)
	t.InitPg()
	t.fs = fs.Get()
	return t
}
func (g *generateHttpTransport) Generate() (err error) {
	err = g.CreateFolderStructure(g.destPath)
	if err != nil {
		return err
	}
	endpImports, err := utils.GetEndpointImportPath(g.name)
	if err != nil {
		return err
	}
	if b, err := g.fs.Exists(g.filePath); err != nil {
		return err
	} else {
		if !b {
			g.generateFirstTime = true
			f := jen.NewFile("http")
			g.fs.WriteFile(g.filePath, f.GoString(), false)
		}
	}
	src, err := g.fs.ReadFile(g.filePath)
	if err != nil {
		return err
	}
	g.file, err = parser.NewFileParser().Parse([]byte(src))
	if err != nil {
		return err
	}
	hasError := false
	errorEncoderFound := false
	err2codeFound := false
	errorDecoderFound := false
	errorWrapperFound := false
	for _, m := range g.file.Structures {
		if m.Name == "errorWrapper" {
			errorWrapperFound = true
		}
	}
	for _, m := range g.serviceInterface.Methods {
		for _, v := range m.Results {
			if v.Type == "error" {
				hasError = true
			}
		}
		decoderFound := false
		encoderFound := false
		handlerFound := false
		for _, v := range g.file.Methods {
			if v.Name == "ErrorEncoder" {
				errorEncoderFound = true
			}
			if v.Name == "err2code" {
				err2codeFound = true
			}
			if v.Name == "ErrorDecoder" {
				errorDecoderFound = true
			}
			if v.Name == fmt.Sprintf("decode%sRequest", m.Name) {
				decoderFound = true
			}
			if v.Name == fmt.Sprintf("encode%sResponse", m.Name) {
				encoderFound = true
			}
			if v.Name == fmt.Sprintf("make%sHandler", m.Name) {
				handlerFound = true
			}
		}
		if !handlerFound {
			g.code.appendMultilineComment([]string{
				fmt.Sprintf("make%sHandler creates the handler logic", m.Name),
			})
			g.code.NewLine()
			g.code.appendFunction(
				fmt.Sprintf("make%sHandler", m.Name),
				nil,
				[]jen.Code{
					jen.Id("m").Id("*").Qual("net/http", "ServeMux"),
					jen.Id("endpoints").Qual(endpImports, "Endpoints"),
					jen.Id("options").Index().Qual(
						"github.com/go-kit/kit/transport/http",
						"ServerOption",
					),
				},
				[]jen.Code{},
				"",
				jen.Id("m").Dot("Handle").Call(
					jen.Lit("/"+strings.Replace(utils.ToLowerSnakeCase(m.Name), "_", "-", -1)),
					jen.Qual("github.com/go-kit/kit/transport/http", "NewServer").Call(
						jen.Id(fmt.Sprintf("endpoints.%sEndpoint", m.Name)),
						jen.Id(fmt.Sprintf("decode%sRequest", m.Name)),
						jen.Id(fmt.Sprintf("encode%sResponse", m.Name)),
						jen.Id("options..."),
					),
				),
			)
			g.code.NewLine()

		}

		if !decoderFound {
			g.code.appendMultilineComment([]string{
				fmt.Sprintf("decode%sResponse  is a transport/http.DecodeRequestFunc that decodes a", m.Name),
				"JSON-encoded sum request from the HTTP request body. Primarily useful in a server.",
			})
			g.code.NewLine()
			g.code.appendFunction(
				fmt.Sprintf("decode%sRequest", m.Name),
				nil,
				[]jen.Code{
					jen.Id("_").Qual("context", "Context"),
					jen.Id("r").Id("*").Qual("net/http", "Request"),
				},
				[]jen.Code{
					jen.Interface(),
					jen.Error(),
				},
				"",
				jen.Id("req").Op(":=").Qual(endpImports, m.Name+"Request").Block(),
				jen.Err().Op(":=").Qual("encoding/json", "NewDecoder").Call(
					jen.Id("r").Dot("Body"),
				).Dot("Decode").Call(jen.Id("&req")),
				jen.Return(jen.Id("req"), jen.Id("err")),
			)
			g.code.NewLine()
		}
		if !encoderFound {
			g.code.appendMultilineComment([]string{
				fmt.Sprintf("encode%sResponse is a transport/http.EncodeResponseFunc that encodes", m.Name),
				"the response as JSON to the response writer",
			})
			g.code.NewLine()
			g.code.appendFunction(
				fmt.Sprintf("encode%sResponse", m.Name),
				nil,
				[]jen.Code{
					jen.Id("ctx").Qual("context", "Context"),
					jen.Id("w").Qual("net/http", "ResponseWriter"),
					jen.Id("response").Interface(),
				},
				[]jen.Code{
					jen.Id("err").Error(),
				},
				"",
				jen.If(
					jen.List(jen.Id("f"), jen.Id("ok")).Op(":=").Id("response.").Call(
						jen.Qual(
							endpImports,
							"Failure",
						),
					).Id(";").Id("ok").Id("&&").Id("f").Dot("Failed").Call().Op("!=").Nil(),
				).Block(
					jen.Id("ErrorEncoder").Call(
						jen.Id("ctx"),
						jen.Id("f").Dot("Failed").Call(),
						jen.Id("w"),
					),
					jen.Return(jen.Nil()),
				),
				jen.Id("w").Dot("Header").Call().Dot("Set").Call(
					jen.Lit("Content-Type"), jen.Lit("application/json; charset=utf-8")),
				jen.Err().Op("=").Qual("encoding/json", "NewEncoder").Call(
					jen.Id("w"),
				).Dot("Encode").Call(jen.Id("response")),
				jen.Return(),
			)
			g.code.NewLine()
		}
	}
	if hasError {
		if !errorEncoderFound {
			g.code.appendFunction(
				"ErrorEncoder",
				nil,
				[]jen.Code{
					jen.Id("_").Qual("context", "Context"),
					jen.Id("err").Id("error"),
					jen.Id("w").Qual("net/http", "ResponseWriter"),
				},
				[]jen.Code{},
				"",
				jen.Id("w").Dot("WriteHeader").Call(jen.Id("err2code").Call(jen.Err())),
				jen.Qual("encoding/json", "NewEncoder").Call(jen.Id("w")).Dot("Encode").Call(
					jen.Id("errorWrapper").Values(
						jen.Dict{
							jen.Id("Error"): jen.Err().Dot("Error").Call(),
						},
					),
				),
			)
			g.code.NewLine()
		}
		if !errorDecoderFound {
			g.code.appendFunction(
				"ErrorDecoder",
				nil,
				[]jen.Code{
					jen.Id("r").Id("*").Qual("net/http", "Response"),
				},
				[]jen.Code{},
				"error",
				jen.Var().Id("w").Id("errorWrapper"),
				jen.If(
					jen.Err().Op(":=").Qual("encoding/json", "NewDecoder").Call(
						jen.Id("r").Dot("Body"),
					).Dot("Decode").Call(jen.Id("&w")).Id(";").Err().Op("!=").Nil(),
				).Block(
					jen.Return(jen.Err()),
				),
				jen.Return(jen.Qual("errors", "New").Call(jen.Id("w").Dot("Error"))),
			)
			g.code.NewLine()

		}
		if !err2codeFound {
			g.code.appendMultilineComment(
				[]string{
					"This is used to set the http status, see an example here :",
					"https://github.com/go-kit/kit/blob/master/examples/addsvc/pkg/addtransport/http.go#L133",
				},
			)
			g.code.NewLine()
			g.code.appendFunction(
				"err2code",
				nil,
				[]jen.Code{
					jen.Err().Error(),
				},
				[]jen.Code{},
				"int",
				jen.Return(jen.Qual("net/http", "StatusInternalServerError")),
			)
			g.code.NewLine()
		}
		if !errorWrapperFound {
			g.code.Raw().Type().Id("errorWrapper").Struct(
				jen.Id("Error").String().Tag(
					map[string]string{
						"json": "error",
					},
				),
			)
			g.code.NewLine()
		}
	}
	if g.generateFirstTime {
		return g.fs.WriteFile(g.filePath, g.srcFile.GoString(), true)
	}
	tmpSrc := g.srcFile.GoString()
	src += "\n" + g.code.Raw().GoString()
	f, err := parser.NewFileParser().Parse([]byte(tmpSrc))
	if err != nil {
		return err
	}
	// See if we need to add any new import
	imp, err := g.getMissingImports(f.Imports, g.file)
	if err != nil {
		return err
	}
	if len(imp) > 0 {
		src, err = g.AddImportsToFile(imp, src)
		if err != nil {
			return err
		}
	}
	s, err := utils.GoImportsSource(g.destPath, src)
	if err != nil {
		return err
	}
	return g.fs.WriteFile(g.filePath, s, true)
}

type generateHttpTransportBase struct {
	BaseGenerator
	name             string
	methods          []string
	interfaceName    string
	destPath         string
	filePath         string
	serviceInterface parser.Interface
}

func newGenerateHttpTransportBase(name string, serviceInterface parser.Interface, methods []string) Gen {
	t := &generateHttpTransportBase{
		name:             name,
		methods:          methods,
		interfaceName:    utils.ToCamelCase(name + "Service"),
		destPath:         fmt.Sprintf(viper.GetString("gk_http_path_format"), utils.ToLowerSnakeCase(name)),
		serviceInterface: serviceInterface,
	}
	t.filePath = path.Join(t.destPath, viper.GetString("gk_http_base_file_name"))
	t.srcFile = jen.NewFilePath(t.destPath)
	t.InitPg()
	t.fs = fs.Get()
	return t
}
func (g *generateHttpTransportBase) Generate() (err error) {
	err = g.CreateFolderStructure(g.destPath)
	if err != nil {
		return err
	}
	g.srcFile.PackageComment("THIS FILE IS AUTO GENERATED BY GK-CLI DO NOT EDIT!!")
	endpointImport, err := utils.GetEndpointImportPath(g.name)
	if err != nil {
		return err
	}
	g.code.appendMultilineComment([]string{
		" NewHTTPHandler returns a handler that makes a set of endpoints available on",
		"predefined paths.",
	})
	g.code.NewLine()
	handles := []jen.Code{}
	for _, m := range g.serviceInterface.Methods {
		handles = append(
			handles,
			jen.Id("make"+m.Name+"Handler").Call(
				jen.Id("m"),
				jen.Id("endpoints"),
				jen.Id("options").Index(jen.Lit(m.Name)),
			),
		)
	}
	body := append([]jen.Code{
		jen.Id("m").Op(":=").Qual("net/http", "NewServeMux").Call()}, handles...)
	body = append(body, jen.Return(jen.Id("m")))
	g.code.appendFunction(
		"NewHTTPHandler",
		nil,
		[]jen.Code{
			jen.Id("endpoints").Qual(endpointImport, "Endpoints"),
			jen.Id("options").Map(jen.String()).Index().Qual("github.com/go-kit/kit/transport/http", "ServerOption"),
		},
		[]jen.Code{},
		"http1.Handler",
		body...,
	)
	g.code.NewLine()
	return g.fs.WriteFile(g.filePath, g.srcFile.GoString(), true)
}

type generateGRPCTransportProto struct {
	BaseGenerator
	name              string
	methods           []string
	interfaceName     string
	generateFirstTime bool
	destPath          string
	protoSrc          *proto.Proto
	pbFilePath        string
	compileFilePath   string
	serviceInterface  parser.Interface
}

func newGenerateGRPCTransportProto(name string, serviceInterface parser.Interface, methods []string) Gen {
	t := &generateGRPCTransportProto{
		name:             name,
		methods:          methods,
		interfaceName:    utils.ToCamelCase(name + "Service"),
		destPath:         fmt.Sprintf(viper.GetString("gk_grpc_pb_path_format"), utils.ToLowerSnakeCase(name)),
		serviceInterface: serviceInterface,
	}
	t.pbFilePath = path.Join(
		t.destPath,
		fmt.Sprintf(viper.GetString("gk_grpc_pb_file_name"), utils.ToLowerSnakeCase(name)),
	)
	t.compileFilePath = path.Join(t.destPath, viper.GetString("gk_http_base_file_name"))
	t.fs = fs.Get()
	return t
}
func (g *generateGRPCTransportProto) Generate() (err error) {
	g.CreateFolderStructure(g.destPath)
	if b, err := g.fs.Exists(g.pbFilePath); err != nil {
		return err
	} else {
		if !b {
			g.generateFirstTime = true
			g.protoSrc = &proto.Proto{}
		} else {
			src, err := g.fs.ReadFile(g.pbFilePath)
			if err != nil {
				return err
			}
			r := bytes.NewReader([]byte(src))
			parser := proto.NewParser(r)
			definition, err := parser.Parse()
			g.protoSrc = definition
			if err != nil {
				return err
			}
		}
	}
	svc := &proto.Service{
		Comment: &proto.Comment{
			Lines: []string{
				fmt.Sprintf("The %s service definition.", utils.ToCamelCase(g.name)),
			},
		},
		Name: utils.ToCamelCase(g.name),
	}
	if g.generateFirstTime {
		g.getServiceRpc(svc)
		g.protoSrc.Elements = append(
			g.protoSrc.Elements,
			&proto.Syntax{
				Value: "proto3",
			},
			&proto.Package{
				Name: "pb",
			},
			svc,
		)
	} else {
		s := g.getService()
		if s == nil {
			s = svc
			g.protoSrc.Elements = append(g.protoSrc.Elements, s)
		}
		g.getServiceRpc(s)
	}
	g.generateRequestResponse()
	buf := new(bytes.Buffer)
	formatter := proto.NewFormatter(buf, " ")
	formatter.Format(g.protoSrc)
	return g.fs.WriteFile(g.pbFilePath, buf.String(), true)
}
func (g *generateGRPCTransportProto) getService() *proto.Service {
	for i, e := range g.protoSrc.Elements {
		if r, ok := e.(*proto.Service); ok {
			if r.Name == utils.ToCamelCase(g.name) {
				return g.protoSrc.Elements[i].(*proto.Service)
			}
		}
	}
	return nil
}
func (g *generateGRPCTransportProto) generateRequestResponse() {
	for _, v := range g.serviceInterface.Methods {
		foundRequest := false
		foundReply := false
		for _, e := range g.protoSrc.Elements {
			if r, ok := e.(*proto.Message); ok {
				if r.Name == v.Name+"Request" {
					foundRequest = true
				}
				if r.Name == v.Name+"Reply" {
					foundReply = true
				}
			}
		}
		if !foundRequest {
			g.protoSrc.Elements = append(g.protoSrc.Elements, &proto.Message{
				Name: v.Name + "Request",
			})
		}
		if !foundReply {
			g.protoSrc.Elements = append(g.protoSrc.Elements, &proto.Message{
				Name: v.Name + "Reply",
			})
		}
	}
}
func (g *generateGRPCTransportProto) getServiceRpc(svc *proto.Service) {
	for _, v := range g.serviceInterface.Methods {
		found := false
		for _, e := range svc.Elements {
			if r, ok := e.(*proto.RPC); ok {
				if r.Name == v.Name {
					found = true
				}
			}
		}
		if found {
			continue
		}
		svc.Elements = append(svc.Elements,
			&proto.RPC{
				Name:        v.Name,
				ReturnsType: v.Name + "Reply",
				RequestType: v.Name + "Request",
			},
		)
	}
}

type generateGRPCTransportBase struct {
	BaseGenerator
	name             string
	methods          []string
	interfaceName    string
	destPath         string
	filePath         string
	serviceInterface parser.Interface
}

func newGenerateGRPCTransportBase(name string, serviceInterface parser.Interface, methods []string) Gen {
	t := &generateGRPCTransportBase{
		name:             name,
		methods:          methods,
		interfaceName:    utils.ToCamelCase(name + "Service"),
		destPath:         fmt.Sprintf(viper.GetString("gk_http_path_format"), utils.ToLowerSnakeCase(name)),
		serviceInterface: serviceInterface,
	}
	t.filePath = path.Join(t.destPath, viper.GetString("gk_http_base_file_name"))
	t.srcFile = jen.NewFilePath(t.destPath)
	t.InitPg()
	t.fs = fs.Get()
	return t
}
func (g *generateGRPCTransportBase) Generate() (err error) {
	return
}
