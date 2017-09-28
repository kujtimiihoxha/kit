package parser

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestFileParser_Parse(t *testing.T) {
	fp := NewFileParser()
	_, err := fp.Parse([]byte(
		`package main

		func main() {
			print("Hello")
		}
		`))
	Convey("Test if parser parses file without errors", t, func() {
		So(err, ShouldBeNil)
	})
}
func TestFileParser_ParseServiceInterface(t *testing.T) {
	fp := NewFileParser()
	f, err := fp.Parse([]byte(
		`package parser

import ct "context"
// My service
type Hi struct {}
type MyService interface{
	Foo(ctx ct.Context, s map[string]string) ([]string, *Hi, error)
}`))
	Convey("Test if parser parses file without errors", t, func() {
		So(err, ShouldBeNil)
		Convey("Test if interface has been found", func() {
			So(len(f.Interfaces), ShouldEqual, 1)
			So(f.Interfaces[0].Name, ShouldEqual, "MyService")
			Convey("Test if interface methods are parsed", func() {
				So(len(f.Interfaces[0].Methods), ShouldEqual, 1)
				So(f.Interfaces[0].Methods[0].Name, ShouldEqual, "Foo")
				Convey("Test if method has right parameters and results", func() {
					m := f.Interfaces[0].Methods[0]
					So(len(m.Parameters), ShouldEqual, 2)
					So(len(m.Results), ShouldEqual, 3)
					So(m.Parameters[0].Name, ShouldEqual, "ctx")
					So(m.Parameters[0].Type, ShouldEqual, "ct.Context")
					So(m.Parameters[1].Name, ShouldEqual, "s")
					So(m.Parameters[1].Type, ShouldEqual, "map[string]string")

					So(m.Results[0].Name, ShouldEqual, "s0")
					So(m.Results[0].Type, ShouldEqual, "[]string")
					So(m.Results[1].Name, ShouldEqual, "h1")
					So(m.Results[1].Type, ShouldEqual, "*Hi")
					So(m.Results[2].Name, ShouldEqual, "e2")
					So(m.Results[2].Type, ShouldEqual, "error")
				})
			})
		})
	})
}
func TestFileParser_ParseStructFunction(t *testing.T) {
	fp := NewFileParser()
	f, err := fp.Parse([]byte(`package main
		type Hi struct{}
		func (a *Hi) hello(){
		print("hello")
		}`))
	Convey("Test if parser parses file without errors", t, func() {
		So(err, ShouldBeNil)
		Convey("Test structure has method", func() {
			So(len(f.Methods), ShouldEqual, 1)
			So(f.Methods[0].Struct.Name, ShouldEqual, "a")
			So(f.Methods[0].Struct.Type, ShouldEqual, "*Hi")
		})
	})
}
func TestFileParser_ParseVariablesConstants(t *testing.T) {
	fp := NewFileParser()
	f, err := fp.Parse([]byte(
		`package main
		var hi = "Hello there"
		var (
		no_value int
		abc string = "hi"
		)
		const (
			my_const = 2
			hello_there float = 4.23
		)
		func main() {
			print(hi)
			print(my_const)
		}
		`))
	Convey("Test if parser parses file without errors", t, func() {
		So(err, ShouldBeNil)
		Convey("Test if variables and constants are found", func() {
			So(len(f.Vars), ShouldEqual, 3)
			So(len(f.Constants), ShouldEqual, 2)
		})
	})
}

func TestFileParser_ParseMiddlewareFuncType(t *testing.T) {
	fp := NewFileParser()
	f, err := fp.Parse([]byte(
		`package main
			type Middleware func(int) int
		`))
	Convey("Test if parser parses file without errors", t, func() {
		So(err, ShouldBeNil)
		Convey("Test if middleware func type is found", func() {
			expect := FuncType{
				Name: "Middleware",
				Parameters: []NamedTypeValue{
					{
						Name: "i0",
						Type: "int",
					},
				},
				Results: []NamedTypeValue{
					{
						Name: "i0",
						Type: "int",
					},
				},
			}
			So(len(f.FuncType.Parameters), ShouldEqual, len(expect.Parameters))
			So(len(f.FuncType.Results), ShouldEqual, len(expect.Results))
			Convey("Test if middleware name and parameters/results are the same", func() {
				So(f.FuncType.Name, ShouldEqual, expect.Name)
				So(f.FuncType.Parameters[0].Name, ShouldEqual, expect.Parameters[0].Name)
				So(f.FuncType.Parameters[0].Type, ShouldEqual, expect.Parameters[0].Type)
				So(f.FuncType.Results[0].Type, ShouldEqual, expect.Results[0].Type)
			})
		})
	})
}
