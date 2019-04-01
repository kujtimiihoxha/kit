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

// GenerateMiddleware implements Gen and is used to generate middleware.
type GenerateMiddleware struct {
	BaseGenerator
	name                 string
	serviceName          string
	interfaceName        string
	generateFirstTime    bool
	destPath             string
	filePath             string
	serviceFile          *parser.File
	isEndpointMiddleware bool
	file                 *parser.File
	serviceInterface     parser.Interface
	generateDefaults     bool
	serviceGenerator     *generateServiceMiddleware
}

// NewGenerateMiddleware returns a initialized and ready generator.
func NewGenerateMiddleware(name, serviceName string, ep bool) Gen {
	i := &GenerateMiddleware{
		name:                 name,
		serviceName:          serviceName,
		isEndpointMiddleware: ep,
		interfaceName:        utils.ToCamelCase(serviceName + "Service"),
		destPath:             fmt.Sprintf(viper.GetString("gk_service_path_format"), utils.ToLowerSnakeCase(serviceName)),
	}
	i.filePath = path.Join(i.destPath, viper.GetString("gk_service_file_name"))
	i.fs = fs.Get()
	return i
}

// Generate generates a new service middleware
func (g *GenerateMiddleware) Generate() (err error) {
	if b, err := g.fs.Exists(g.filePath); err != nil {
		return err
	} else if !b {
		logrus.Errorf("Service %s was not found", g.serviceName)
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
	gi := newGenerateServiceMiddleware(g.serviceName, g.file, g.serviceInterface, false)
	g.serviceGenerator = gi.(*generateServiceMiddleware)
	if g.isEndpointMiddleware {
		g.destPath = fmt.Sprintf(viper.GetString("gk_endpoint_path_format"), utils.ToLowerSnakeCase(g.serviceName))
		g.filePath = path.Join(g.destPath, viper.GetString("gk_endpoint_middleware_file_name"))
		return g.generateEndpointMiddleware()
	}
	return g.generateServiceMiddleware()
}

func (g *GenerateMiddleware) generateServiceMiddleware() (err error) {
	err = g.serviceGenerator.Generate()
	if err != nil {
		return err
	}
	middlewareStructFound := false
	for _, v := range g.serviceGenerator.file.Structures {
		if v.Name == g.name+"Middleware" {
			middlewareStructFound = true
		}
	}
	mdwStrucName := utils.ToLowerFirstCamelCase(g.name) + "Middleware"
	if !middlewareStructFound {
		g.serviceGenerator.code.appendStruct(
			mdwStrucName,
			jen.Id("next").Id(g.interfaceName),
		)
	}
	mthdFound := false
	for _, v := range g.serviceGenerator.file.Methods {
		if v.Name == utils.ToCamelCase(g.name)+"Middleware" {
			mthdFound = true
			break
		}
	}
	if !mthdFound {
		g.serviceGenerator.code.appendMultilineComment([]string{
			fmt.Sprintf(
				"%s returns a %s Middleware.",
				utils.ToCamelCase(g.name)+"Middleware",
				g.interfaceName,
			),
		})
		g.serviceGenerator.code.NewLine()
		pt := NewPartialGenerator(nil)
		pt.appendFunction(
			"",
			nil,
			[]jen.Code{
				jen.Id("next").Id(g.interfaceName),
			},
			[]jen.Code{},
			g.interfaceName,
			jen.Return(jen.Id("&"+mdwStrucName).Values(jen.Id("next"))),
		)
		pt.NewLine()
		g.serviceGenerator.code.appendFunction(
			utils.ToCamelCase(g.name)+"Middleware",
			nil,
			[]jen.Code{},
			[]jen.Code{},
			"Middleware",
			jen.Return(pt.Raw()),
		)
		g.serviceGenerator.code.NewLine()
	}
	g.serviceGenerator.generateMethodMiddleware(mdwStrucName, false)
	if g.serviceGenerator.generateFirstTime {
		return g.fs.WriteFile(g.serviceGenerator.filePath, g.serviceGenerator.srcFile.GoString(), true)
	}
	src, err := g.fs.ReadFile(g.serviceGenerator.filePath)
	if err != nil {
		return err
	}
	src += "\n" + g.serviceGenerator.code.Raw().GoString()
	tmpSrc := g.serviceGenerator.srcFile.GoString()
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
		src, err = g.serviceGenerator.AddImportsToFile(imp, src)
		if err != nil {
			return err
		}
	}
	s, err := utils.GoImportsSource(g.serviceGenerator.destPath, src)
	if err != nil {
		return err
	}
	return g.fs.WriteFile(g.serviceGenerator.filePath, s, true)
}
func (g *GenerateMiddleware) generateEndpointMiddleware() (err error) {
	g.srcFile = jen.NewFilePath("endpoint")
	g.InitPg()
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
	middlewareFound := false
	for _, v := range g.file.Methods {
		if v.Name == utils.ToCamelCase(g.name)+"Middleware" {
			middlewareFound = true
		}
	}
	if !middlewareFound {
		g.code.appendMultilineComment([]string{
			fmt.Sprintf("%s returns an endpoint middleware", utils.ToCamelCase(g.name)+"Middleware"),
		})
		g.code.NewLine()
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
			jen.Comment("Add your middleware logic here"),
			jen.Return(jen.Id("next").Call(jen.Id("ctx"), jen.Id("request"))),
		)
		g.code.appendFunction(
			utils.ToCamelCase(g.name)+"Middleware",
			nil,
			[]jen.Code{},
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
func (g *GenerateMiddleware) serviceFound() bool {
	for n, v := range g.file.Interfaces {
		if v.Name == g.interfaceName {
			g.serviceInterface = v
			return true
		} else if n == len(g.file.Interfaces)-1 {
			logrus.Errorf("Could not find the service interface in `%s`", g.serviceName)
			return false
		}
	}
	return false
}
func (g *GenerateMiddleware) removeBadMethods() {
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
