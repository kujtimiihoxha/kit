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

// SUPPORTED_TRANSPORTS is an array containing the supported transport types.
var SUPPORTED_TRANSPORTS = []string{"http", "grpc", "thrift"}

type GenerateService struct {
	BaseGenerator
	pg                       *PartialGenerator
	name                     string
	transport                string
	interfaceName            string
	serviceStructName        string
	destPath                 string
	methods                  []string
	filePath                 string
	file                     *parser.File
	serviceInterface         parser.Interface
	sMiddleware, eMiddleware bool
}

// NewGenerateService returns a initialized and ready generator.
//
// The name parameter is the name of the service that will be created
// this name should be without the `Service` suffix
//
// The sMiddleware specifies if the default service middleware should be
// created
//
// The eMiddleware specifies if the default endpoint middleware should be
// created
func NewGenerateService(name, transport string, sMiddleware, eMiddleware bool, methods []string) Gen {
	i := &GenerateService{
		name:          name,
		interfaceName: utils.ToCamelCase(name + "Service"),
		destPath:      fmt.Sprintf(viper.GetString("gk_service_path_format"), utils.ToLowerSnakeCase(name)),
		sMiddleware:   sMiddleware,
		eMiddleware:   eMiddleware,
		methods:       methods,
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
func (g *GenerateService) Generate() (err error) {
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
	g.generateServiceStruct()
	g.generateServiceMethods()
	g.generateNewBasicStructMethod()
	g.generateNewMethod()
	svcSrc += "\n" + g.pg.String()
	s, err := utils.GoImportsSource(g.destPath, svcSrc)
	err = g.fs.WriteFile(g.filePath, s, true)
	if err != nil {
		return err
	}
	mdwG := newGenerateServiceMiddleware(g.name, g.file, g.serviceInterface, g.sMiddleware)
	err = mdwG.Generate()
	if err != nil {
		return err
	}
	epGB := newGenerateServiceEndpointsBase(g.name, g.serviceInterface)
	err = epGB.Generate()
	if err != nil {
		return err
	}
	epG := newGenerateServiceEndpoints(g.name, g.file.Imports, g.serviceInterface, g.eMiddleware)
	err = epG.Generate()
	if err != nil {
		return err
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
	default:
		logrus.Warn("This transport type is not yet implemented")
	}
	mG := newGenerateCmd(g.name, g.serviceInterface, g.sMiddleware, g.eMiddleware, g.methods)
	return mG.Generate()
}
func (g *GenerateService) generateServiceMethods() {
	stp := ""
	methodParameterNames := []parser.NamedTypeValue{}
	for _, v := range g.serviceInterface.Methods {
		methodParameterNames = append(methodParameterNames, v.Parameters...)
		methodParameterNames = append(methodParameterNames, v.Results...)
	}
	stp = g.GenerateNameBySample(g.serviceStructName, methodParameterNames)
	for _, m := range g.serviceInterface.Methods {
		exists := false
		for _, v := range g.file.Methods {
			if v.Name == m.Name && v.Struct.Type == "*"+g.serviceStructName {
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
		g.pg.appendFunction(
			m.Name,
			jen.Id(stp).Id("*"+g.serviceStructName),
			sp,
			rs,
			"",
			body...,
		)
		g.pg.NewLine()
	}
}
func (g *GenerateService) generateServiceStruct() {
	for _, v := range g.file.Structures {
		if v.Name == g.serviceStructName {
			logrus.Infof("Service `%s` structure already exists so it will not be recreated.", g.serviceStructName)
			return
		}
	}
	g.pg.appendStruct(g.serviceStructName)
}
func (g *GenerateService) generateNewMethod() {
	for _, v := range g.file.Methods {
		if v.Name == "New" {
			logrus.Infof("Service method `%s` already exists so it will not be recreated.", v.Name)
			return
		}
	}
	g.pg.Raw().Commentf(
		"New returns a %s with all of the expected middleware wired in.",
		g.interfaceName,
	).Line()
	fn := fmt.Sprintf("New%s", utils.ToCamelCase(g.serviceStructName))
	body := []jen.Code{
		jen.Var().Id("svc").Id(g.interfaceName).Op("=").Id(fn).Call(),
		jen.For(
			jen.List(jen.Id("_"), jen.Id("m")).Op(":=").Range().Id("middleware"),
		).Block(
			jen.Id("svc").Op("=").Id("m").Call(jen.Id("svc")),
		),
		jen.Return(jen.Id("svc")),
	}
	g.pg.appendFunction(
		"New",
		nil,
		[]jen.Code{
			jen.Id("middleware").Id("[]Middleware"),
		},
		[]jen.Code{},
		g.interfaceName,
		body...)
	g.pg.NewLine()
}
func (g *GenerateService) generateNewBasicStructMethod() {
	fn := fmt.Sprintf("New%s", utils.ToCamelCase(g.serviceStructName))
	for _, v := range g.file.Methods {
		if v.Name == fn {
			logrus.Infof("Service method `%s` already exists so it will not be recreated.", v.Name)
			return
		}
	}
	g.pg.Raw().Commentf(
		"New%s returns a naive, stateless implementation of %s.",
		utils.ToCamelCase(g.serviceStructName),
		g.interfaceName,
	).Line()
	body := []jen.Code{
		jen.Return(jen.Id(fmt.Sprintf("&%s{}", g.serviceStructName))),
	}
	g.pg.appendFunction(fn, nil, []jen.Code{}, []jen.Code{}, g.interfaceName, body...)
	g.pg.NewLine()
}
func (g *GenerateService) serviceFound() bool {
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
func (g *GenerateService) removeBadMethods() {
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

type generateServiceMiddleware struct {
	BaseGenerator
	name              string
	interfaceName     string
	generateFirstTime bool
	destPath          string
	filePath          string
	serviceFile       *parser.File
	file              *parser.File
	serviceInterface  parser.Interface
	generateDefaults  bool
}

func newGenerateServiceMiddleware(name string, serviceFile *parser.File,
	serviceInterface parser.Interface, generateDefaults bool) Gen {
	gsm := &generateServiceMiddleware{
		name:             name,
		interfaceName:    utils.ToCamelCase(name + "Service"),
		destPath:         fmt.Sprintf(viper.GetString("gk_service_path_format"), utils.ToLowerSnakeCase(name)),
		serviceInterface: serviceInterface,
		serviceFile:      serviceFile,
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
	if b, err := g.fs.Exists(g.filePath); err != nil {
		return err
	} else {
		if !b {
			g.generateFirstTime = true
			f := jen.NewFile("service")
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
	tpFound := false
	if g.file.FuncType.Name == "Middleware" {
		if len(g.file.FuncType.Parameters) == 1 && len(g.file.FuncType.Results) == 1 {
			if g.file.FuncType.Parameters[0].Type == g.serviceInterface.Name &&
				g.file.FuncType.Results[0].Type == g.serviceInterface.Name {
				tpFound = true
			}
		}
	}
	if !tpFound {
		g.code.Raw().Comment("Middleware describes a service middleware.").Line()
		g.code.Raw().Type().Id("Middleware").Func().Params(jen.Id(g.interfaceName)).Id(g.interfaceName).Line()
		g.code.NewLine()
	}
	if g.generateDefaults {
		strFound := false
		for _, v := range g.file.Structures {
			if v.Name == "loggingMiddleware" {
				strFound = true
				break
			}
		}
		if !strFound {
			g.code.appendStruct(
				"loggingMiddleware",
				jen.Id("logger").Qual("github.com/go-kit/kit/log", "Logger"),
				jen.Id("next").Id(g.interfaceName),
			)
		}
		mthdFound := false
		for _, v := range g.file.Methods {
			if v.Name == "LoggingMiddleware" {
				mthdFound = true
				break
			}
		}
		if !mthdFound {
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
					jen.Id("logger").Qual("github.com/go-kit/kit/log", "Logger"),
				},
				[]jen.Code{},
				"Middleware",
				jen.Return(pt.Raw()),
			)
			g.code.NewLine()
			g.code.NewLine()
		}
		g.generateMethodMiddleware()
	}
	if g.generateFirstTime {
		return g.fs.WriteFile(g.filePath, g.srcFile.GoString(), true)
	}
	src += "\n" + g.code.Raw().GoString()
	tmpSrc := g.srcFile.GoString()
	f, err := parser.NewFileParser().Parse([]byte(tmpSrc))
	if err != nil {
		return err
	}
	// See if we need to add any new import
	imp := []parser.NamedTypeValue{}
	for _, v := range f.Imports {
		for i, vo := range g.file.Imports {
			if v.Type == vo.Type && v.Name == vo.Name {
				break
			} else if i == len(g.file.Imports)-1 {
				imp = append(imp, v)
			}
		}
	}
	if len(g.file.Imports) == 0 {
		imp = f.Imports
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

func (g *generateServiceMiddleware) generateMethodMiddleware() {
	stp := ""
	methodParameterNames := []parser.NamedTypeValue{}
	for _, v := range g.serviceInterface.Methods {
		methodParameterNames = append(methodParameterNames, v.Parameters...)
		methodParameterNames = append(methodParameterNames, v.Results...)
	}
	stp = g.GenerateNameBySample("loggingMiddleware", methodParameterNames)
	for _, m := range g.serviceInterface.Methods {
		mthdFound := false
		for _, v := range g.file.Methods {
			if v.Name == m.Name && v.Struct.Type == "loggingMiddleware" {
				mthdFound = true
				break
			}
		}
		if !mthdFound {
			middlewareFuncParam := []jen.Code{}
			middlewareFuncResult := []jen.Code{}
			loggerLog := []jen.Code{}
			middlewareReturn := []jen.Code{}
			for _, p := range m.Parameters {
				pth := g.EnsureThatWeUseQualifierIfNeeded(p.Type, g.serviceFile.Imports)
				if pth != "" {
					s := strings.Split(p.Type, ".")
					middlewareFuncParam = append(middlewareFuncParam, jen.Id(p.Name).Qual(pth, s[1]))
				} else {
					if p.Type == "context.Context" {
						middlewareFuncParam = append(middlewareFuncParam, jen.Id(p.Name).Qual("context", "Context"))
					} else {
						middlewareFuncParam = append(middlewareFuncParam, jen.Id(p.Name).Id(p.Type))
					}
				}
				middlewareReturn = append(middlewareReturn, jen.Id(p.Name))
				if p.Type != "context.Context" {
					loggerLog = append(loggerLog, jen.Lit(p.Name), jen.Id(p.Name))
				}
			}
			for _, p := range m.Results {
				pth := g.EnsureThatWeUseQualifierIfNeeded(p.Type, g.serviceFile.Imports)
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
}

type generateServiceEndpoints struct {
	BaseGenerator
	name              string
	interfaceName     string
	destPath          string
	filePath          string
	serviceImports    []parser.NamedTypeValue
	serviceInterface  parser.Interface
	file              *parser.File
	generateDefaults  bool
	generateFirstTime bool
}

func newGenerateServiceEndpoints(name string, imports []parser.NamedTypeValue,
	serviceInterface parser.Interface, generateDefaults bool) Gen {
	gsm := &generateServiceEndpoints{
		name:             name,
		interfaceName:    utils.ToCamelCase(name + "Service"),
		destPath:         fmt.Sprintf(viper.GetString("gk_endpoint_path_format"), utils.ToLowerSnakeCase(name)),
		serviceInterface: serviceInterface,
		serviceImports:   imports,
	}
	gsm.filePath = path.Join(gsm.destPath, viper.GetString("gk_endpoint_file_name"))
	gsm.generateDefaults = generateDefaults
	gsm.srcFile = jen.NewFilePath(gsm.destPath)
	gsm.InitPg()
	gsm.fs = fs.Get()
	return gsm
}
func (g *generateServiceEndpoints) Generate() error {
	g.CreateFolderStructure(g.destPath)
	if b, err := g.fs.Exists(g.filePath); err != nil {
		return err
	} else {
		if !b {
			g.generateFirstTime = true
			f := jen.NewFile("endpoint")
			g.fs.WriteFile(g.filePath, f.GoString(), false)
		}
	}
	epSrc, err := g.fs.ReadFile(g.filePath)
	if err != nil {
		return err
	}
	g.file, err = parser.NewFileParser().Parse([]byte(epSrc))
	if err != nil {
		return err
	}
	g.generateMethodEndpoint()
	g.generateEndpointsClientMethods()
	if g.generateDefaults {
		mdw := newGenerateEndpointMiddleware(g.name)
		err = mdw.Generate()
		if err != nil {
			return err
		}
	}
	if g.generateFirstTime {
		return g.fs.WriteFile(g.filePath, g.srcFile.GoString(), true)
	}
	epSrc += "\n" + g.code.Raw().GoString()
	tmpSrc := g.srcFile.GoString()
	f, err := parser.NewFileParser().Parse([]byte(tmpSrc))
	if err != nil {
		return err
	}
	// See if we need to add any new import
	imp := []parser.NamedTypeValue{}
	for _, v := range f.Imports {
		for i, vo := range g.file.Imports {
			if v.Type == vo.Type && v.Name == vo.Name {
				break
			} else if i == len(g.file.Imports)-1 {
				imp = append(imp, v)
			}
		}
	}
	if len(g.file.Imports) == 0 {
		imp = f.Imports
	}
	if len(imp) > 0 {
		epSrc, err = g.AddImportsToFile(imp, epSrc)
		if err != nil {
			return err
		}
	}
	s, err := utils.GoImportsSource(g.destPath, epSrc)
	if err != nil {
		return err
	}
	return g.fs.WriteFile(g.filePath, s, true)
}

func (g *generateServiceEndpoints) generateEndpointsClientMethods() {
	stp := ""
	methodParameterNames := []parser.NamedTypeValue{}
	for _, v := range g.serviceInterface.Methods {
		methodParameterNames = append(methodParameterNames, v.Parameters...)
		methodParameterNames = append(methodParameterNames, v.Results...)
	}
	stp = g.GenerateNameBySample("Endpoints", methodParameterNames)
	for _, m := range g.serviceInterface.Methods {
		found := false
		for _, v := range g.file.Methods {
			if v.Name == m.Name && v.Struct.Type == "Endpoints" {
				found = true
				break
			}
		}
		if found {
			continue
		}
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
			pth := g.EnsureThatWeUseQualifierIfNeeded(p.Type, g.serviceImports)
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
			pth := g.EnsureThatWeUseQualifierIfNeeded(p.Type, g.serviceImports)
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
		g.code.Raw().Commentf("%s implements Service. Primarily useful in a client.", m.Name).Line()
		g.code.appendFunction(
			m.Name,
			jen.Id(stp).Id("Endpoints"),
			sp,
			rs,
			"",
			body...,
		)
		g.code.NewLine()
	}
}

func (g *generateServiceEndpoints) generateMethodEndpoint() (err error) {
	sImp, err := utils.GetServiceImportPath(g.name)
	if err != nil {
		return err
	}
	for _, m := range g.serviceInterface.Methods {
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
			pth := g.EnsureThatWeUseQualifierIfNeeded(p.Type, g.serviceImports)
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
			pth := g.EnsureThatWeUseQualifierIfNeeded(p.Type, g.serviceImports)
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
		requestStructExists := false
		responseStructExists := false
		makeMethdExists := false
		for _, v := range g.file.Structures {
			if v.Name == m.Name+"Request" {
				requestStructExists = true
			}
			if v.Name == m.Name+"Response" {
				responseStructExists = true
			}
			if requestStructExists && responseStructExists {
				break
			}
		}
		for _, v := range g.file.Methods {
			if v.Name == "Make"+m.Name+"Endpoint" {
				makeMethdExists = true
				break
			}
		}
		if !requestStructExists {
			g.code.Raw().Commentf("%sRequest collects the request parameters for the %s method.", m.Name, m.Name)
			g.code.NewLine()
			g.code.appendStruct(
				m.Name+"Request",
				reqFields...,
			)
			g.code.NewLine()
		}
		if !responseStructExists {
			g.code.Raw().Commentf("%sResponse collects the response parameters for the %s method.", m.Name, m.Name)
			g.code.NewLine()
			g.code.appendStruct(
				m.Name+"Response",
				resFields...,
			)
			g.code.NewLine()
		}
		if !makeMethdExists {
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
			g.code.Raw().Commentf("Make%sEndpoint returns an endpoint that invokes %s on the service.", m.Name, m.Name)
			g.code.NewLine()
			g.code.appendFunction(
				"Make"+m.Name+"Endpoint",
				nil,
				[]jen.Code{
					jen.Id("s").Qual(sImp, g.interfaceName),
				},
				[]jen.Code{
					jen.Qual("github.com/go-kit/kit/endpoint", "Endpoint"),
				},
				"",
				jen.Return(pt.Raw()),
			)
			g.code.NewLine()
		}
	}
	return
}

type generateServiceEndpointsBase struct {
	BaseGenerator
	name             string
	interfaceName    string
	destPath         string
	filePath         string
	serviceInterface parser.Interface
}

func newGenerateServiceEndpointsBase(name string, serviceInterface parser.Interface) Gen {
	gsm := &generateServiceEndpointsBase{
		name:             name,
		interfaceName:    utils.ToCamelCase(name + "Service"),
		destPath:         fmt.Sprintf(viper.GetString("gk_endpoint_path_format"), utils.ToLowerSnakeCase(name)),
		serviceInterface: serviceInterface,
	}
	gsm.filePath = path.Join(gsm.destPath, viper.GetString("gk_endpoint_base_file_name"))
	gsm.srcFile = jen.NewFilePath(gsm.destPath)
	gsm.InitPg()
	gsm.fs = fs.Get()
	return gsm
}
func (g *generateServiceEndpointsBase) Generate() (err error) {
	err = g.CreateFolderStructure(g.destPath)
	if err != nil {
		return err
	}
	fields := []jen.Code{}
	for _, v := range g.serviceInterface.Methods {
		fields = append(fields, jen.Id(v.Name+"Endpoint").Qual("github.com/go-kit/kit/endpoint", "Endpoint"))
	}
	g.srcFile.PackageComment("THIS FILE IS AUTO GENERATED BY GK-CLI DO NOT EDIT!!")
	g.code.appendMultilineComment([]string{
		"Endpoints collects all of the endpoints that compose a profile service. It's",
		"meant to be used as a helper struct, to collect all of the endpoints into a",
		"single parameter.",
	})
	g.code.NewLine()
	g.code.appendStruct(
		"Endpoints",
		fields...,
	)
	eps := jen.Dict{}
	loops := []jen.Code{}
	for _, v := range g.serviceInterface.Methods {
		eps[jen.Id(v.Name+"Endpoint")] = jen.Id("Make" + v.Name + "Endpoint").Call(jen.Id("s"))
		l := jen.For(jen.List(jen.Id("_"), jen.Id("m")).Op(":=").Range().Id("mdw").Index(jen.Lit(v.Name)))
		l.Block(
			jen.Id("eps").Dot(v.Name + "Endpoint").Op("=").Id("m").Call(jen.Id("eps").Dot(v.Name + "Endpoint")),
		)
		loops = append(loops, l)
	}
	svcImport, err := utils.GetServiceImportPath(g.name)
	if err != nil {
		return err
	}
	body := append([]jen.Code{
		jen.Id("eps").Op(":=").Id("Endpoints").Values(
			eps,
		),
	}, loops...)
	body = append(body, jen.Return(jen.Id("eps")))
	g.code.appendMultilineComment([]string{
		"New returns a Endpoints struct that wraps the provided service, and wires in all of the",
		"expected endpoint middlewares",
	})
	g.code.NewLine()
	g.code.appendFunction(
		"New",
		nil,
		[]jen.Code{
			jen.Id("s").Qual(svcImport, g.interfaceName),
			jen.Id("mdw").Map(
				jen.String(),
			).Index().Id("endpoint.Middleware"),
		},
		[]jen.Code{},
		"Endpoints",
		body...,
	)
	g.code.NewLine()
	return g.fs.WriteFile(g.filePath, g.srcFile.GoString(), true)
}

type generateEndpointMiddleware struct {
	BaseGenerator
	name              string
	generateFirstTime bool
	interfaceName     string
	file              *parser.File
	destPath          string
	filePath          string
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
	err = g.CreateFolderStructure(g.destPath)
	if err != nil {
		return err
	}
	if b, err := g.fs.Exists(g.filePath); err != nil {
		return err
	} else {
		if !b {
			g.generateFirstTime = true
			f := jen.NewFile("endpoint")
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
	defaultInstrumeningExists := false
	defaultLoggingExists := false
	for _, v := range g.file.Methods {
		if v.Name == "InstrumentingMiddleware" {
			defaultInstrumeningExists = true
		}
		if v.Name == "LoggingMiddleware" {
			defaultLoggingExists = true
		}
	}
	if !defaultInstrumeningExists {
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
	}
	if !defaultLoggingExists {
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
	}
	if g.generateFirstTime {
		return g.fs.WriteFile(g.filePath, g.srcFile.GoString(), true)
	}

	src += "\n" + g.code.Raw().GoString()
	tmpSrc := g.srcFile.GoString()
	f, err := parser.NewFileParser().Parse([]byte(tmpSrc))
	if err != nil {
		return err
	}
	// See if we need to add any new import
	imp := []parser.NamedTypeValue{}
	for _, v := range f.Imports {
		for i, vo := range g.file.Imports {
			if v.Type == vo.Type && v.Name == vo.Name {
				break
			} else if i == len(g.file.Imports)-1 {
				imp = append(imp, v)
			}
		}
	}
	if len(g.file.Imports) == 0 {
		imp = f.Imports
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
	foundMap := false
	for _, v := range g.file.Vars {
		if v.Name == "URL_MAP" {
			foundMap = true
		}
	}
	if !foundMap {
		vl := jen.Dict{}
		for _, m := range g.serviceInterface.Methods {
			if len(g.methods) > 0 {
				notFound := true
				for _, v := range g.methods {
					if m.Name == v {
						notFound = false
						break
					}
				}
				if notFound {
					continue
				}
			}
			vl[jen.Lit(m.Name)] = jen.Lit("/" + strings.Replace(utils.ToLowerSnakeCase(m.Name), "_", "-", -1))
		}
		g.code.Raw().Comment("Url map for service method, update here if you want to change the url of method")
		g.code.NewLine()
		g.code.Raw().Var().Id("URL_MAP").Op("=").Map(jen.Id("string")).String().Values(vl).Line()
	}
	for _, m := range g.serviceInterface.Methods {
		if len(g.methods) > 0 {
			notFound := true
			for _, v := range g.methods {
				if m.Name == v {
					notFound = false
					break
				}
			}
			if notFound {
				continue
			}
		}
		decoderFound := false
		encoderFound := false
		for _, v := range g.file.Methods {
			if v.Name == fmt.Sprintf("decode%sRequest", m.Name) {
				decoderFound = true
			}
			if v.Name == fmt.Sprintf("encode%sResponse", m.Name) {
				encoderFound = true
			}
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
	imp := []parser.NamedTypeValue{}
	for _, v := range f.Imports {
		for i, vo := range g.file.Imports {
			if v.Type == vo.Type && v.Name == vo.Name {
				break
			} else if i == len(g.file.Imports)-1 {
				imp = append(imp, v)
			}
		}
	}
	if len(g.file.Imports) == 0 {
		imp = f.Imports
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

		if len(g.methods) > 0 {
			notFound := true
			for _, v := range g.methods {
				if m.Name == v {
					notFound = false
					break
				}
			}
			if notFound {
				continue
			}
		}
		handles = append(
			handles,
			jen.Id("m").Dot("Handle").Call(
				jen.Id("URL_MAP").Index(jen.Lit(m.Name)),
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
	return g.fs.WriteFile(g.filePath, g.srcFile.GoString(), true)
}

type generateCmd struct {
	BaseGenerator
	name                               string
	methods                            []string
	interfaceName                      string
	destPath                           string
	filePath                           string
	generateSvcDefaultsMiddleware      bool
	generateEndpointDefaultsMiddleware bool
	serviceInterface                   parser.Interface
}

func newGenerateCmd(name string, serviceInterface parser.Interface,
	generateSacDefaultsMiddleware bool, generateEndpointDefaultsMiddleware bool, methods []string) Gen {
	t := &generateCmd{
		name:                               name,
		methods:                            methods,
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
			jen.Id("mw").Op("=").Index().Qual(svcImport, "Middleware").Block(),
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
			jen.Id("mw").Op("=").Map(jen.Id("string")).Index().Qual(
				"github.com/go-kit/kit/endpoint", "Middleware").Block(),
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