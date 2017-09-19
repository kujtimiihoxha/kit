package generator

import (
	"fmt"
	"path"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/dave/jennifer/jen"
	"github.com/kujtimiihoxha/gk-cli/fs"
	"github.com/kujtimiihoxha/gk-cli/parser"
	"github.com/kujtimiihoxha/gk-cli/utils"
	"github.com/spf13/viper"
)

// SUPPORTED_TRANSPORTS is an array containing the supported transport types.
var SUPPORTED_TRANSPORTS = []string{"http", "grpc", "thrift"}

type InitService struct {
	BaseGenerator
	pg                       *PartialGenerator
	name                     string
	transport                string
	interfaceName            string
	serviceStructName        string
	destPath                 string
	filePath                 string
	file                     *parser.File
	serviceInterface         parser.Interface
	sMiddleware, eMiddleware bool
}

// NewInitService returns a initialized and ready generator.
//
// The name parameter is the name of the service that will be created
// this name should be without the `Service` suffix
//
// The sMiddleware specifies if the default service middleware should be
// created
//
// The eMiddleware specifies if the default endpoint middleware should be
// created
func NewInitService(name, transport string, sMiddleware, eMiddleware bool) Gen {
	i := &InitService{
		name:          name,
		interfaceName: utils.ToCamelCase(name + "Service"),
		destPath:      fmt.Sprintf(viper.GetString("gk_service_path_format"), utils.ToLowerSnakeCase(name)),
		sMiddleware:   sMiddleware,
		eMiddleware:   eMiddleware,
	}
	i.filePath = path.Join(i.destPath, viper.GetString("gk_service_file_name"))
	i.pg = NewPartialGenerator(nil)
	i.serviceStructName = utils.ToLowerFirstCamelCase(viper.GetString("gk_service_struct_prefix") + "-" + i.interfaceName)
	i.transport = transport
	// Not used.
	i.srcFile = jen.NewFilePath("")
	i.InitPg()
	//
	i.fs = fs.Get()
	return i
}
func (i *InitService) Generate() (err error) {
	for n, v := range SUPPORTED_TRANSPORTS {
		if v == i.transport {
			break
		} else if n == len(SUPPORTED_TRANSPORTS)-1 {
			logrus.Errorf("Transport `%s` not supported", i.transport)
			return
		}
	}
	if b, err := i.fs.Exists(i.filePath); err != nil {
		return err
	} else {
		if !b {
			logrus.Errorf("Service %s was not found", i.name)
			return nil
		}
	}
	svcSrc, err := i.fs.ReadFile(i.filePath)
	if err != nil {
		return err
	}
	i.file, err = parser.NewFileParser().Parse([]byte(svcSrc))
	if !i.serviceFound() {
		return
	}
	i.removeBadMethods()
	if len(i.serviceInterface.Methods) == 0 {
		logrus.Error("The service has no suitable methods please implement the interface methods")
		return
	}
	i.generateServiceStruct()
	i.generateServiceMethods()
	i.generateNewBasicStructMethod()
	i.generateNewMethod()
	svcSrc += "\n" + i.pg.String()
	s, err := utils.GoImportsSource(i.destPath, svcSrc)
	err = i.fs.WriteFile(i.filePath, s, true)
	if err != nil {
		return err
	}
	mdwG := newGenerateServiceMiddleware(i.name, i.file, i.serviceInterface, !i.sMiddleware)
	err = mdwG.Generate()
	if err != nil {
		return err
	}
	epG := newGenerateServiceEndpoints(i.name, i.file, i.serviceInterface, !i.eMiddleware)
	err = epG.Generate()
	if err != nil {
		return err
	}
	switch i.transport {
	case "http":
		tG := newGenerateHttpTransport(i.name, i.serviceInterface)
		err = tG.Generate()
		if err != nil {
			return err
		}
	default:
		logrus.Warn("This transport type is not yet implemented")
	}
	mG := newGenerateCmd(i.name, i.serviceInterface, !i.sMiddleware, !i.eMiddleware)
	return mG.Generate()
}
func (i *InitService) generateServiceMethods() {
	stp := ""
	methodParameterNames := []parser.NamedTypeValue{}
	for _, v := range i.serviceInterface.Methods {
		methodParameterNames = append(methodParameterNames, v.Parameters...)
		methodParameterNames = append(methodParameterNames, v.Results...)
	}
	stp = i.GenerateNameBySample(i.serviceStructName, methodParameterNames)
	for _, m := range i.serviceInterface.Methods {
		exists := false
		for _, v := range i.file.Methods {
			if v.Name == m.Name && v.Struct.Type == "*"+i.serviceStructName {
				logrus.Infof("Service method `%s` already exists so it will not be recreated.", v.Name)
				exists = true
				break
			}
		}
		if exists {
			continue
		}
		sp := []jen.Code{}
		for _, p := range m.Parameters {
			sp = append(sp, jen.Id(p.Name).Id(p.Type))
		}
		rs := []jen.Code{}
		rt := []jen.Code{}
		for _, p := range m.Results {
			rs = append(rs, jen.Id(p.Name).Id(p.Type))
			rt = append(rt, jen.Id(p.Name))
		}

		body := []jen.Code{
			jen.Comment("TODO implement the business logic of " + m.Name),
			jen.Return(rt...),
		}
		i.pg.appendFunction(
			m.Name,
			jen.Id(stp).Id("*"+i.serviceStructName),
			sp,
			rs,
			"",
			body...,
		)
		i.pg.NewLine()
	}
}
func (i *InitService) generateServiceStruct() {
	for _, v := range i.file.Structures {
		if v.Name == i.serviceStructName {
			logrus.Infof("Service `%s` structure already exists so it will not be recreated.", i.serviceStructName)
			return
		}
	}
	i.pg.appendStruct(i.serviceStructName)
}
func (i *InitService) generateNewMethod() {
	for _, v := range i.file.Methods {
		if v.Name == "New" {
			logrus.Infof("Service method `%s` already exists so it will not be recreated.", v.Name)
			return
		}
	}
	i.pg.Raw().Commentf(
		"New returns a %s with all of the expected middleware wired in.",
		i.interfaceName,
	).Line()
	fn := fmt.Sprintf("New%s", utils.ToCamelCase(i.serviceStructName))
	body := []jen.Code{
		jen.Var().Id("svc").Id(i.interfaceName).Op("=").Id(fn).Call(),
		jen.For(
			jen.List(jen.Id("_"), jen.Id("m")).Op(":=").Range().Id("middleware"),
		).Block(
			jen.Id("svc").Op("=").Id("m").Call(jen.Id("svc")),
		),
		jen.Return(jen.Id("svc")),
	}
	i.pg.appendFunction(
		"New",
		nil,
		[]jen.Code{
			jen.Id("middleware").Id("[]Middleware"),
		},
		[]jen.Code{},
		i.interfaceName,
		body...)
	i.pg.NewLine()
}
func (i *InitService) generateNewBasicStructMethod() {
	fn := fmt.Sprintf("New%s", utils.ToCamelCase(i.serviceStructName))
	for _, v := range i.file.Methods {
		if v.Name == fn {
			logrus.Infof("Service method `%s` already exists so it will not be recreated.", v.Name)
			return
		}
	}
	i.pg.Raw().Commentf(
		"New%s returns a naive, stateless implementation of %s.",
		utils.ToCamelCase(i.serviceStructName),
		i.interfaceName,
	).Line()
	body := []jen.Code{
		jen.Return(jen.Id(fmt.Sprintf("&%s{}", i.serviceStructName))),
	}
	i.pg.appendFunction(fn, nil, []jen.Code{}, []jen.Code{}, i.interfaceName, body...)
	i.pg.NewLine()
}
func (i *InitService) serviceFound() bool {
	for n, v := range i.file.Interfaces {
		if v.Name == i.interfaceName {
			i.serviceInterface = v
			return true
		} else if n == len(i.file.Interfaces)-1 {
			logrus.Errorf("Could not find the service interface in `%s`", i.name)
			return false
		}
	}
	return false
}
func (i *InitService) removeBadMethods() {
	keepMethods := []parser.Method{}
	for _, v := range i.serviceInterface.Methods {
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
	i.serviceInterface.Methods = keepMethods
}

type generateServiceMiddleware struct {
	BaseGenerator
	name             string
	interfaceName    string
	destPath         string
	filePath         string
	file             *parser.File
	serviceInterface parser.Interface
	generateDefaults bool
}

func newGenerateServiceMiddleware(name string, file *parser.File,
	serviceInterface parser.Interface, generateDefaults bool) Gen {
	gsm := &generateServiceMiddleware{
		name:             name,
		interfaceName:    utils.ToCamelCase(name + "Service"),
		destPath:         fmt.Sprintf(viper.GetString("gk_service_path_format"), utils.ToLowerSnakeCase(name)),
		serviceInterface: serviceInterface,
		file:             file,
	}
	gsm.filePath = path.Join(gsm.destPath, viper.GetString("gk_service_middleware_file_name"))
	gsm.generateDefaults = generateDefaults
	gsm.srcFile = jen.NewFilePath(gsm.destPath)
	gsm.InitPg()
	gsm.fs = fs.Get()
	return gsm
}
func (g *generateServiceMiddleware) Generate() error {
	g.CreateFolderStructure(g.destPath)
	g.code.Raw().Comment("Middleware describes a service middleware.").Line()
	g.code.Raw().Type().Id("Middleware").Func().Params(jen.Id(g.interfaceName)).Id(g.interfaceName).Line()
	g.code.NewLine()
	if g.generateDefaults {
		g.code.appendStruct(
			"loggingMiddleware",
			jen.Id("logger").Qual("github.com/go-kit/kit/log", "Logger"),
			jen.Id("next").Id(g.interfaceName),
		)
		g.code.appendMultilineComment([]string{
			"LoggingMiddleware takes a logger as a dependency",
			fmt.Sprintf("and returns a %s Middleware.", g.interfaceName),
		})
		g.code.NewLine()
		pt := NewPartialGenerator(nil)
		pt.appendFunction(
			"",
			nil,
			[]jen.Code{
				jen.Id("next").Id(g.interfaceName),
			},
			[]jen.Code{},
			g.interfaceName,
			jen.Return(jen.Id("&loggingMiddleware").Values(jen.Id("logger"), jen.Id("next"))),
		)
		pt.NewLine()
		g.code.appendFunction(
			"LoggingMiddleware",
			nil,
			[]jen.Code{
				jen.Id("logger").Id("log.Logger"),
			},
			[]jen.Code{},
			"Middleware",
			jen.Return(pt.Raw()),
		)
		g.code.NewLine()
		g.code.NewLine()
		g.generateMethodMiddleware()
	}
	return g.fs.WriteFile(g.filePath, g.srcFile.GoString(), false)
}

func (g *generateServiceMiddleware) generateMethodMiddleware() {
	stp := ""
	methodParameterNames := []parser.NamedTypeValue{}
	for _, v := range g.serviceInterface.Methods {
		methodParameterNames = append(methodParameterNames, v.Parameters...)
		methodParameterNames = append(methodParameterNames, v.Results...)
	}
	stp = g.GenerateNameBySample("loggingMiddleware", methodParameterNames)
	for _, m := range g.serviceInterface.Methods {
		middlewareFuncParam := []jen.Code{}
		middlewareFuncResult := []jen.Code{}
		loggerLog := []jen.Code{}
		middlewareReturn := []jen.Code{}
		for _, p := range m.Parameters {
			pth := g.EnsureThatWeUseQualifierIfNeeded(p.Type, g.file.Imports)
			if pth != "" {
				s := strings.Split(p.Type, ".")
				middlewareFuncParam = append(middlewareFuncParam, jen.Id(p.Name).Qual(pth, s[1]))
			} else {
				middlewareFuncParam = append(middlewareFuncParam, jen.Id(p.Name).Id(p.Type))
			}
			middlewareReturn = append(middlewareReturn, jen.Id(p.Name))
			if p.Type != "context.Context" {
				loggerLog = append(loggerLog, jen.Lit(p.Name), jen.Id(p.Name))
			}
		}
		for _, p := range m.Results {
			pth := g.EnsureThatWeUseQualifierIfNeeded(p.Type, g.file.Imports)
			if pth != "" {
				s := strings.Split(p.Type, ".")
				middlewareFuncResult = append(middlewareFuncResult, jen.Id(p.Name).Qual(pth, s[1]))
			} else {
				middlewareFuncResult = append(middlewareFuncResult, jen.Id(p.Name).Id(p.Type))
			}
			loggerLog = append(loggerLog, jen.Lit(p.Name), jen.Id(p.Name))
		}
		loggerLog = append([]jen.Code{jen.Lit("method"), jen.Lit(m.Name)}, loggerLog...)
		deferBlock := jen.Id(stp).Dot("logger").Dot("Log").Call(
			loggerLog...,
		)
		g.code.appendFunction(
			m.Name,
			jen.Id(stp).Id("loggingMiddleware"),
			middlewareFuncParam,
			middlewareFuncResult,
			"",
			jen.Defer().Func().Call().Block(deferBlock).Call(),
			jen.Return(jen.Id(stp).Dot("next").Dot(m.Name).Call(middlewareReturn...)),
		)
		g.code.NewLine()
	}
}

type generateServiceEndpoints struct {
	BaseGenerator
	name             string
	interfaceName    string
	destPath         string
	filePath         string
	file             *parser.File
	serviceInterface parser.Interface
	generateDefaults bool
}

func newGenerateServiceEndpoints(name string, file *parser.File,
	serviceInterface parser.Interface, generateDefaults bool) Gen {
	gsm := &generateServiceEndpoints{
		name:             name,
		interfaceName:    utils.ToCamelCase(name + "Service"),
		destPath:         fmt.Sprintf(viper.GetString("gk_endpoint_path_format"), utils.ToLowerSnakeCase(name)),
		serviceInterface: serviceInterface,
		file:             file,
	}
	gsm.filePath = path.Join(gsm.destPath, viper.GetString("gk_endpoint_file_name"))
	gsm.generateDefaults = generateDefaults
	gsm.srcFile = jen.NewFilePath(gsm.destPath)
	gsm.InitPg()
	gsm.fs = fs.Get()
	return gsm
}
func (e *generateServiceEndpoints) Generate() error {
	e.CreateFolderStructure(e.destPath)
	fields := []jen.Code{}
	for _, v := range e.serviceInterface.Methods {
		fields = append(fields, jen.Id(v.Name+"Endpoint").Qual("github.com/go-kit/kit/endpoint", "Endpoint"))
	}
	e.code.appendMultilineComment([]string{
		"Endpoints collects all of the endpoints that compose a profile service. It's",
		"meant to be used as a helper struct, to collect all of the endpoints into a",
		"single parameter.",
	})
	e.code.NewLine()
	e.code.appendStruct(
		"Endpoints",
		fields...,
	)
	err := e.generateNewMethod()
	if err != nil {
		return err
	}
	e.generateMethodEndpoint()
	e.generateEndpointsClientMethods()
	if e.generateDefaults {
		mdw := newGenerateEndpointMiddleware(e.name)
		err = mdw.Generate()
		if err != nil {
			return err
		}
	}
	return e.fs.WriteFile(e.filePath, e.srcFile.GoString(), false)
}

func (e *generateServiceEndpoints) generateEndpointsClientMethods() {
	stp := ""
	methodParameterNames := []parser.NamedTypeValue{}
	for _, v := range e.serviceInterface.Methods {
		methodParameterNames = append(methodParameterNames, v.Parameters...)
		methodParameterNames = append(methodParameterNames, v.Results...)
	}
	stp = e.GenerateNameBySample("Endpoints", methodParameterNames)
	for _, m := range e.serviceInterface.Methods {
		req := jen.Dict{}
		resList := []jen.Code{}
		sp := []jen.Code{}
		for _, p := range m.Parameters {
			tp := p.Type
			ts := strings.Split(tp, ".")
			if len(ts) == 1 {
				if tp[:1] == strings.ToUpper(tp[:1]) {
					// If the type of the parameter is not `something.MyType` and it starts with an uppercase
					// than the type was defined inside the service package.
					tp = "service." + tp
				}
			}
			pth := e.EnsureThatWeUseQualifierIfNeeded(p.Type, e.file.Imports)
			if pth != "" {
				s := strings.Split(p.Type, ".")
				sp = append(sp, jen.Id(p.Name).Qual(pth, s[1]))
			} else {
				sp = append(sp, jen.Id(p.Name).Id(tp))
			}
			if p.Type != "context.Context" {
				req[jen.Id(utils.ToCamelCase(p.Name))] = jen.Id(p.Name)
			}
		}
		rs := []jen.Code{}
		rt := []jen.Code{}
		for _, p := range m.Results {
			tp := p.Type
			ts := strings.Split(tp, ".")
			if len(ts) == 1 {
				if tp[:1] == strings.ToUpper(tp[:1]) {
					// If the type of the parameter is not `something.MyType` and it starts with an uppercase
					// than the type was defined inside the service package.
					tp = "service." + tp
				}
			}
			pth := e.EnsureThatWeUseQualifierIfNeeded(p.Type, e.file.Imports)
			if pth != "" {
				s := strings.Split(p.Type, ".")
				rs = append(rs, jen.Id(p.Name).Qual(pth, s[1]))
			} else {
				rs = append(rs, jen.Id(p.Name).Id(tp))
			}
			rt = append(rt, jen.Id(p.Name))
			resList = append(
				resList,
				jen.Id("response").Dot("").Call(jen.Id(m.Name+"Response")).Dot(utils.ToCamelCase(p.Name)),
			)
		}

		body := []jen.Code{
			jen.Id("request").Op(":=").Id(m.Name + "Request").Values(req),
			jen.List(jen.Id("response"), jen.Err()).Op(":=").Id(stp).Dot(m.Name + "Endpoint").Call(
				jen.List(jen.Id("ctx"), jen.Id("request")),
			),
			jen.If(
				jen.Err().Op("!=").Nil().Block(
					jen.Return(),
				),
			),
			jen.Return(jen.List(resList...)),
		}
		e.code.Raw().Commentf("%s implements Service. Primarily useful in a client.", m.Name).Line()
		e.code.appendFunction(
			m.Name,
			jen.Id(stp).Id("Endpoints"),
			sp,
			rs,
			"",
			body...,
		)
		e.code.NewLine()
	}
}

func (e *generateServiceEndpoints) generateMethodEndpoint() {
	for _, m := range e.serviceInterface.Methods {
		// For the request struct
		reqFields := []jen.Code{}
		// For the response struct
		resFields := []jen.Code{}

		mCallParam := []jen.Code{}
		respParam := jen.Dict{}
		retList := []jen.Code{}
		for _, p := range m.Parameters {
			if p.Type == "context.Context" {
				mCallParam = append(mCallParam, jen.Id(p.Name))
				continue
			}
			tp := p.Type
			ts := strings.Split(tp, ".")
			if len(ts) == 1 {
				if tp[:1] == strings.ToUpper(tp[:1]) {
					// If the type of the parameter is not `something.MyType` and it starts with an uppercase
					// than the type was defined inside the service package.
					tp = "service." + tp
				}
			}
			pth := e.EnsureThatWeUseQualifierIfNeeded(p.Type, e.file.Imports)
			if pth != "" {
				s := strings.Split(p.Type, ".")
				reqFields = append(reqFields, jen.Id(utils.ToCamelCase(p.Name)).Qual(pth, s[1]).Tag(map[string]string{
					"json": utils.ToLowerSnakeCase(utils.ToCamelCase(p.Name)),
				}))
			} else {
				reqFields = append(reqFields, jen.Id(utils.ToCamelCase(p.Name)).Id(tp).Tag(map[string]string{
					"json": utils.ToLowerSnakeCase(p.Name),
				}))
			}
			mCallParam = append(mCallParam, jen.Id("req").Dot(utils.ToCamelCase(p.Name)))

		}
		for _, p := range m.Results {
			tp := p.Type
			ts := strings.Split(tp, ".")
			if len(ts) == 1 {
				if tp[:1] == strings.ToUpper(tp[:1]) {
					// If the type of the parameter is not `something.MyType` and it starts with an uppercase
					// than the type was defined inside the service package.
					tp = "service." + tp
				}
			}
			pth := e.EnsureThatWeUseQualifierIfNeeded(p.Type, e.file.Imports)
			if pth != "" {
				s := strings.Split(p.Type, ".")
				resFields = append(resFields, jen.Id(utils.ToCamelCase(p.Name)).Qual(pth, s[1]).Tag(map[string]string{
					"json": utils.ToLowerSnakeCase(p.Name),
				}))
			} else {
				resFields = append(resFields, jen.Id(utils.ToCamelCase(p.Name)).Id(tp).Tag(map[string]string{
					"json": utils.ToLowerSnakeCase(p.Name),
				}))
			}
			respParam[jen.Id(utils.ToCamelCase(p.Name))] = jen.Id(p.Name)
			retList = append(retList, jen.Id(p.Name))
		}
		e.code.Raw().Commentf("%sRequest collects the request parameters for the %s method.", m.Name, m.Name)
		e.code.NewLine()
		e.code.appendStruct(
			m.Name+"Request",
			reqFields...,
		)
		e.code.NewLine()
		e.code.Raw().Commentf("%sResponse collects the response parameters for the %s method.", m.Name, m.Name)
		e.code.NewLine()
		e.code.appendStruct(
			m.Name+"Response",
			resFields...,
		)
		e.code.NewLine()
		pt := NewPartialGenerator(nil)
		pt.appendFunction(
			"",
			nil,
			[]jen.Code{
				jen.Id("ctx").Qual("context", "Context"),
				jen.Id("request").Interface(),
			},
			[]jen.Code{
				jen.Interface(),
				jen.Error(),
			},
			"",
			jen.Id("req").Op(":=").Id("request").Dot("").Call(
				jen.Id(m.Name+"Request"),
			),
			jen.List(retList...).Op(":=").Id("s").Dot(m.Name).Call(mCallParam...),
			jen.Return(jen.Id(m.Name+"Response").Values(respParam), jen.Nil()),
		)
		e.code.Raw().Commentf("Make%sEndpoint returns an endpoint that invokes %s on the service.", m.Name, m.Name)
		e.code.NewLine()
		e.code.appendFunction(
			"Make"+m.Name+"Endpoint",
			nil,
			[]jen.Code{
				jen.Id("s").Id("service").Dot(e.interfaceName),
			},
			[]jen.Code{},
			"endpoint.Endpoint",
			jen.Return(pt.Raw()),
		)
		e.code.NewLine()
	}
}
func (e *generateServiceEndpoints) generateNewMethod() (err error) {
	eps := jen.Dict{}
	loops := []jen.Code{}
	for _, v := range e.serviceInterface.Methods {
		eps[jen.Id(v.Name+"Endpoint")] = jen.Id("Make" + v.Name + "Endpoint").Call(jen.Id("s"))
		l := jen.For(jen.List(jen.Id("_"), jen.Id("m")).Op(":=").Range().Id("mdw").Index(jen.Lit(v.Name)))
		l.Block(
			jen.Id("eps").Dot(v.Name + "Endpoint").Op("=").Id("m").Call(jen.Id("eps").Dot(v.Name + "Endpoint")),
		)
		loops = append(loops, l)
	}
	svcImport, err := utils.GetServiceImportPath(e.name)
	if err != nil {
		return err
	}
	body := append([]jen.Code{
		jen.Id("eps").Op(":=").Id("Endpoints").Values(
			eps,
		),
	}, loops...)
	body = append(body, jen.Return(jen.Id("eps")))
	e.code.appendMultilineComment([]string{
		"New returns a Endpoints struct that wraps the provided service, and wires in all of the",
		"expected endpoint middlewares",
	})
	e.code.NewLine()
	e.code.appendFunction(
		"New",
		nil,
		[]jen.Code{
			jen.Id("s").Qual(svcImport, e.interfaceName),
			jen.Id("mdw").Map(
				jen.String(),
			).Index().Id("endpoint.Middleware"),
		},
		[]jen.Code{},
		"Endpoints",
		body...,
	)
	e.code.NewLine()
	return
}

type generateEndpointMiddleware struct {
	BaseGenerator
	name          string
	interfaceName string
	destPath      string
	filePath      string
}

func newGenerateEndpointMiddleware(name string) Gen {
	gsm := &generateEndpointMiddleware{
		name:          name,
		interfaceName: utils.ToCamelCase(name + "Service"),
		destPath:      fmt.Sprintf(viper.GetString("gk_endpoint_path_format"), utils.ToLowerSnakeCase(name)),
	}
	gsm.filePath = path.Join(gsm.destPath, viper.GetString("gk_endpoint_middleware_file_name"))
	gsm.srcFile = jen.NewFilePath(gsm.destPath)
	gsm.InitPg()
	gsm.fs = fs.Get()
	return gsm
}
func (g *generateEndpointMiddleware) Generate() (err error) {
	g.code.appendMultilineComment([]string{
		"InstrumentingMiddleware returns an endpoint middleware that records",
		"the duration of each invocation to the passed histogram. The middleware adds",
		"a single field: \"success\", which is \"true\" if no error is returned, and",
		"\"false\" otherwise.",
	})
	g.code.NewLine()
	deferBlock := jen.Defer()
	pl := NewPartialGenerator(deferBlock)
	pl.appendFunction(
		"",
		nil,
		[]jen.Code{
			jen.Id("begin").Qual("time", "Time"),
		},
		[]jen.Code{},
		"",
		jen.Id("duration").Dot("With").Call(
			jen.Lit("success"),
			jen.Qual("fmt", "Sprint").Call(jen.Id("err").Op("==").Nil()),
		).Dot("Observe").Call(jen.Id("time").Dot("Since").Call(jen.Id("begin")).Dot(
			"Seconds").Call(),
		),
	)
	inF := NewPartialGenerator(nil)
	inF.appendFunction(
		"",
		nil,
		[]jen.Code{
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("request").Interface(),
		},
		[]jen.Code{
			jen.Id("response").Interface(),
			jen.Id("err").Error(),
		},
		"",
		pl.Raw().Call(jen.Id("time").Dot("Now").Call()),
		jen.Return(jen.Id("next").Call(jen.Id("ctx"), jen.Id("request"))),
	)
	g.code.appendFunction(
		"InstrumentingMiddleware",
		nil,
		[]jen.Code{
			jen.Id("duration").Qual("github.com/go-kit/kit/metrics", "Histogram"),
		},
		[]jen.Code{},
		"endpoint.Middleware",
		jen.Return(
			jen.Func().Params(
				jen.Id("next").Qual("github.com/go-kit/kit/endpoint", "Endpoint"),
			).Id("endpoint.Endpoint").Block(
				jen.Return(inF.Raw()),
			),
		),
	)
	g.code.NewLine()
	g.code.appendMultilineComment([]string{
		"LoggingMiddleware returns an endpoint middleware that logs the",
		"duration of each invocation, and the resulting error, if any.",
	})
	g.code.NewLine()
	deferBlock1 := jen.Defer()
	pl1 := NewPartialGenerator(deferBlock1)
	pl1.appendFunction(
		"",
		nil,
		[]jen.Code{
			jen.Id("begin").Qual("time", "Time"),
		},
		[]jen.Code{},
		"",
		jen.Id("logger").Dot("Log").Call(
			jen.Lit("transport_error"),
			jen.Id("err"),
			jen.Lit("took"),
			jen.Id("time").Dot("Since").Call(jen.Id("begin")),
		),
	)
	inF1 := NewPartialGenerator(nil)
	inF1.appendFunction(
		"",
		nil,
		[]jen.Code{
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("request").Interface(),
		},
		[]jen.Code{
			jen.Id("response").Interface(),
			jen.Id("err").Error(),
		},
		"",
		pl1.Raw().Call(jen.Id("time").Dot("Now").Call()),
		jen.Return(jen.Id("next").Call(jen.Id("ctx"), jen.Id("request"))),
	)
	g.code.appendFunction(
		"LoggingMiddleware",
		nil,
		[]jen.Code{
			jen.Id("logger").Qual("github.com/go-kit/kit/log", "Logger"),
		},
		[]jen.Code{},
		"endpoint.Middleware",
		jen.Return(
			jen.Func().Params(
				jen.Id("next").Qual("github.com/go-kit/kit/endpoint", "Endpoint"),
			).Id("endpoint.Endpoint").Block(
				jen.Return(inF1.Raw()),
			),
		),
	)
	g.code.NewLine()
	return g.fs.WriteFile(g.filePath, g.srcFile.GoString(), false)
}

type generateHttpTransport struct {
	BaseGenerator
	name             string
	interfaceName    string
	destPath         string
	filePath         string
	serviceInterface parser.Interface
}

func newGenerateHttpTransport(name string, serviceInterface parser.Interface) Gen {
	t := &generateHttpTransport{
		name:             name,
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
	g.CreateFolderStructure(g.destPath)
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
			jen.Id("m").Dot("Handle").Call(
				jen.Lit("/"+strings.Replace(utils.ToLowerSnakeCase(m.Name), "_", "-", -1)),
				jen.Qual("github.com/go-kit/kit/transport/http", "NewServer").Call(
					jen.Id(fmt.Sprintf("endpoints.%sEndpoint", m.Name)),
					jen.Id(fmt.Sprintf("decode%sRequest", m.Name)),
					jen.Id(fmt.Sprintf("encode%sResponse", m.Name)),
					jen.Id(fmt.Sprintf("options[\"%s\"]...", m.Name)),
				),
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
	for _, m := range g.serviceInterface.Methods {

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
			jen.Id("req").Op(":=").Id("endpoint").Dot(m.Name+"Request").Block(),
			jen.Err().Op(":=").Qual("encoding/json", "NewDecoder").Call(
				jen.Id("r").Dot("Body"),
			).Dot("Decode").Call(jen.Id("&req")),
			jen.Return(jen.Id("req"), jen.Id("err")),
		)
		g.code.NewLine()
		g.code.appendMultilineComment([]string{
			fmt.Sprintf("encode%sResponse is a transport/http.EncodeResponseFunc that encodes", m.Name),
			"the response as JSON to the response writer",
		})
		g.code.NewLine()
		g.code.appendFunction(
			fmt.Sprintf("encode%sResponse", m.Name),
			nil,
			[]jen.Code{
				jen.Id("_").Qual("context", "Context"),
				jen.Id("w").Qual("net/http", "ResponseWriter"),
				jen.Id("response").Interface(),
			},
			[]jen.Code{
				jen.Id("err").Error(),
			},
			"",
			jen.Id("w").Dot("Header").Call().Dot("Set").Call(
				jen.Lit("Content-Type"), jen.Lit("application/json; charset=utf-8")),
			jen.Err().Op("=").Qual("encoding/json", "NewEncoder").Call(
				jen.Id("w"),
			).Dot("Encode").Call(jen.Id("response")),
			jen.Return(),
		)
		g.code.NewLine()
	}
	return g.fs.WriteFile(g.filePath, g.srcFile.GoString(), false)
}

type generateCmd struct {
	BaseGenerator
	name                               string
	interfaceName                      string
	destPath                           string
	filePath                           string
	generateSvcDefaultsMiddleware      bool
	generateEndpointDefaultsMiddleware bool
	serviceInterface                   parser.Interface
}

func newGenerateCmd(name string, serviceInterface parser.Interface,
	generateSacDefaultsMiddleware bool, generateEndpointDefaultsMiddleware bool) Gen {
	t := &generateCmd{
		name:                               name,
		interfaceName:                      utils.ToCamelCase(name + "Service"),
		destPath:                           fmt.Sprintf(viper.GetString("gk_cmd_path_format"), utils.ToLowerSnakeCase(name)),
		serviceInterface:                   serviceInterface,
		generateSvcDefaultsMiddleware:      generateSacDefaultsMiddleware,
		generateEndpointDefaultsMiddleware: generateEndpointDefaultsMiddleware,
	}
	t.filePath = path.Join(t.destPath, "main.go")
	t.srcFile = jen.NewFile("main")
	t.InitPg()
	t.fs = fs.Get()
	return t
}

func (g *generateCmd) Generate() (err error) {
	err = g.CreateFolderStructure(g.destPath)
	if err != nil {
		return err
	}
	err = g.generateMainMethod()
	if err != nil {
		return err
	}
	err = g.generateServiceMiddlewareMethod()
	if err != nil {
		return err
	}
	err = g.generateEndpointMiddlewareMethod()
	if err != nil {
		return err
	}
	return g.fs.WriteFile(g.filePath, g.srcFile.GoString(), false)
}

func (g *generateCmd) generateServiceMiddlewareMethod() (err error) {
	svcImport, err := utils.GetServiceImportPath(g.name)
	body := []jen.Code{}
	if !g.generateSvcDefaultsMiddleware {
		body = append(
			body,
			jen.Comment("Append your middleware here."),
		)
	} else {
		body = append(
			body,
			jen.Id("mw").Op("=").Index().Qual(svcImport, "Middleware").Block(),
			jen.Id("mw").Op("=").Append(
				jen.Id("mw"), jen.Id("service").Dot("LoggingMiddleware").Call(jen.Id("logger")),
			),
			jen.Comment("Append your middleware here."),
		)
	}
	body = append(body, jen.Return())
	g.code.NewLine()
	g.code.appendFunction(
		"getServiceMiddleware",
		nil,
		[]jen.Code{
			jen.Id("logger").Id("log.Logger"),
		},
		[]jen.Code{
			jen.Id("mw").Index().Qual(svcImport, "Middleware"),
		},
		"",
		body...,
	)
	g.code.NewLine()
	return
}
func (g *generateCmd) generateEndpointMiddlewareMethod() (err error) {
	endpointMdw, err := utils.GetEndpointImportPath(g.name)
	body := []jen.Code{}
	if !g.generateEndpointDefaultsMiddleware {
		body = append(
			body,
			jen.Comment("Append your middleware here."),
		)
	} else {
		body = append(
			body,
			jen.Id("mw").Op("=").Map(jen.Id("string")).Index().Qual(
				"github.com/go-kit/kit/endpoint", "Middleware").Block(),
			jen.Id("duration").Op(":=").Qual(
				"github.com/go-kit/kit/metrics/prometheus",
				"NewSummaryFrom",
			).Call(jen.Qual("github.com/prometheus/client_golang/prometheus", "SummaryOpts").Values(
				jen.Dict{
					jen.Id("Namespace"): jen.Lit("example"),
					jen.Id("Subsystem"): jen.Lit(g.name),
					jen.Id("Name"):      jen.Lit("request_duration_seconds"),
					jen.Id("Help"):      jen.Lit("Request duration in seconds."),
				},
			), jen.Index().String().Values(jen.Lit("method"), jen.Lit("success"))),
		)
		mdw := map[string][]jen.Code{}
		for _, m := range g.serviceInterface.Methods {
			if mdw[m.Name] == nil {
				mdw[m.Name] = []jen.Code{}
			}
			mdw[m.Name] = append(
				mdw[m.Name],
				jen.Qual(endpointMdw, "LoggingMiddleware").Call(
					jen.Id("log").Dot("With").Call(
						jen.Id("logger"),
						jen.Lit("method"),
						jen.Lit(m.Name),
					)),
			)
			mdw[m.Name] = append(
				mdw[m.Name],
				jen.Qual(endpointMdw, "InstrumentingMiddleware").Call(
					jen.Id("duration").Dot("With").Call(
						jen.Lit("method"),
						jen.Lit(m.Name),
					)),
			)
		}
		for _, m := range g.serviceInterface.Methods {
			body = append(
				body,
				jen.Id("mw").Index(jen.Lit(m.Name)).Op("=").Index().Qual("github.com/go-kit/kit/endpoint", "Middleware").Values(
					jen.List(mdw[m.Name]...),
				),
			)
		}
	}
	body = append(body, jen.Return())
	g.code.NewLine()
	g.code.appendFunction(
		"getEndpointMiddleware",
		nil,
		[]jen.Code{
			jen.Id("logger").Id("log.Logger"),
		},
		[]jen.Code{
			jen.Id("mw").Map(jen.Id("string")).Index().Qual("github.com/go-kit/kit/endpoint", "Middleware"),
		},
		"",
		body...,
	)
	g.code.NewLine()
	return
}

func (g *generateCmd) generateMainMethod() (err error) {
	pl := NewPartialGenerator(nil)
	pl.appendMultilineComment([]string{
		"Define our flags. Your service probably won't need to bind listeners for",
		"*all* supported transports, or support both Zipkin and LightStep, and so",
		"on, but we do it here for demonstration purposes.",
	})
	pl.Raw().Line()
	pl.Raw().Id("fs").Op(":=").Qual("flag", "NewFlagSet").Call(
		jen.Lit(g.name), jen.Id("flag").Dot("ExitOnError"),
	).Line()
	pl.Raw().Id("debugAddr").Op(":=").Id("fs").Dot("String").Call(
		jen.Lit("debug.addr"), jen.Lit(":8080"), jen.Lit("Debug and metrics listen address"),
	).Line()
	pl.Raw().Id("httpAddr").Op(":=").Id("fs").Dot("String").Call(
		jen.Lit("http-addr"), jen.Lit(":8001"), jen.Lit("HTTP listen address"),
	).Line()
	pl.Raw().Id("appdashAddr").Op(":=").Id("fs").Dot("String").Call(
		jen.Lit("appdash-addr"), jen.Lit(""), jen.Lit("Enable Appdash tracing via an Appdash server host:port"),
	).Line()
	pl.Raw().Id("fs").Dot("Parse").Call(
		jen.Id("os").Dot("Args").Index(jen.Lit(1), jen.Empty()),
	).Line().Line()
	g.generateLogger(pl)
	g.generateTracerLogic(pl)
	pl.NewLine()
	pl.Raw().Qual("net/http", "DefaultServeMux").Dot("Handle").Call(
		jen.Lit("/metrics"),
		jen.Qual("github.com/prometheus/client_golang/prometheus/promhttp", "Handler").Call(),
	)
	pl.NewLine()
	svcImport, err := utils.GetServiceImportPath(g.name)
	if err != nil {
		return err
	}
	epImport, err := utils.GetEndpointImportPath(g.name)

	if err != nil {
		return err
	}
	httpImport, err := utils.GetHttpTransportImportPath(g.name)
	if err != nil {
		return err
	}
	opt := jen.Dict{}
	for _, v := range g.serviceInterface.Methods {
		opt[jen.Lit(v.Name)] =
			jen.Values(
				jen.List(
					jen.Qual("github.com/go-kit/kit/transport/http", "ServerErrorLogger").Call(jen.Id("logger")),
					jen.Qual("github.com/go-kit/kit/transport/http", "ServerBefore").Call(
						jen.Qual("github.com/go-kit/kit/tracing/opentracing", "HTTPToContext").Call(
							jen.Id("tracer"),
							jen.Lit(v.Name),
							jen.Id("logger"),
						),
					),
				),
			)
	}
	pl.Raw().Id("options").Op(":=").Map(jen.String()).Index().Qual(
		"github.com/go-kit/kit/transport/http",
		"ServerOption",
	).Values(
		opt,
	)
	pl.NewLine()
	pl.Raw().Id("svc").Op(":=").Qual(svcImport, "New").Call(
		jen.Id("getServiceMiddleware").Call(jen.Id("logger")),
	)
	pl.NewLine()
	pl.Raw().Id("eps").Op(":=").Qual(epImport, "New").Call(
		jen.Id("svc"),
		jen.Id("getEndpointMiddleware").Call(jen.Id("logger")),
	)
	pl.NewLine()
	pl.Raw().Id("httpHandler").Op(":=").Qual(httpImport, "NewHTTPHandler").Call(
		jen.Id("eps"),
		jen.Id("options"),
	)
	g.appendGroups(pl)
	pl.NewLine()
	g.code.appendFunction(
		"main",
		nil,
		[]jen.Code{},
		[]jen.Code{},
		"",
		pl.Raw(),
	)
	return
}
func (g *generateCmd) appendGroups(pl *PartialGenerator) {
	pl.NewLine()
	pl.Raw().Var().Id("g").Qual("github.com/oklog/oklog/pkg/group", "Group").Line()
	pl.Raw().Block(
		jen.List(jen.Id("debugListener"), jen.Id("err")).Op(":=").Qual("net", "Listen").Call(
			jen.Lit("tcp"),
			jen.Id("*debugAddr"),
		),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Id("logger").Dot("Log").Call(
				jen.Lit("transport"),
				jen.Lit("debug/HTTP"),
				jen.Lit("during"),
				jen.Lit("Listen"),
				jen.Lit("err"),
				jen.Id("err"),
			),
			jen.Qual("os", "Exit").Call(jen.Lit(1)),
		),
		jen.Id("g").Dot("Add").Call(
			jen.List(
				jen.Func().Params().Error().Block(
					jen.Id("logger").Dot("Log").Call(
						jen.Lit("transport"),
						jen.Lit("debug/HTTP"),
						jen.Lit("addr"),
						jen.Id("*debugAddr"),
					),
					jen.Return(
						jen.Qual("net/http", "Serve").Call(
							jen.Id("debugListener"),
							jen.Qual("net/http", "DefaultServeMux"),
						),
					),
				),
				jen.Func().Params(jen.Error()).Block(
					jen.Id("debugListener").Dot("Close").Call(),
				),
			),
		),
	)
	pl.NewLine()
	pl.Raw().Block(
		jen.List(jen.Id("httpListener"), jen.Id("err")).Op(":=").Qual("net", "Listen").Call(
			jen.Lit("tcp"),
			jen.Id("*httpAddr"),
		),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Id("logger").Dot("Log").Call(
				jen.Lit("transport"),
				jen.Lit("HTTP"),
				jen.Lit("during"),
				jen.Lit("Listen"),
				jen.Lit("err"),
				jen.Id("err"),
			),
			jen.Qual("os", "Exit").Call(jen.Lit(1)),
		),
		jen.Id("g").Dot("Add").Call(
			jen.List(
				jen.Func().Params().Error().Block(
					jen.Id("logger").Dot("Log").Call(
						jen.Lit("transport"),
						jen.Lit("HTTP"),
						jen.Lit("addr"),
						jen.Id("*httpAddr"),
					),
					jen.Return(
						jen.Qual("net/http", "Serve").Call(
							jen.Id("httpListener"),
							jen.Id("httpHandler"),
						),
					),
				),
				jen.Func().Params(jen.Error()).Block(
					jen.Id("httpListener").Dot("Close").Call(),
				),
			),
		),
	)
	pl.NewLine()
	pl.Raw().Block(
		jen.Id("cancelInterrupt").Op(":=").Make(jen.Chan().Struct()),
		jen.Id("g").Dot("Add").Call(
			jen.List(
				jen.Func().Params().Error().Block(
					jen.Id("c").Op(":=").Make(jen.Chan().Qual("os", "Signal"), jen.Lit(1)),
					jen.Qual("os/signal", "Notify").Call(
						jen.Id("c"),
						jen.Qual("syscall", "SIGINT"),
						jen.Qual("syscall", "SIGTERM"),
					),
					jen.Select().Block(
						jen.Case(jen.Id("sig").Op(":=").Id("<-c")).Block(
							jen.Return(
								jen.Qual("fmt", "Errorf").Call(
									jen.Lit("received signal %s"),
									jen.Id("sig"),
								),
							),
						),
						jen.Case(jen.Id("<-cancelInterrupt")).Block(
							jen.Return(
								jen.Nil(),
							),
						),
					),
				),
				jen.Func().Params(jen.Error()).Block(
					jen.Id("close").Call(jen.Id("cancelInterrupt")),
				),
			),
		),
	)
	pl.NewLine()
	//	logger.Log("exit", g.Run())
	pl.Raw().Id("logger").Dot("Log").Call(
		jen.Lit("exit"),
		jen.Id("g").Dot("Run").Call(),
	)
}
func (g *generateCmd) generateLogger(pl *PartialGenerator) {
	pl.appendMultilineComment([]string{
		"Create a single logger, which we'll use and give to other components.",
	})
	pl.NewLine()
	pl.Raw().Var().Id("logger").Qual("github.com/go-kit/kit/log", "Logger")
	pl.Raw().Line().Block(
		jen.Id("logger").Op("=").Id("log").Dot("NewLogfmtLogger").Call(
			jen.Qual("os", "Stderr"),
		),
		jen.Line(),
		jen.Id("logger").Op("=").Id("log").Dot("With").Call(
			jen.Id("logger"),
			jen.Lit("ts"),
			jen.Id("log").Dot("DefaultTimestampUTC"),
		),
		jen.Line(),
		jen.Id("logger").Op("=").Id("log").Dot("With").Call(
			jen.Id("logger"),
			jen.Lit("caller"),
			jen.Id("log").Dot("DefaultCaller"),
		),
	)
}
func (g *generateCmd) generateTracerLogic(pl *PartialGenerator) {
	pl.appendMultilineComment([]string{
		"Determine which tracer to use. We'll pass the tracer to all the",
		"components that use it, as a dependency.",
	})
	pl.NewLine()
	pl.Raw().Var().Id("tracer").Qual("github.com/opentracing/opentracing-go", "Tracer")
	pl.Raw().Line().Block(
		jen.If(
			jen.Id("*appdashAddr").Op("!=").Lit("").Block(
				jen.Id("logger").Dot("Log").Call(
					jen.Lit("tracer"),
					jen.Lit("Appdash"),
					jen.Lit("addr"),
					jen.Id("*appdashAddr"),
				),
				//tracer = appdashot.NewTracer(appdash.NewRemoteCollector(*appdashAddr))
				jen.Id("tracer").Op("=").Qual(
					"sourcegraph.com/sourcegraph/appdash/opentracing",
					"NewTracer").Call(
					jen.Qual("sourcegraph.com/sourcegraph/appdash", "NewRemoteCollector").Call(
						jen.Id("*appdashAddr"),
					),
				),
			),
		).Else().Block(
			jen.Id("logger").Dot("Log").Call(
				jen.Lit("tracer"),
				jen.Lit("none"),
			),
			jen.Id("tracer").Op("=").Qual(
				"github.com/opentracing/opentracing-go",
				"GlobalTracer").Call(),
		),
	)
}
