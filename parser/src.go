package parser

// File represents a go source file.
type File struct {
	Comment string
	Package string
	// Only used to get the middleware type
	FuncType   FuncType
	Imports    []NamedTypeValue
	Constants  []NamedTypeValue
	Vars       []NamedTypeValue
	Interfaces []Interface
	Structures []Struct
	Methods    []Method
}

// Struct stores go struct information.
type Struct struct {
	Name    string
	Comment string
	Vars    []NamedTypeValue
}

// FuncType is used to store e.x (type Middleware func(a)a) types
type FuncType struct {
	Name       string
	Parameters []NamedTypeValue
	Results    []NamedTypeValue
}

// Interface stores go interface information.
type Interface struct {
	Name    string
	Comment string
	Methods []Method
}

// Method stores go method information.
type Method struct {
	Comment    string
	Name       string
	Struct     NamedTypeValue
	Body       string
	Parameters []NamedTypeValue
	Results    []NamedTypeValue
}

// NamedTypeValue  is used to store any type of name type = value ( e.x  var a = 2)
type NamedTypeValue struct {
	Name  string
	Type  string
	Value string
}

// NewNameType create a NamedTypeValue without a value.
func NewNameType(name string, tp string) NamedTypeValue {
	return NamedTypeValue{
		Name: name,
		Type: tp,
	}
}

// NewNameTypeValue create a NamedTypeValue with a value.
func NewNameTypeValue(name string, tp string, vl string) NamedTypeValue {
	return NamedTypeValue{
		Name:  name,
		Type:  tp,
		Value: vl,
	}
}

// NewMethod creates a new method.
func NewMethod(name string, str NamedTypeValue, body string, parameters, results []NamedTypeValue) Method {
	return Method{
		Name:       name,
		Comment:    "",
		Struct:     str,
		Body:       body,
		Parameters: parameters,
		Results:    results,
	}
}

// NewInterface creates a new interface.
func NewInterface(name string, methods []Method) Interface {
	return Interface{
		Name:    name,
		Comment: "",
		Methods: methods,
	}
}

// NewStruct creates a new struct.
func NewStruct(name string, vars []NamedTypeValue) Struct {
	return Struct{
		Name:    name,
		Comment: "",
		Vars:    vars,
	}
}

// NewFile creates a new empty file.
func NewFile() File {
	return File{
		Interfaces: []Interface{},
		Imports:    []NamedTypeValue{},
		Structures: []Struct{},
		Vars:       []NamedTypeValue{},
		Constants:  []NamedTypeValue{},
		Methods:    []Method{},
	}
}
