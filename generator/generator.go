package generator

import (
	"fmt"
	"go/ast"
	ps "go/parser"
	"go/token"

	"strings"

	"strconv"

	"bytes"
	"go/format"

	"github.com/dave/jennifer/jen"
	"github.com/kujtimiihoxha/kit/fs"
	"github.com/kujtimiihoxha/kit/parser"
	"github.com/kujtimiihoxha/kit/utils"
	"github.com/sirupsen/logrus"
)

// Gen represents a generator.
type Gen interface {
	Generate() error
}

// BaseGenerator implements some basic generator functionality used by all generators.
type BaseGenerator struct {
	srcFile *jen.File
	code    *PartialGenerator
	fs      *fs.KitFs
}

// InitPg initiates the partial generator (used when we don't want to generate the full source only portions)
func (b *BaseGenerator) InitPg() {
	b.code = NewPartialGenerator(b.srcFile.Empty())
}
func (b *BaseGenerator) getMissingImports(imp []parser.NamedTypeValue, f *parser.File) ([]parser.NamedTypeValue, error) {
	n := []parser.NamedTypeValue{}
	for _, v := range imp {
		for i, vo := range f.Imports {
			if vo.Name == "" {
				tp, err := strconv.Unquote(vo.Type)
				if err != nil {
					return n, err
				}
				if v.Type == vo.Type && strings.HasSuffix(tp, v.Name) {
					break
				}
			}
			if v.Type == vo.Type && v.Name == vo.Name {
				break
			} else if i == len(f.Imports)-1 {
				n = append(n, v)
			}
		}
	}
	if len(f.Imports) == 0 {
		n = imp
	}
	return n, nil
}

// CreateFolderStructure create folder structure of path
func (b *BaseGenerator) CreateFolderStructure(path string) error {
	e, err := b.fs.Exists(path)
	if err != nil {
		return err
	}
	if !e {
		logrus.Debug(fmt.Sprintf("Creating missing folder structure : %s", path))
		return b.fs.MkdirAll(path)
	}
	return nil
}

// GenerateNameBySample is used to generate a variable name using a sample.
//
// The exclude parameter represents the names that it can not use.
//
// E.x  sample = "hello" this will return the name "h" if it is not in any NamedTypeValue name.
func (b *BaseGenerator) GenerateNameBySample(sample string, exclude []parser.NamedTypeValue) string {
	sn := 1
	name := utils.ToLowerFirstCamelCase(sample)[:sn]
	for _, v := range exclude {
		if v.Name == name {
			sn++
			if sn > len(sample) {
				sample = string(len(sample) - sn)
			}
			name = utils.ToLowerFirstCamelCase(sample)[:sn]
		}
	}
	return name
}

// EnsureThatWeUseQualifierIfNeeded is used to see if we need to import a path of a given type.
func (b *BaseGenerator) EnsureThatWeUseQualifierIfNeeded(tp string, imp []parser.NamedTypeValue) string {
	if bytes.HasPrefix([]byte(tp), []byte("...")) {
		return ""
	}
	if t := strings.Split(tp, "."); len(t) > 0 {
		s := t[0]
		for _, v := range imp {
			i, _ := strconv.Unquote(v.Type)
			if strings.HasSuffix(i, s) || v.Name == s {
				return i
			}
		}
		return ""
	}
	return ""
}

// AddImportsToFile adds missing imports toa file that we edit with the generator
func (b *BaseGenerator) AddImportsToFile(imp []parser.NamedTypeValue, src string) (string, error) {
	// Create the AST by parsing src
	fset := token.NewFileSet()
	f, err := ps.ParseFile(fset, "", src, 0)
	if err != nil {
		return "", err
	}
	found := false
	// Add the imports
	for i := 0; i < len(f.Decls); i++ {
		d := f.Decls[i]
		switch d.(type) {
		case *ast.FuncDecl:
			// No action
		case *ast.GenDecl:
			dd := d.(*ast.GenDecl)

			// IMPORT Declarations
			if dd.Tok == token.IMPORT {
				if dd.Rparen == 0 || dd.Lparen == 0 {
					dd.Rparen = f.Package
					dd.Lparen = f.Package
				}
				found = true
				// Add the new import
				for _, v := range imp {
					iSpec := &ast.ImportSpec{
						Name: &ast.Ident{Name: v.Name},
						Path: &ast.BasicLit{Value: v.Type},
					}
					dd.Specs = append(dd.Specs, iSpec)
				}
			}
		}
	}
	if !found {
		dd := ast.GenDecl{
			TokPos: f.Package + 1,
			Tok:    token.IMPORT,
			Specs:  []ast.Spec{},
			Lparen: f.Package,
			Rparen: f.Package,
		}
		lastPos := 0
		for _, v := range imp {
			lastPos += len(v.Name) + len(v.Type) + 1
			iSpec := &ast.ImportSpec{
				Name:   &ast.Ident{Name: v.Name},
				Path:   &ast.BasicLit{Value: v.Type},
				EndPos: token.Pos(lastPos),
			}
			dd.Specs = append(dd.Specs, iSpec)

		}
		f.Decls = append([]ast.Decl{&dd}, f.Decls...)
	}

	// Sort the imports
	ast.SortImports(fset, f)
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s", buf.Bytes()), nil
}

// PartialGenerator wraps a jen statement
type PartialGenerator struct {
	raw *jen.Statement
}

// NewPartialGenerator returns a partial generator
func NewPartialGenerator(st *jen.Statement) *PartialGenerator {
	if st != nil {
		return &PartialGenerator{
			raw: st,
		}
	}
	return &PartialGenerator{
		raw: &jen.Statement{},
	}
}
func (p *PartialGenerator) appendMultilineComment(c []string) {
	for i, v := range c {
		if i != len(c)-1 {
			p.raw.Comment(v).Line()
			continue
		}
		p.raw.Comment(v)
	}
}

// Raw returns the jen statement.
func (p *PartialGenerator) Raw() *jen.Statement {
	return p.raw
}

// String returns the source code string
func (p *PartialGenerator) String() string {
	return p.raw.GoString()
}
func (p *PartialGenerator) appendInterface(name string, methods []jen.Code) {
	p.raw.Type().Id(name).Interface(methods...).Line()
}

func (p *PartialGenerator) appendStruct(name string, fields ...jen.Code) {
	p.raw.Type().Id(name).Struct(fields...).Line()
}

// NewLine insert a new line in code.
func (p *PartialGenerator) NewLine() {
	p.raw.Line()
}

func (p *PartialGenerator) appendFunction(name string, stp *jen.Statement,
	parameters []jen.Code, results []jen.Code, oneResponse string, body ...jen.Code) {
	p.raw.Func()
	if stp != nil {
		p.raw.Params(stp)
	}
	if name != "" {
		p.raw.Id(name)
	}
	p.raw.Params(parameters...)
	if oneResponse != "" {
		p.raw.Id(oneResponse)
	} else if len(results) > 0 {
		p.raw.Params(results...)
	}
	p.raw.Block(body...)
}
