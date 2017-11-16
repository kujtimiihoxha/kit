package generator

import (
	"fmt"
	"path"

	"github.com/dave/jennifer/jen"
	"github.com/kujtimiihoxha/kit/fs"
	"github.com/kujtimiihoxha/kit/parser"
	"github.com/kujtimiihoxha/kit/utils"
	"github.com/spf13/viper"
)

type generateModel struct {
	BaseGenerator
	name     string
	model    string
	destPath string
	filePath string
	file     *parser.File
}

func NewGenerateModel(name string, model string) Gen {
	t := &generateModel{
		name:     name,
		destPath: fmt.Sprintf(viper.GetString("gk_model_path_format"), utils.ToLowerSnakeCase(name)),
		model:    model,
	}
	t.filePath = path.Join(t.destPath, utils.ToLowerFirstCamelCase(model)+".go")
	t.srcFile = jen.NewFilePath(t.destPath)
	t.InitPg()
	t.fs = fs.Get()
	t.fs = fs.Get()
	return t
}
func (g *generateModel) Generate() (err error) {
	err = g.CreateFolderStructure(g.destPath)
	if err != nil {
		return err
	}
	if b, err := g.fs.Exists(g.filePath); err != nil {
		return err
	} else if b {
		return nil
	}
	g.code.appendStruct(
		utils.ToUpperFirst(utils.ToCamelCase(g.model)),
		jen.Qual("github.com/kujtimiihoxha/shqip-core/db", "BaseModel"),
	)
	g.code.appendStruct(
		utils.ToUpperFirst(utils.ToCamelCase(g.model)+"Controller"),
		jen.Id(utils.ToLowerFirstCamelCase(g.model)).Id("*").Id(utils.ToUpperFirst(utils.ToCamelCase(g.model))),
	)
	imp, err := utils.GetDBImportPath(g.name)
	if err != nil {
		return err
	}
	m := utils.ToLowerFirstCamelCase(string(g.model[0]))
	g.code.appendFunction(
		"Insert",
		jen.Id(m).Id("*").Id(utils.ToUpperFirst(utils.ToCamelCase(g.model)+"Controller")),
		[]jen.Code{},
		[]jen.Code{
			jen.Error(),
		},
		"",
		jen.Err().Op(":=").Qual(imp, "Session").Call().Dot("Create").Call(
			jen.Id(m).Dot(utils.ToLowerFirstCamelCase(g.model)),
		).Dot("Error"),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(
				jen.Qual("github.com/kujtimiihoxha/shqip-core/errors", "NewDBCreateError").Call(
					jen.Id("err"),
				),
			),
		),
		jen.Return(jen.Nil()),
	)
	g.code.NewLine()
	g.code.appendFunction(
		"GetById",
		jen.Id(m).Id("*").Id(utils.ToUpperFirst(utils.ToCamelCase(g.model)+"Controller")),
		[]jen.Code{
			jen.Id("id").Uint(),
		},
		[]jen.Code{
			jen.Error(),
		},
		"",
		jen.Err().Op(":=").Qual(imp, "Session").Call().Dot("First").Call(
			jen.Id(m).Dot(utils.ToLowerFirstCamelCase(g.model)),
			jen.Id("id"),
		).Dot("Error"),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(
				jen.Qual("github.com/kujtimiihoxha/shqip-core/errors", "NewDBGetByIDError").Call(
					jen.Id("id"),
					jen.Id("err"),
				),
			),
		),
		jen.Return(jen.Nil()),
	)
	g.code.NewLine()
	g.code.raw.Comment("THIS WILL CHANGE ALL COLUMNS BE CAREFUL").Line()
	g.code.appendFunction(
		"Save",
		jen.Id(m).Id("*").Id(utils.ToUpperFirst(utils.ToCamelCase(g.model)+"Controller")),
		[]jen.Code{},
		[]jen.Code{
			jen.Error(),
		},
		"",
		jen.Err().Op(":=").Qual(imp, "Session").Call().Dot("Save").Call(
			jen.Id(m).Dot(utils.ToLowerFirstCamelCase(g.model)),
		).Dot("Error"),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(
				jen.Qual("github.com/kujtimiihoxha/shqip-core/errors", "NewDBUpdateError").Call(
					jen.Id("err"),
				),
			),
		),
		jen.Return(jen.Nil()),
	)
	g.code.NewLine()
	g.code.appendFunction(
		"Delete",
		jen.Id(m).Id("*").Id(utils.ToUpperFirst(utils.ToCamelCase(g.model)+"Controller")),
		[]jen.Code{},
		[]jen.Code{
			jen.Error(),
		},
		"",
		jen.Err().Op(":=").Qual(imp, "Session").Call().Dot("Delete").Call(
			jen.Id(m).Dot(utils.ToLowerFirstCamelCase(g.model)),
		).Dot("Error"),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(
				jen.Qual("github.com/kujtimiihoxha/shqip-core/errors", "NewDBDeleteError").Call(
					jen.Id("err"),
				),
			),
		),
		jen.Return(jen.Nil()),
	)
	src := g.srcFile.GoString()
	s, err := utils.GoImportsSource(g.destPath, src)
	if err != nil {
		return err
	}
	return g.fs.WriteFile(g.filePath, s, true)
}
