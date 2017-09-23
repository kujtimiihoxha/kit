package generator

import (
	"fmt"
	"path"

	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/dave/jennifer/jen"
	"github.com/kujtimiihoxha/kit/fs"
	"github.com/kujtimiihoxha/kit/parser"
	"github.com/kujtimiihoxha/kit/utils"
	"github.com/spf13/viper"
)

type GenerateClient struct {
	BaseGenerator
	name             string
	transport        string
	interfaceName    string
	destPath         string
	filePath         string
	serviceDestPath  string
	serviceFilePath  string
	serviceFile      *parser.File
	serviceInterface parser.Interface
}

func NewGenerateClient(name string, transport string) Gen {
	i := &GenerateClient{
		name:            name,
		interfaceName:   utils.ToCamelCase(name + "Service"),
		destPath:        fmt.Sprintf(viper.GetString("gk_client_cmd_path_format"), utils.ToLowerSnakeCase(name)),
		serviceDestPath: fmt.Sprintf(viper.GetString("gk_service_path_format"), utils.ToLowerSnakeCase(name)),
		transport:       transport,
	}
	i.serviceFilePath = path.Join(i.serviceDestPath, viper.GetString("gk_service_file_name"))
	i.filePath = path.Join(i.destPath, viper.GetString("gk_service_file_name"))
	i.srcFile = jen.NewFilePath(i.destPath)
	i.InitPg()
	i.fs = fs.Get()
	return i
}
func (g *GenerateClient) Generate() (err error) {
	for n, v := range SUPPORTED_TRANSPORTS {
		if v == g.transport {
			break
		} else if n == len(SUPPORTED_TRANSPORTS)-1 {
			logrus.Errorf("Transport `%s` not supported", g.transport)
			return
		}
	}
	if b, err := g.fs.Exists(g.serviceFilePath); err != nil {
		return err
	} else {
		if !b {
			logrus.Errorf("Service %s was not found", g.name)
			return nil
		}
	}
	svcSrc, err := g.fs.ReadFile(g.serviceFilePath)
	if err != nil {
		return err
	}
	g.serviceFile, err = parser.NewFileParser().Parse([]byte(svcSrc))
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
		cg := newGenerateHttpClient(g.name, g.serviceInterface, g.serviceFile)
		err = cg.Generate()
		if err != nil {
			return err
		}
	default:
		logrus.Warn("This transport type is not yet implemented")
	}

	return
}
func (g *GenerateClient) serviceFound() bool {
	for n, v := range g.serviceFile.Interfaces {
		if v.Name == g.interfaceName {
			g.serviceInterface = v
			return true
		} else if n == len(g.serviceFile.Interfaces)-1 {
			logrus.Errorf("Could not find the service interface in `%s`", g.name)
			return false
		}
	}
	return false
}
func (g *GenerateClient) removeBadMethods() {
	keepMethods := []parser.Method{}
	for _, v := range g.serviceInterface.Methods {
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

type generateHttpClient struct {
	BaseGenerator
	name             string
	interfaceName    string
	destPath         string
	filePath         string
	serviceInterface parser.Interface
	serviceFile      *parser.File
}

func newGenerateHttpClient(name string, serviceInterface parser.Interface, serviceFile *parser.File) Gen {
	i := &generateHttpClient{
		name:             name,
		interfaceName:    utils.ToCamelCase(name + "Service"),
		destPath:         fmt.Sprintf(viper.GetString("gk_http_client_path_format"), utils.ToLowerSnakeCase(name)),
		serviceInterface: serviceInterface,
		serviceFile:      serviceFile,
	}
	i.filePath = path.Join(i.destPath, viper.GetString("gk_http_client_file_name"))
	i.srcFile = jen.NewFilePath(i.destPath)
	i.InitPg()
	i.fs = fs.Get()
	return i
}
func (g *generateHttpClient) Generate() (err error) {
	g.CreateFolderStructure(g.destPath)
	endpointImport, err := utils.GetEndpointImportPath(g.name)
	if err != nil {
		return err
	}
	serviceImport, err := utils.GetServiceImportPath(g.name)
	if err != nil {
		return err
	}
	g.code.appendMultilineComment([]string{
		"New returns an AddService backed by an HTTP server living at the remote",
		"instance. We expect instance to come from a service discovery system, so",
		"likely of the form \"host:port\".",
	})

	g.code.NewLine()
	handles := []jen.Code{}
	respS := jen.Dict{}
	for _, m := range g.serviceInterface.Methods {
		respS[jen.Id(m.Name+"Endpoint")] = jen.Id(utils.ToLowerFirstCamelCase(m.Name) + "Endpoint")
		handles = append(
			handles,
			jen.Var().Id(utils.ToLowerFirstCamelCase(m.Name)+"Endpoint").Qual(
				"github.com/go-kit/kit/endpoint",
				"Endpoint",
			).Line().Block(
				jen.Id(utils.ToLowerFirstCamelCase(m.Name)+"Endpoint").Op("=").Qual(
					"github.com/go-kit/kit/transport/http",
					"NewClient",
				).Call(
					jen.Lit("POST"),
					jen.Id("copyURL").Call(
						jen.Id("u"), jen.Lit(
							"/"+strings.Replace(utils.ToLowerSnakeCase(m.Name), "_", "-", -1),
						),
					),
					jen.Id("encodeHTTPGenericRequest"),
					jen.Id(fmt.Sprintf("decode%sResponse", m.Name)),
					jen.Id(fmt.Sprintf("options[\"%s\"]...", m.Name)),
				).Dot("Endpoint").Call(),
			).Line(),
		)
	}
	body := append([]jen.Code{
		jen.If(
			jen.Id("!").Qual("strings", "HasPrefix").Call(
				jen.Id("instance"),
				jen.Lit("http"),
			),
		).Block(
			jen.Id("instance").Op("=").Lit("http://").Op("+").Id("instance"),
		),
		jen.List(
			jen.Id("u"),
			jen.Id("err"),
		).Op(":=").Qual("net/url", "Parse").Call(jen.Id("instance")),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Nil(), jen.Err()),
		),
	},
		handles...,
	)
	body = append(
		body,
		jen.Return(
			jen.Qual(endpointImport, "Endpoints").Values(
				respS,
			),
			jen.Nil(),
		),
	)
	g.code.appendFunction(
		"New",
		nil,
		[]jen.Code{
			jen.Id("instance").String(),
			jen.Id("options").Map(jen.String()).Index().Qual("github.com/go-kit/kit/transport/http", "ClientOption"),
		},
		[]jen.Code{
			jen.Qual(serviceImport, g.serviceInterface.Name),
			jen.Error(),
		},
		"",
		body...,
	)
	err = g.generateDecodeEncodeMethods(endpointImport)
	if err != nil {
		return err
	}
	g.code.appendFunction(
		"copyURL",
		nil,
		[]jen.Code{
			jen.Id("base").Id("*").Qual("net/url", "URL"),
			jen.Id("path").Id("string"),
		},
		[]jen.Code{
			jen.Id("next").Id("*").Qual("net/url", "URL"),
		},
		"",
		jen.Id("n").Op(":=").Id("*base"),
		jen.Id("n").Dot("Path").Op("=").Id("path"),
		jen.Id("next").Op("=").Id("&n"),
		jen.Return(),
	)
	g.code.NewLine()
	return g.fs.WriteFile(g.filePath, g.srcFile.GoString(), false)
}
func (g *generateHttpClient) generateDecodeEncodeMethods(endpointImport string) (err error) {
	httpImport, err := utils.GetHttpTransportImportPath(g.name)
	if err != nil {
		return err
	}
	g.code.NewLine()
	g.code.appendMultilineComment([]string{
		"EncodeHTTPGenericRequest is a transport/http.EncodeRequestFunc that",
		"SON-encodes any request to the request body. Primarily useful in a client.",
	})
	g.code.NewLine()
	g.code.appendFunction(
		"encodeHTTPGenericRequest",
		nil,
		[]jen.Code{
			jen.Id("_").Qual("context", "Context"),
			jen.Id("r").Id("*").Qual("net/http", "Request"),
			jen.Id("request").Interface(),
		},
		[]jen.Code{},
		"error",
		jen.Var().Id("buf").Qual("bytes", "Buffer").Line(),
		jen.If(
			jen.Err().Op(":=").Qual("encoding/json", "NewEncoder").Call(
				jen.Id("&buf"),
			).Dot("Encode").Call(jen.Id("request")).Id(";").Err().Op("!=").Nil().Block(
				jen.Return(jen.Err()),
			),
		),
		jen.Id("r").Dot("Body").Op("=").Qual("io/ioutil", "NopCloser").Call(
			jen.Id("&buf"),
		),
		jen.Return(jen.Nil()),
	)
	g.code.NewLine()
	for _, m := range g.serviceInterface.Methods {
		g.code.appendMultilineComment([]string{
			fmt.Sprintf("decode%sResponse is a transport/http.DecodeResponseFunc that decodes", m.Name),
			"a JSON-encoded concat response from the HTTP response body. If the response",
			"as a non-200 status code, we will interpret that as an error and attempt to",
			" decode the specific error message from the response body.",
		})
		g.code.NewLine()
		g.code.appendFunction(
			fmt.Sprintf("decode%sResponse", m.Name),
			nil,
			[]jen.Code{
				jen.Id("_").Qual("context", "Context"),
				jen.Id("r").Id("*").Qual("net/http", "Response"),
			},
			[]jen.Code{
				jen.Interface(),
				jen.Error(),
			},
			"",
			jen.If(
				jen.Id("r").Dot("StatusCode").Op("!=").Qual("net/http", "StatusOK"),
			).Block(
				jen.Return(jen.Nil(), jen.Qual(httpImport, "ErrorDecoder").Call(jen.Id("r"))),
			),
			jen.Var().Id("resp").Qual(endpointImport, m.Name+"Response"),
			jen.Err().Op(":=").Qual("encoding/json", "NewDecoder").Call(
				jen.Id("r").Dot("Body"),
			).Dot("Decode").Call(jen.Id("&resp")),
			jen.Return(jen.Id("resp"), jen.Err()),
		)
		g.code.NewLine()
	}
	return
}
