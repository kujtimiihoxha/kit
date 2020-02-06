package generator

import (
	"fmt"
	"path"
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/kujtimiihoxha/kit/fs"
	"github.com/kujtimiihoxha/kit/parser"
	"github.com/kujtimiihoxha/kit/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// SupportedTransports is an array containing the supported transport types.
var SupportedTransports = []string{"http", "grpc"}

// GenerateService implements Gen and is used to generate the service.
type GenerateService struct {
	BaseGenerator
	pg                                   *PartialGenerator
	name                                 string
	transport                            string
	interfaceName                        string
	serviceStructName                    string
	destPath                             string
	methods                              []string
	filePath                             string
	file                                 *parser.File
	serviceInterface                     parser.Interface
	sMiddleware, gorillaMux, eMiddleware bool
}

// NewGenerateService returns a initialized and ready generator.
func NewGenerateService(name, transport string, sMiddleware, gorillaMux, eMiddleware bool, methods []string) Gen {
	i := &GenerateService{
		name:          name,
		interfaceName: utils.ToCamelCase(name + "Service"),
		destPath:      fmt.Sprintf(viper.GetString("gk_service_path_format"), utils.ToLowerSnakeCase(name)),
		sMiddleware:   sMiddleware,
		eMiddleware:   eMiddleware,
		gorillaMux:    gorillaMux,
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

// Generate generates the service.
func (g *GenerateService) Generate() (err error) {
	for n, v := range SupportedTransports {
		if v == g.transport {
			break
		} else if n == len(SupportedTransports)-1 {
			logrus.Errorf("Transport `%s` not supported", g.transport)
			return
		}
	}
	if b, err := g.fs.Exists(g.filePath); err != nil {
		return err
	} else if !b {
		logrus.Errorf("Service %s was not found", g.name)
		return nil
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
	if err != nil {
		return err
	}
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
	tp := NewGenerateTransport(g.name, g.gorillaMux, g.transport, g.methods)
	err = tp.Generate()
	if err != nil {
		return err
	}
	mbG := newGenerateCmdBase(g.name, g.serviceInterface, g.sMiddleware, g.eMiddleware, g.methods)
	err = mbG.Generate()
	if err != nil {
		return err
	}
	mG := newGenerateCmd(g.name, g.serviceInterface, g.sMiddleware, g.eMiddleware, g.methods)
	return mG.Generate()
}
func (g *GenerateService) generateServiceMethods() {
	var stp string
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
				logrus.Debugf("Service method `%s` already exists so it will not be recreated.", v.Name)
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
			logrus.Debugf("Service `%s` structure already exists so it will not be recreated.", g.serviceStructName)
			return
		}
	}
	g.pg.appendStruct(g.serviceStructName)
}
func (g *GenerateService) generateNewMethod() {
	for _, v := range g.file.Methods {
		if v.Name == "New" {
			logrus.Debugf("Service method `%s` already exists so it will not be recreated.", v.Name)
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
			logrus.Debugf("Service method `%s` already exists so it will not be recreated.", v.Name)
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
	} else if !b {
		g.generateFirstTime = true
		f := jen.NewFile("service")
		g.fs.WriteFile(g.filePath, f.GoString(), false)
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
		g.generateMethodMiddleware("loggingMiddleware", true)
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

func (g *generateServiceMiddleware) generateMethodMiddleware(mdw string, df bool) {
	var stp string
	methodParameterNames := []parser.NamedTypeValue{}
	for _, v := range g.serviceInterface.Methods {
		methodParameterNames = append(methodParameterNames, v.Parameters...)
		methodParameterNames = append(methodParameterNames, v.Results...)
	}
	stp = g.GenerateNameBySample(mdw, methodParameterNames)
	for _, m := range g.serviceInterface.Methods {
		mthdFound := false
		for _, v := range g.file.Methods {
			if v.Name == m.Name && v.Struct.Type == mdw {
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
			var deferBlock *jen.Statement
			if df {
				deferBlock = jen.Defer().Func().Call().Block(jen.Id(stp).Dot("logger").Dot("Log").Call(
					loggerLog...,
				)).Call()
			} else {
				deferBlock = jen.Comment("Implement your middleware logic here").Line().Line()
			}
			g.code.appendFunction(
				m.Name,
				jen.Id(stp).Id(mdw),
				middlewareFuncParam,
				middlewareFuncResult,
				"",
				deferBlock,
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
	} else if !b {
		g.generateFirstTime = true
		f := jen.NewFile("endpoint")
		g.fs.WriteFile(g.filePath, f.GoString(), false)
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
	imp, err := g.getMissingImports(f.Imports, g.file)
	if err != nil {
		return err
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
	var stp string
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
		ctxN := "ctx"
		rpName := "response"
		rqName := "request"
		i := 0
		for _, p := range m.Parameters {
			if p.Name == rpName {
				rpName = rpName + fmt.Sprintf("%d", i)
				i++
			}
			if p.Name == rqName {
				rqName = rqName + fmt.Sprintf("%d", i)
				i++
			}
			tp := p.Type
			ts := strings.Split(tp, ".")
			if len(ts) == 1 {
				if tp[:1] == strings.ToUpper(tp[:1]) && tp[0] != '[' && tp[0] != '*' {
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
			} else {
				ctxN = p.Name
			}
		}
		rs := []jen.Code{}
		rt := []jen.Code{}
		for _, p := range m.Results {
			if p.Name == rpName {
				rpName = rpName + fmt.Sprintf("%d", i)
				i++
			}
			if p.Name == rqName {
				rqName = rqName + fmt.Sprintf("%d", i)
				i++
			}
			tp := p.Type
			ts := strings.Split(tp, ".")
			if len(ts) == 1 {
				if tp[:1] == strings.ToUpper(tp[:1]) && tp[0] != '[' && tp[0] != '*' {
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
				jen.Id(rpName).Dot("").Call(jen.Id(m.Name+"Response")).Dot(utils.ToCamelCase(p.Name)),
			)
		}

		body := []jen.Code{
			jen.Id(rqName).Op(":=").Id(m.Name + "Request").Values(req),
			jen.List(jen.Id(rpName), jen.Err()).Op(":=").Id(stp).Dot(m.Name + "Endpoint").Call(
				jen.List(jen.Id(ctxN), jen.Id(rqName)),
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
	errTypeFound := false
	for _, m := range g.serviceInterface.Methods {
		// For the request struct
		reqFields := []jen.Code{}
		// For the response struct
		resFields := []jen.Code{}

		mCallParam := []jen.Code{}
		respParam := jen.Dict{}
		retList := []jen.Code{}
		ctxN := "ctx"
		for _, p := range m.Parameters {
			if p.Type == "context.Context" {
				ctxN = p.Name
				mCallParam = append(mCallParam, jen.Id(p.Name))
				continue
			}
			tp := p.Type
			ts := strings.Split(tp, ".")
			if len(ts) == 1 {
				if tp[:1] == strings.ToUpper(tp[:1]) && tp[0] != '[' && tp[0] != '*' {
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
				reqFields = append(reqFields, jen.Id(utils.ToCamelCase(p.Name)).Id(strings.Replace(tp, "...", "[]", 1)).Tag(map[string]string{
					"json": utils.ToLowerSnakeCase(p.Name),
				}))
			}
			mCallParam = append(mCallParam, jen.Id("req").Dot(utils.ToCamelCase(p.Name)))

		}
		methodHasError := false
		errName := ""
		for _, p := range m.Results {
			if p.Type == "error" {
				errTypeFound = true
				methodHasError = true
				errName = utils.ToCamelCase(p.Name)
			}
			tp := p.Type
			ts := strings.Split(tp, ".")
			if len(ts) == 1 {
				if tp[:1] == strings.ToUpper(tp[:1]) && tp[0] != '[' && tp[0] != '*' {
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
		failedFound := false
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
			}
			if v.Name == "Failed" && v.Struct.Type == m.Name+"Response" {
				failedFound = true
			}
			if failedFound && makeMethdExists {
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
			bd := []jen.Code{
				jen.Id("req").Op(":=").Id("request").Dot("").Call(
					jen.Id(m.Name + "Request"),
				),
				jen.List(retList...).Op(":=").Id("s").Dot(m.Name).Call(mCallParam...),
				jen.Return(jen.Id(m.Name+"Response").Values(respParam), jen.Nil()),
			}
			if len(mCallParam) == 1 {
				bd = bd[1:]
			}
			pt.appendFunction(
				"",
				nil,
				[]jen.Code{
					jen.Id(ctxN).Qual("context", "Context"),
					jen.Id("request").Interface(),
				},
				[]jen.Code{
					jen.Interface(),
					jen.Error(),
				},
				"",
				bd...,
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
		if !failedFound && methodHasError {
			g.code.Raw().Comment("Failed implements Failer.").Line()
			g.code.appendFunction(
				"Failed",
				jen.Id("r").Id(m.Name+"Response"),
				[]jen.Code{},
				[]jen.Code{},
				"error",
				jen.Return(jen.Id("r").Dot(errName)),
			)
			g.code.NewLine()
		}
	}
	if errTypeFound {
		failureFound := false
		for _, v := range g.file.Interfaces {
			if v.Name == "Failure" {
				failureFound = true
			}
		}
		if !failureFound {
			g.code.appendMultilineComment(
				[]string{
					"Failure is an interface that should be implemented by response types.",
					"Response encoders can check if responses are Failer, and if so they've",
					"failed, and if so encode them using a separate write path based on the error.",
				},
			)
			g.code.NewLine()

			g.code.Raw().Type().Id("Failure").Interface(
				jen.Id("Failed").Params().Error(),
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
	} else if !b {
		g.generateFirstTime = true
		f := jen.NewFile("endpoint")
		g.fs.WriteFile(g.filePath, f.GoString(), false)
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

type generateCmdBase struct {
	BaseGenerator
	name                               string
	methods                            []string
	destPath                           string
	filePath                           string
	httpDestPath                       string
	grpcDestPath                       string
	httpFilePath                       string
	grpcFilePath                       string
	httpFile                           *parser.File
	grpcFile                           *parser.File
	generateSvcDefaultsMiddleware      bool
	generateEndpointDefaultsMiddleware bool
	serviceInterface                   parser.Interface
}

func newGenerateCmdBase(name string, serviceInterface parser.Interface,
	generateSacDefaultsMiddleware bool, generateEndpointDefaultsMiddleware bool, methods []string) Gen {
	t := &generateCmdBase{
		name:                               name,
		methods:                            methods,
		destPath:                           fmt.Sprintf(viper.GetString("gk_cmd_service_path_format"), utils.ToLowerSnakeCase(name)),
		serviceInterface:                   serviceInterface,
		generateSvcDefaultsMiddleware:      generateSacDefaultsMiddleware,
		httpDestPath:                       fmt.Sprintf(viper.GetString("gk_http_path_format"), utils.ToLowerSnakeCase(name)),
		grpcDestPath:                       fmt.Sprintf(viper.GetString("gk_grpc_path_format"), utils.ToLowerSnakeCase(name)),
		generateEndpointDefaultsMiddleware: generateEndpointDefaultsMiddleware,
	}
	t.filePath = path.Join(t.destPath, viper.GetString("gk_cmd_base_file_name"))
	t.httpFilePath = path.Join(t.httpDestPath, viper.GetString("gk_http_file_name"))
	t.grpcFilePath = path.Join(t.grpcDestPath, viper.GetString("gk_grpc_file_name"))
	t.srcFile = jen.NewFile("service")
	t.InitPg()
	t.fs = fs.Get()
	return t
}
func (g *generateCmdBase) Generate() (err error) {
	err = g.CreateFolderStructure(g.destPath)
	if err != nil {
		return err
	}
	g.srcFile.PackageComment("THIS FILE IS AUTO GENERATED BY GK-CLI DO NOT EDIT!!")
	endpointImport, err := utils.GetEndpointImportPath(g.name)
	if err != nil {
		return err
	}
	serviceImport, err := utils.GetServiceImportPath(g.name)
	if err != nil {
		return err
	}
	httpImport, err := utils.GetHTTPTransportImportPath(g.name)
	if err != nil {
		return err
	}
	existingHTTP := false
	if b, err := g.fs.Exists(g.httpFilePath); err != nil {
		return err
	} else if b {
		existingHTTP = true
	}
	existingGRPC := false
	if b, err := g.fs.Exists(g.grpcFilePath); err != nil {
		return err
	} else if b {
		existingGRPC = true
	}
	cd := []jen.Code{
		jen.Id("g").Op("=").Id("&").Qual(
			"github.com/oklog/oklog/pkg/group", "Group",
		).Block(),
	}
	if existingHTTP {
		src, err := g.fs.ReadFile(g.httpFilePath)
		if err != nil {
			return err
		}
		g.httpFile, err = parser.NewFileParser().Parse([]byte(src))
		if err != nil {
			return err
		}
		cd = append(cd, jen.Id("initHttpHandler").Call(jen.Id("endpoints"), jen.Id("g")))
	}
	if existingGRPC {
		src, err := g.fs.ReadFile(g.grpcFilePath)
		if err != nil {
			return err
		}
		g.grpcFile, err = parser.NewFileParser().Parse([]byte(src))
		if err != nil {
			return err
		}
		cd = append(cd, jen.Id("initGRPCHandler").Call(jen.Id("endpoints"), jen.Id("g")))
	}
	cd = append(cd, jen.Return(jen.Id("g")))
	g.code.appendFunction(
		"createService",
		nil,
		[]jen.Code{
			jen.Id("endpoints").Qual(endpointImport, "Endpoints"),
		},
		[]jen.Code{
			jen.Id("g").Id("*").Qual("github.com/oklog/oklog/pkg/group", "Group"),
		},
		"",
		cd...,
	)

	g.code.NewLine()
	if existingHTTP {
		opt := jen.Dict{}
		for _, v := range g.serviceInterface.Methods {
			for _, m := range g.httpFile.Methods {
				if m.Name == "make"+v.Name+"Handler" {
					methodHasError := false
					for _, p := range append(v.Parameters, v.Results...) {
						if p.Type == "error" {
							methodHasError = true
						}
					}
					pt := []jen.Code{}
					if methodHasError {
						pt = append(
							pt,
							jen.Qual("github.com/go-kit/kit/transport/http", "ServerErrorEncoder").Call(
								jen.Qual(httpImport, "ErrorEncoder"),
							),
						)
					}

					pt = append(
						pt,
						jen.Qual("github.com/go-kit/kit/transport/http", "ServerErrorLogger").Call(jen.Id("logger")),
						jen.Qual("github.com/go-kit/kit/transport/http", "ServerBefore").Call(
							jen.Qual("github.com/go-kit/kit/tracing/opentracing", "HTTPToContext").Call(
								jen.Id("tracer"),
								jen.Lit(v.Name),
								jen.Id("logger"),
							),
						),
					)
					opt[jen.Lit(v.Name)] =
						jen.Values(
							jen.List(
								pt...,
							),
						)
				}
			}
		}
		pl := NewPartialGenerator(nil)
		pl.Raw().Id("options").Op(":=").Map(jen.String()).Index().Qual(
			"github.com/go-kit/kit/transport/http",
			"ServerOption",
		).Values(
			opt,
		).Line()
		pl.Raw().Return(jen.Id("options"))
		g.code.appendFunction(
			"defaultHttpOptions",
			nil,
			[]jen.Code{
				jen.Id("logger").Qual("github.com/go-kit/kit/log", "Logger"),
				jen.Id("tracer").Qual("github.com/opentracing/opentracing-go", "Tracer"),
			},
			[]jen.Code{
				jen.Map(jen.String()).Index().Qual("github.com/go-kit/kit/transport/http", "ServerOption"),
			},
			"",
			pl.Raw(),
		)
		g.code.NewLine()
	}
	if existingGRPC {
		opt := jen.Dict{}
		for _, v := range g.serviceInterface.Methods {
			for _, m := range g.grpcFile.Methods {
				if m.Name == "make"+v.Name+"Handler" {
					opt[jen.Lit(v.Name)] =
						jen.Values(
							jen.List(
								jen.Qual("github.com/go-kit/kit/transport/grpc", "ServerErrorLogger").Call(jen.Id("logger")),
								jen.Qual("github.com/go-kit/kit/transport/grpc", "ServerBefore").Call(
									jen.Qual("github.com/go-kit/kit/tracing/opentracing", "GRPCToContext").Call(
										jen.Id("tracer"),
										jen.Lit(v.Name),
										jen.Id("logger"),
									),
								),
							),
						)
				}
			}
		}
		pl := NewPartialGenerator(nil)
		pl.Raw().Id("options").Op(":=").Map(jen.String()).Index().Qual(
			"github.com/go-kit/kit/transport/grpc",
			"ServerOption",
		).Values(
			opt,
		).Line()
		pl.Raw().Return(jen.Id("options"))
		g.code.appendFunction(
			"defaultGRPCOptions",
			nil,
			[]jen.Code{
				jen.Id("logger").Qual("github.com/go-kit/kit/log", "Logger"),
				jen.Id("tracer").Qual("github.com/opentracing/opentracing-go", "Tracer"),
			},
			[]jen.Code{
				jen.Map(jen.String()).Index().Qual("github.com/go-kit/kit/transport/grpc", "ServerOption"),
			},
			"",
			pl.Raw(),
		)
		g.code.NewLine()
	}
	if g.generateEndpointDefaultsMiddleware {
		body := []jen.Code{}
		mdw := map[string][]jen.Code{}
		for _, m := range g.serviceInterface.Methods {
			if mdw[m.Name] == nil {
				mdw[m.Name] = []jen.Code{}
			}
			mdw[m.Name] = append(
				mdw[m.Name],
				jen.Qual(endpointImport, "LoggingMiddleware").Call(
					jen.Id("log").Dot("With").Call(
						jen.Id("logger"),
						jen.Lit("method"),
						jen.Lit(m.Name),
					)),
			)
			mdw[m.Name] = append(
				mdw[m.Name],
				jen.Qual(endpointImport, "InstrumentingMiddleware").Call(
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
		g.code.appendFunction(
			"addDefaultEndpointMiddleware",
			nil,
			[]jen.Code{
				jen.Id("logger").Qual("github.com/go-kit/kit/log", "Logger"),
				jen.Id("duration").Id("*").Qual("github.com/go-kit/kit/metrics/prometheus", "Summary"),
				jen.Id("mw").Map(jen.String()).Index().Qual("github.com/go-kit/kit/endpoint", "Middleware"),
			},
			[]jen.Code{},
			"",
			body...,
		)
		g.code.NewLine()
	}
	if g.generateSvcDefaultsMiddleware {
		g.code.appendFunction(
			"addDefaultServiceMiddleware",
			nil,
			[]jen.Code{
				jen.Id("logger").Qual("github.com/go-kit/kit/log", "Logger"),
				jen.Id("mw").Index().Qual(serviceImport, "Middleware"),
			},
			[]jen.Code{
				jen.Index().Qual(serviceImport, "Middleware"),
			},
			"",
			jen.Return(
				jen.Append(jen.Id("mw"), jen.Qual(serviceImport, "LoggingMiddleware").Call(jen.Id("logger"))),
			),
		)
		g.code.NewLine()
	}
	mth := []jen.Code{}
	for _, v := range g.serviceInterface.Methods {
		mth = append(mth, jen.Lit(v.Name))
	}
	g.code.appendFunction(
		"addEndpointMiddlewareToAllMethods",
		nil,
		[]jen.Code{
			jen.Id("mw").Map(jen.String()).Index().Qual("github.com/go-kit/kit/endpoint", "Middleware"),
			jen.Id("m").Qual("github.com/go-kit/kit/endpoint", "Middleware"),
		},
		[]jen.Code{},
		"",
		jen.Id("methods").Op(":=").Index().String().Values(mth...),
		jen.For(jen.List(jen.Id("_"), jen.Id("v")).Op(":=").Range().Id("methods")).Block(
			jen.Id("mw").Index(jen.Id("v")).Op("=").Append(jen.Id("mw").Index(jen.Id("v")), jen.Id("m")),
		),
	)
	g.code.NewLine()
	return g.fs.WriteFile(g.filePath, g.srcFile.GoString(), true)
}

type generateCmd struct {
	BaseGenerator
	name                               string
	methods                            []string
	generateFirstTime                  bool
	file                               *parser.File
	interfaceName                      string
	destPath                           string
	filePath                           string
	httpDestPath                       string
	grpcDestPath                       string
	httpFilePath                       string
	grpcFilePath                       string
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
		destPath:                           fmt.Sprintf(viper.GetString("gk_cmd_service_path_format"), utils.ToLowerSnakeCase(name)),
		httpDestPath:                       fmt.Sprintf(viper.GetString("gk_http_path_format"), utils.ToLowerSnakeCase(name)),
		grpcDestPath:                       fmt.Sprintf(viper.GetString("gk_grpc_path_format"), utils.ToLowerSnakeCase(name)),
		serviceInterface:                   serviceInterface,
		generateSvcDefaultsMiddleware:      generateSacDefaultsMiddleware,
		generateEndpointDefaultsMiddleware: generateEndpointDefaultsMiddleware,
	}
	t.filePath = path.Join(t.destPath, viper.GetString("gk_cmd_svc_file_name"))
	t.httpFilePath = path.Join(t.httpDestPath, viper.GetString("gk_http_file_name"))
	t.grpcFilePath = path.Join(t.grpcDestPath, viper.GetString("gk_grpc_file_name"))
	t.srcFile = jen.NewFile("service")
	t.InitPg()
	t.fs = fs.Get()
	return t
}

// Generate generates the service main.
func (g *generateCmd) Generate() (err error) {
	err = g.CreateFolderStructure(g.destPath)
	if err != nil {
		return err
	}
	if b, err := g.fs.Exists(g.filePath); err != nil {
		return err
	} else if !b {
		g.generateFirstTime = true
		f := jen.NewFile("service")
		g.fs.WriteFile(g.filePath, f.GoString(), false)
	}
	src, err := g.fs.ReadFile(g.filePath)
	if err != nil {
		return err
	}
	g.file, err = parser.NewFileParser().Parse([]byte(src))
	if err != nil {
		return err
	}
	g.generateVars()
	runFound := false
	for _, v := range g.file.Methods {
		if v.Name == "Run" {
			runFound = true
		}
	}
	if !runFound {
		p, err := g.generateRun()
		if err != nil {
			return err
		}
		g.code.appendFunction(
			"Run",
			nil,
			[]jen.Code{},
			[]jen.Code{},
			"",
			p.Raw(),
		)
	}
	if b, err := g.fs.Exists(g.httpFilePath); err != nil {
		return err
	} else if b {
		err = g.generateInitHTTP()
		if err != nil {
			return err
		}
	}
	if b, err := g.fs.Exists(g.grpcFilePath); err != nil {
		return err
	} else if b {
		err = g.generateInitGRPC()
		if err != nil {
			return err
		}
	}
	err = g.generateGetMiddleware()
	if err != nil {
		return err
	}
	g.generateDefaultMetrics()
	g.generateCancelInterrupt()
	g.generateCmdMain()
	if g.generateFirstTime {
		return g.fs.WriteFile(g.filePath, g.srcFile.GoString(), true)
	}
	tmpSrc := g.srcFile.GoString()
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
		pSrc := g.code.Raw().GoString()
		foundSameImport := false
		inx := 0
		// Small(stupid) workaround
		txt := "abcdefghijkl"
		mp := map[string]string{}
		keep := imp
		for a, i := range imp {
			for _, v := range g.file.Imports {
				if v.Type == i.Type && i.Name != v.Name {
					mp[txt+i.Name] = v.Name
					pSrc = strings.Replace(pSrc, i.Name+".", txt+i.Name+".", -1)
					keep = append(imp[:a], imp[a+1:]...)
				}
			}
		}

		for a, i := range keep {
			for _, v := range g.file.Imports {
				if v.Name == i.Name {
					foundSameImport = true
					inx = a
				}
			}
		}
		oldName := keep[inx].Name
		if foundSameImport {
			a := 1
			for {
				canUse := true
				for _, v := range g.file.Imports {
					if fmt.Sprintf("%s%d", keep[inx].Name, a) == v.Name {
						canUse = false
						break
					}
				}
				if canUse {
					keep[inx].Name = fmt.Sprintf("%s%d", keep[inx].Name, a)
					break
				}
				a++
			}
			pSrc = strings.Replace(pSrc, oldName+".", keep[inx].Name+".", -1)
			for k, v := range mp {
				pSrc = strings.Replace(pSrc, k+".", v+".", -1)
			}
			src += "\n" + pSrc
		} else {
			for k, v := range mp {
				pSrc = strings.Replace(pSrc, k+".", v+".", -1)
			}
			src += "\n" + pSrc
		}
		src, err = g.AddImportsToFile(keep, src)
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
func (g *generateCmd) generateRun() (*PartialGenerator, error) {
	pg := NewPartialGenerator(nil)
	pg.Raw().Id("fs").Dot("Parse").Call(jen.Qual("os", "Args").Index(jen.Lit(1), jen.Empty()))
	pg.Raw().Line().Line().Comment("Create a single logger, which we'll use and give to other components.").Line()
	pg.Raw().Id("logger").Op("=").Qual("github.com/go-kit/kit/log", "NewLogfmtLogger").Call(
		jen.Qual("os", "Stderr"),
	).Line()
	pg.Raw().Id("logger").Op("=").Qual("github.com/go-kit/kit/log", "With").Call(
		jen.Id("logger"),
		jen.Lit("ts"),
		jen.Qual("github.com/go-kit/kit/log", "DefaultTimestampUTC"),
	).Line()
	pg.Raw().Id("logger").Op("=").Qual("github.com/go-kit/kit/log", "With").Call(
		jen.Id("logger"),
		jen.Lit("caller"),
		jen.Qual("github.com/go-kit/kit/log", "DefaultCaller"),
	).Line().Line()
	pg.appendMultilineComment(
		[]string{
			" Determine which tracer to use. We'll pass the tracer to all the",
			"components that use it, as a dependency",
		},
	)
	pg.NewLine()
	pg.Raw().If(
		jen.Id("*zipkinURL").Op("!=").Lit(""),
	).Block(
		jen.Id("logger").Dot("Log").Call(
			jen.Lit("tracer"),
			jen.Lit("Zipkin"),
			jen.Lit("URL"),
			jen.Id("*zipkinURL"),
		),
		jen.Id("reporter").Op(":=").Qual(
			"github.com/openzipkin/zipkin-go/reporter/http", "NewReporter",
		).Call(jen.Id("*zipkinURL")),
		jen.Defer().Id("reporter").Dot("Close").Call(),
		jen.List(jen.Id("endpoint"), jen.Id("err")).Op(":=").Qual(
			"github.com/openzipkin/zipkin-go", "NewEndpoint",
		).Call(
			jen.Lit(g.name),
			jen.Lit("localhost:80"),
		),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Id("logger").Dot("Log").Call(
				jen.Lit("err"),
				jen.Id("err"),
			),
			jen.Qual("os", "Exit").Call(jen.Lit(1)),
		),
		jen.Id("localEndpoint").Op(":=").Qual("github.com/openzipkin/zipkin-go", "WithLocalEndpoint").Call(jen.Id("endpoint")),
		jen.List(jen.Id("nativeTracer"), jen.Id("err")).Op(":=").Qual(
			"github.com/openzipkin/zipkin-go", "NewTracer",
		).Call(jen.Id("reporter"), jen.Id("localEndpoint")),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Id("logger").Dot("Log").Call(
				jen.Lit("err"),
				jen.Id("err"),
			),
			jen.Qual("os", "Exit").Call(jen.Lit(1)),
		),
		jen.Id("tracer").Op("=").Qual(
			"github.com/openzipkin-contrib/zipkin-go-opentracing", "Wrap",
		).Call(
			jen.Id("nativeTracer"),
		),
	).Else().If(jen.Id("*lightstepToken").Op("!=").Lit("")).Block(
		jen.Id("logger").Dot("Log").Call(
			jen.Lit("tracer"),
			jen.Lit("LightStep"),
		),
		jen.Id("tracer").Op("=").Qual(
			"github.com/lightstep/lightstep-tracer-go", "NewTracer",
		).Call(jen.Qual(
			"github.com/lightstep/lightstep-tracer-go", "Options",
		).Values(
			jen.Dict{
				jen.Id("AccessToken"): jen.Id("*lightstepToken"),
			},
		),
		),
		jen.Defer().Qual(
			"github.com/lightstep/lightstep-tracer-go", "Flush",
		).Call(jen.Qual("context","Background").Call(), jen.Id("tracer")),
	).Else().If(jen.Id("*appdashAddr").Op("!=").Lit("")).Block(
		jen.Id("logger").Dot("Log").Call(
			jen.Lit("tracer"),
			jen.Lit("Appdash"),
			jen.Lit("addr"),
			jen.Id("*appdashAddr"),
		),
		jen.Id("collector").Op(":=").Qual(
			"sourcegraph.com/sourcegraph/appdash", "NewRemoteCollector",
		).Call(jen.Id("*appdashAddr")),
		jen.Id("tracer").Op("=").Qual(
			"sourcegraph.com/sourcegraph/appdash/opentracing", "NewTracer",
		).Call(jen.Id("collector")),
		jen.Defer().Id("collector").Dot("Close").Call(),
	).Else().Block(
		jen.Id("logger").Dot("Log").Call(
			jen.Lit("tracer"),
			jen.Lit("none"),
		),
		jen.Id("tracer").Op("=").Qual(
			"github.com/opentracing/opentracing-go", "GlobalTracer",
		).Call(),
	).Line().Line()

	svcImport, err := utils.GetServiceImportPath(g.name)
	if err != nil {
		return nil, err
	}
	epImport, err := utils.GetEndpointImportPath(g.name)
	if err != nil {
		return nil, err
	}
	pg.Raw().Id("svc").Op(":=").Qual(svcImport, "New").Call(
		jen.Id("getServiceMiddleware").Call(jen.Id("logger")),
	).Line()
	pg.Raw().Id("eps").Op(":=").Qual(epImport, "New").Call(
		jen.Id("svc"),
		jen.Id("getEndpointMiddleware").Call(jen.Id("logger")),
	).Line()
	pg.Raw().Id("g").Op(":=").Id("createService").Call(
		jen.Id("eps"),
	).Line()
	pg.Raw().Id("initMetricsEndpoint").Call(jen.Id("g")).Line()
	pg.Raw().Id("initCancelInterrupt").Call(jen.Id("g")).Line()
	pg.Raw().Id("logger").Dot("Log").Call(
		jen.Lit("exit"),
		jen.Id("g").Dot("Run").Call(),
	).Line()
	return pg, nil
}
func (g *generateCmd) generateVars() {
	if g.generateFirstTime {
		g.code.Raw().Var().Id("tracer").Qual("github.com/opentracing/opentracing-go", "Tracer").Line()
		g.code.Raw().Var().Id("logger").Qual("github.com/go-kit/kit/log", "Logger").Line()
		g.code.appendMultilineComment(
			[]string{
				"Define our flags. Your service probably won't need to bind listeners for",
				"all* supported transports, but we do it here for demonstration purposes.",
			},
		)
		g.code.NewLine()
		g.code.Raw().Var().Id("fs").Op("=").Qual("flag", "NewFlagSet").Call(
			jen.Lit(g.name), jen.Qual("flag", "ExitOnError"),
		)
		g.code.NewLine()
		g.code.Raw().Var().Id("debugAddr").Op("=").Id("fs").Dot("String").Call(
			jen.Lit("debug.addr"),
			jen.Lit(":8080"),
			jen.Lit("Debug and metrics listen address"),
		)
		g.code.NewLine()
		g.code.Raw().Var().Id("httpAddr").Op("=").Id("fs").Dot("String").Call(
			jen.Lit("http-addr"),
			jen.Lit(":8081"),
			jen.Lit("HTTP listen address"),
		)
		g.code.NewLine()
		g.code.Raw().Var().Id("grpcAddr").Op("=").Id("fs").Dot("String").Call(
			jen.Lit("grpc-addr"),
			jen.Lit(":8082"),
			jen.Lit("gRPC listen address"),
		)
		g.code.NewLine()
		g.code.Raw().Var().Id("thriftAddr").Op("=").Id("fs").Dot("String").Call(
			jen.Lit("thrift-addr"),
			jen.Lit(":8083"),
			jen.Lit("Thrift listen address"),
		)
		g.code.NewLine()
		g.code.Raw().Var().Id("thriftProtocol").Op("=").Id("fs").Dot("String").Call(
			jen.Lit("thrift-protocol"),
			jen.Lit("binary"),
			jen.Lit("binary, compact, json, simplejson"),
		)
		g.code.NewLine()
		g.code.Raw().Var().Id("thriftBuffer").Op("=").Id("fs").Dot("Int").Call(
			jen.Lit("thrift-buffer"),
			jen.Lit(0),
			jen.Lit("0 for unbuffered"),
		)
		g.code.NewLine()
		g.code.Raw().Var().Id("thriftFramed").Op("=").Id("fs").Dot("Bool").Call(
			jen.Lit("thrift-framed"),
			jen.Lit(false),
			jen.Lit("true to enable framing"),
		)
		g.code.NewLine()
		g.code.Raw().Var().Id("zipkinURL").Op("=").Id("fs").Dot("String").Call(
			jen.Lit("zipkin-url"),
			jen.Lit(""),
			jen.Lit("Enable Zipkin tracing via a collector URL e.g. http://localhost:9411/api/v1/spans"),
		)
		g.code.NewLine()
		g.code.Raw().Var().Id("lightstepToken").Op("=").Id("fs").Dot("String").Call(
			jen.Lit("lightstep-token"),
			jen.Lit(""),
			jen.Lit("Enable LightStep tracing via a LightStep access token"),
		)
		g.code.NewLine()
		g.code.Raw().Var().Id("appdashAddr").Op("=").Id("fs").Dot("String").Call(
			jen.Lit("appdash-addr"),
			jen.Lit(""),
			jen.Lit("Enable Appdash tracing via an Appdash server host:port"),
		)
		g.code.NewLine()
	}
}
func (g *generateCmd) generateInitHTTP() (err error) {
	for _, v := range g.file.Methods {
		if v.Name == "initHttpHandler" {
			return
		}
	}
	httpImport, err := utils.GetHTTPTransportImportPath(g.name)
	if err != nil {
		return err
	}

	epImport, err := utils.GetEndpointImportPath(g.name)
	if err != nil {
		return err
	}

	pt := NewPartialGenerator(nil)
	pt.Raw().Id("options").Op(":=").Id("defaultHttpOptions").Call(
		jen.Id("logger"),
		jen.Id("tracer"),
	).Line().Comment("Add your http options here").Line().Line()
	pt.Raw().Id("httpHandler").Op(":=").Qual(httpImport, "NewHTTPHandler").Call(
		jen.Id("endpoints"),
		jen.Id("options"),
	).Line()

	pt.Raw().List(jen.Id("httpListener"), jen.Err()).Op(":=").Qual("net", "Listen").Call(
		jen.Lit("tcp"),
		jen.Id("*httpAddr"),
	).Line()
	pt.Raw().If(
		jen.Err().Op("!=").Nil().Block(
			jen.Id("logger").Dot("Log").Call(
				jen.Lit("transport"),
				jen.Lit("HTTP"),
				jen.Lit("during"),
				jen.Lit("Listen"),
				jen.Lit("err"),
				jen.Err(),
			),
		),
	).Line()
	pt.Raw().Id("g").Dot("Add").Call(
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
	).Line()
	g.code.NewLine()
	g.code.appendFunction(
		"initHttpHandler",
		nil,
		[]jen.Code{
			jen.Id("endpoints").Qual(epImport, "Endpoints"),
			jen.Id("g").Id("*").Qual("github.com/oklog/oklog/pkg/group", "Group"),
		},
		[]jen.Code{},
		"",
		pt.Raw(),
	)
	return
}
func (g *generateCmd) generateInitGRPC() (err error) {
	for _, v := range g.file.Methods {
		if v.Name == "initGRPCHandler" {
			return
		}
	}
	grpcImport, err := utils.GetGRPCTransportImportPath(g.name)
	if err != nil {
		return err
	}
	pbImport, err := utils.GetPbImportPath(g.name)
	if err != nil {
		return err
	}

	epImport, err := utils.GetEndpointImportPath(g.name)
	if err != nil {
		return err
	}

	pt := NewPartialGenerator(nil)
	pt.Raw().Id("options").Op(":=").Id("defaultGRPCOptions").Call(
		jen.Id("logger"),
		jen.Id("tracer"),
	).Line().Comment("Add your GRPC options here").Line().Line()
	pt.Raw().Id("grpcServer").Op(":=").Qual(grpcImport, "NewGRPCServer").Call(
		jen.Id("endpoints"),
		jen.Id("options"),
	).Line()

	pt.Raw().List(jen.Id("grpcListener"), jen.Err()).Op(":=").Qual("net", "Listen").Call(
		jen.Lit("tcp"),
		jen.Id("*grpcAddr"),
	).Line()
	pt.Raw().If(
		jen.Err().Op("!=").Nil().Block(
			jen.Id("logger").Dot("Log").Call(
				jen.Lit("transport"),
				jen.Lit("gRPC"),
				jen.Lit("during"),
				jen.Lit("Listen"),
				jen.Lit("err"),
				jen.Err(),
			),
		),
	).Line()
	pt.Raw().Id("g").Dot("Add").Call(
		jen.Func().Params().Error().Block(
			jen.Id("logger").Dot("Log").Call(
				jen.Lit("transport"),
				jen.Lit("gRPC"),
				jen.Lit("addr"),
				jen.Id("*grpcAddr"),
			),
			jen.Id("baseServer").Op(":=").Qual("google.golang.org/grpc", "NewServer").Call(),
			jen.Qual(pbImport, fmt.Sprintf("Register%sServer", utils.ToCamelCase(g.name))).Call(
				jen.Id("baseServer"),
				jen.Id("grpcServer"),
			),
			jen.Return(
				jen.Id("baseServer").Dot("Serve").Call(
					jen.Id("grpcListener"),
				),
			),
		),
		jen.Func().Params(jen.Error()).Block(
			jen.Id("grpcListener").Dot("Close").Call(),
		),
	).Line()
	g.code.NewLine()
	g.code.appendFunction(
		"initGRPCHandler",
		nil,
		[]jen.Code{
			jen.Id("endpoints").Qual(epImport, "Endpoints"),
			jen.Id("g").Id("*").Qual("github.com/oklog/oklog/pkg/group", "Group"),
		},
		[]jen.Code{},
		"",
		pt.Raw(),
	)
	return
}
func (g *generateCmd) generateGetMiddleware() (err error) {
	for _, v := range g.file.Methods {
		if v.Name == "getServiceMiddleware" {
			return
		}
	}
	svcImport, err := utils.GetServiceImportPath(g.name)
	if err != nil {
		return err
	}
	c := []jen.Code{
		jen.Id("mw").Op("=").Index().Qual(svcImport, "Middleware").Block(),
	}
	if g.generateSvcDefaultsMiddleware {
		c = append(
			c,
			jen.Id("mw").Op("=").Id("addDefaultServiceMiddleware").Call(
				jen.Id("logger"),
				jen.Id("mw"),
			),
		)
	}
	c = append(c, jen.Comment("Append your middleware here").Line(), jen.Return())
	g.code.NewLine()
	g.code.appendFunction(
		"getServiceMiddleware",
		nil,
		[]jen.Code{
			jen.Id("logger").Qual("github.com/go-kit/kit/log", "Logger"),
		},
		[]jen.Code{
			jen.Id("mw").Index().Qual(svcImport, "Middleware"),
		},
		"",
		c...,
	)
	g.code.NewLine()
	c = []jen.Code{
		jen.Id("mw").Op("=").Map(jen.String()).Index().Qual(
			"github.com/go-kit/kit/endpoint",
			"Middleware",
		).Block(),
	}
	if g.generateEndpointDefaultsMiddleware {
		c = append(
			c,
			jen.Id("duration").Op(":=").Qual("github.com/go-kit/kit/metrics/prometheus", "NewSummaryFrom").Call(
				jen.Qual("github.com/prometheus/client_golang/prometheus", "SummaryOpts").Values(
					jen.Dict{
						jen.Id("Help"):      jen.Lit("Request duration in seconds."),
						jen.Id("Name"):      jen.Lit("request_duration_seconds"),
						jen.Id("Namespace"): jen.Lit("example"),
						jen.Id("Subsystem"): jen.Lit(g.name),
					},
				),
				jen.Index().String().Values(jen.Lit("method"), jen.Lit("success")),
			),
			jen.Id("addDefaultEndpointMiddleware").Call(
				jen.Id("logger"), jen.Id("duration"), jen.Id("mw"),
			),
		)
	}
	c = append(
		c,
		jen.Comment("Add you endpoint middleware here").Line(),
		jen.Return(),
	)
	g.code.appendFunction(
		"getEndpointMiddleware",
		nil,
		[]jen.Code{
			jen.Id("logger").Qual("github.com/go-kit/kit/log", "Logger"),
		},
		[]jen.Code{
			jen.Id("mw").Map(jen.String()).Index().Qual(
				"github.com/go-kit/kit/endpoint",
				"Middleware",
			),
		},
		"",
		c...,
	)
	return
}
func (g *generateCmd) generateDefaultMetrics() {
	if g.generateFirstTime {
		g.code.NewLine()
		g.code.appendFunction(
			"initMetricsEndpoint",
			nil,
			[]jen.Code{
				jen.Id("g").Id("*").Qual("github.com/oklog/oklog/pkg/group", "Group"),
			},
			[]jen.Code{},
			"",
			jen.Qual("net/http", "DefaultServeMux").Dot("Handle").Call(
				jen.Lit("/metrics"),
				jen.Qual("github.com/prometheus/client_golang/prometheus/promhttp", "Handler").Call(),
			),
			jen.List(jen.Id("debugListener"), jen.Err()).Op(":=").Qual("net", "Listen").Call(
				jen.Lit("tcp"),
				jen.Id("*debugAddr"),
			),
			jen.If(
				jen.Err().Op("!=").Nil().Block(
					jen.Id("logger").Dot("Log").Call(
						jen.Lit("transport"),
						jen.Lit("debug/HTTP"),
						jen.Lit("during"),
						jen.Lit("Listen"),
						jen.Lit("err"),
						jen.Err(),
					),
				),
			),
			jen.Id("g").Dot("Add").Call(
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
		)
	}
}
func (g *generateCmd) generateCancelInterrupt() {
	if g.generateFirstTime {
		g.code.NewLine()
		g.code.appendFunction(
			"initCancelInterrupt",
			nil,
			[]jen.Code{
				jen.Id("g").Id("*").Qual("github.com/oklog/oklog/pkg/group", "Group"),
			},
			[]jen.Code{},
			"",
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
	}
}
func (g *generateCmd) generateCmdMain() error {
	mainDest := fmt.Sprintf(viper.GetString("gk_cmd_path_format"), g.name)
	mainFilePath := path.Join(mainDest, "main.go")
	g.CreateFolderStructure(mainDest)
	if b, err := g.fs.Exists(mainFilePath); err != nil {
		return err
	} else if b {
		return nil
	}
	cmdSvcImport, err := utils.GetCmdServiceImportPath(g.name)
	if err != nil {
		return err
	}
	src := jen.NewFile("main")
	src.Func().Id("main").Params().Block(
		jen.Qual(cmdSvcImport, "Run").Call(),
	)
	return g.fs.WriteFile(mainFilePath, src.GoString(), false)
}
