package generator

import (
	"reflect"
	"testing"

	"github.com/dave/jennifer/jen"
	"github.com/emicklei/proto"
	"github.com/kujtimiihoxha/kit/fs"
	"github.com/kujtimiihoxha/kit/parser"
)

func TestNewGenerateTransport(t *testing.T) {
	setDefaults()
	type args struct {
		name       string
		gorillaMux bool
		transport  string
		methods    []string
	}
	tests := []struct {
		name string
		args args
		want Gen
	}{
		{
			name: "Test if generator is created properly",
			args: args{
				name:       "test",
				gorillaMux: false,
				transport:  "http",
				methods:    []string{},
			},
			want: &GenerateTransport{
				BaseGenerator: func() BaseGenerator {
					b := BaseGenerator{}
					b.srcFile = jen.NewFilePath("")
					b.InitPg()
					b.fs = fs.Get()
					return b
				}(),
				name:          "test",
				interfaceName: "TestService",
				transport:     "http",
				filePath:      "test/pkg/service/service.go",
				destPath:      "test/pkg/service",
				methods:       []string{},
			},
		},
		{
			name: "Test if bad name format still works",
			args: args{
				name:       "t es_t",
				gorillaMux: false,
				transport:  "http",
				methods:    []string{},
			},
			want: &GenerateTransport{
				BaseGenerator: func() BaseGenerator {
					b := BaseGenerator{}
					b.srcFile = jen.NewFilePath("")
					b.InitPg()
					b.fs = fs.Get()
					return b
				}(),
				name:          "t es_t",
				interfaceName: "TEsTService",
				transport:     "http",
				filePath:      "tes_t/pkg/service/service.go",
				destPath:      "tes_t/pkg/service",
				methods:       []string{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewGenerateTransport(tt.args.name, tt.args.gorillaMux, tt.args.transport, tt.args.methods); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewGenerateTransport() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateTransport_Generate(t *testing.T) {
	setDefaults()
	type fields struct {
		BaseGenerator    BaseGenerator
		name             string
		transport        string
		interfaceName    string
		destPath         string
		methods          []string
		filePath         string
		file             *parser.File
		serviceInterface parser.Interface
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Test if generator throws error when no service file",
			fields: fields{
				BaseGenerator: func() BaseGenerator {
					b := BaseGenerator{}
					b.srcFile = jen.NewFilePath("")
					b.InitPg()
					b.fs = fs.NewDefaultFs("")
					return b
				}(),
				name:          "test",
				interfaceName: "TestService",
				transport:     "http",
				filePath:      "test/pkg/service/service.go",
				destPath:      "test/pkg/service",
				methods:       []string{},
			},
			wantErr: true,
		}, {
			name: "Test if generator throws error when no service file",
			fields: fields{
				BaseGenerator: func() BaseGenerator {
					b := BaseGenerator{}
					b.srcFile = jen.NewFilePath("")
					b.InitPg()
					f := fs.NewDefaultFs("")
					f.MkdirAll("test/pkg/service")
					f.WriteFile("test/pkg/service/service.go", `package service
import "context"
type TestService interface{
		Foo(ctx context.Context, a string)(a int, err error)
}`, true)
					b.fs = f
					return b
				}(),
				name:          "test",
				interfaceName: "TestService",
				transport:     "http",
				filePath:      "test/pkg/service/service.go",
				destPath:      "test/pkg/service",
				methods:       []string{},
			},
			wantErr: false,
		}, {
			name: "Test if generator throws error when no service interface found in file",
			fields: fields{
				BaseGenerator: func() BaseGenerator {
					b := BaseGenerator{}
					b.srcFile = jen.NewFilePath("")
					b.InitPg()
					f := fs.NewDefaultFs("")
					f.MkdirAll("test/pkg/service")
					f.WriteFile("test/pkg/service/service.go", `package service
import "context"
`, true)
					b.fs = f
					return b
				}(),
				name:          "test",
				interfaceName: "TestService",
				transport:     "http",
				filePath:      "test/pkg/service/service.go",
				destPath:      "test/pkg/service",
				methods:       []string{},
			},
			wantErr: true,
		}, {
			name: "Test if generator throws error when transport not supported",
			fields: fields{
				BaseGenerator: func() BaseGenerator {
					b := BaseGenerator{}
					b.srcFile = jen.NewFilePath("")
					b.InitPg()
					f := fs.NewDefaultFs("")
					f.MkdirAll("test/pkg/service")
					f.WriteFile("test/pkg/service/service.go", `package service
import "context"
type TestService interface{
		Foo(ctx context.Context, a string)(a int, err error)
}`, true)
					b.fs = f
					return b
				}(),
				name:          "test",
				interfaceName: "TestService",
				transport:     "blla",
				filePath:      "test/pkg/service/service.go",
				destPath:      "test/pkg/service",
				methods:       []string{},
			},
			wantErr: true,
		}, {
			name: "Test if grpc create successful",
			fields: fields{
				BaseGenerator: func() BaseGenerator {
					b := BaseGenerator{}
					b.srcFile = jen.NewFilePath("")
					b.InitPg()
					f := fs.NewDefaultFs("")
					f.MkdirAll("test/pkg/service")
					f.WriteFile("test/pkg/service/service.go", `package service
					import "context"
					type TestService interface{
							Foo(ctx context.Context, a string)(a int, err error)
					}`, true)
					b.fs = f
					return b
				}(),
				name: "test",
				file: func() *parser.File {
					f := fs.Get()
					s, _ := f.ReadFile("test/pkg/service/service.go")
					fl, _ := parser.NewFileParser().Parse([]byte(s))
					return fl
				}(),
				interfaceName: "TestService",
				transport:     "grpc",
				filePath:      "test/pkg/service/service.go",
				destPath:      "test/pkg/service",
				methods:       []string{},
			},
			wantErr: false,
		}, {
			name: "Test if http create successful",
			fields: fields{
				BaseGenerator: func() BaseGenerator {
					b := BaseGenerator{}
					b.srcFile = jen.NewFilePath("")
					b.InitPg()
					f := fs.NewDefaultFs("")
					f.MkdirAll("test/pkg/service")
					f.WriteFile("test/pkg/service/service.go", `package service
					import "context"
					type TestService interface{
							Foo(ctx context.Context, a string)(a int, err error)
					}`, true)
					b.fs = f
					return b
				}(),
				name: "test",
				file: func() *parser.File {
					f := fs.Get()
					s, _ := f.ReadFile("test/pkg/service/service.go")
					fl, _ := parser.NewFileParser().Parse([]byte(s))
					return fl
				}(),
				interfaceName: "TestService",
				transport:     "http",
				filePath:      "test/pkg/service/service.go",
				destPath:      "test/pkg/service",
				methods:       []string{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &GenerateTransport{
				BaseGenerator:    tt.fields.BaseGenerator,
				name:             tt.fields.name,
				transport:        tt.fields.transport,
				interfaceName:    tt.fields.interfaceName,
				destPath:         tt.fields.destPath,
				methods:          tt.fields.methods,
				filePath:         tt.fields.filePath,
				file:             tt.fields.file,
				serviceInterface: tt.fields.serviceInterface,
			}
			if err := g.Generate(); (err != nil) != tt.wantErr {
				t.Errorf("GenerateTransport.Generate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateTransport_serviceFound(t *testing.T) {
	type fields struct {
		BaseGenerator    BaseGenerator
		name             string
		transport        string
		interfaceName    string
		destPath         string
		methods          []string
		filePath         string
		file             *parser.File
		serviceInterface parser.Interface
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "Test if service interface is found successfully",
			fields: fields{
				BaseGenerator: func() BaseGenerator {
					b := BaseGenerator{}
					b.srcFile = jen.NewFilePath("")
					b.InitPg()
					f := fs.NewDefaultFs("")
					f.MkdirAll("test/pkg/service")
					f.WriteFile("test/pkg/service/service.go", `package service
					import "context"
					type TestService interface{
							Foo(ctx context.Context, a string)(a int, err error)
					}`, true)
					b.fs = f
					return b
				}(),
				name: "test",
				file: func() *parser.File {
					f := fs.Get()
					s, _ := f.ReadFile("test/pkg/service/service.go")
					fl, _ := parser.NewFileParser().Parse([]byte(s))
					return fl
				}(),
				interfaceName: "TestService",
				transport:     "http",
				filePath:      "test/pkg/service/service.go",
				destPath:      "test/pkg/service",
				methods:       []string{},
			},
			want: true,
		},
		{
			name: "Test if service interface is not found",
			fields: fields{
				BaseGenerator: func() BaseGenerator {
					b := BaseGenerator{}
					b.srcFile = jen.NewFilePath("")
					b.InitPg()
					f := fs.NewDefaultFs("")
					f.MkdirAll("test/pkg/service")
					f.WriteFile("test/pkg/service/service.go", `package service
					import "context"`, true)
					b.fs = f
					return b
				}(),
				name: "test",
				file: func() *parser.File {
					f := fs.Get()
					s, _ := f.ReadFile("test/pkg/service/service.go")
					fl, _ := parser.NewFileParser().Parse([]byte(s))
					return fl
				}(),
				interfaceName: "TestService",
				transport:     "http",
				filePath:      "test/pkg/service/service.go",
				destPath:      "test/pkg/service",
				methods:       []string{},
			},
			want: false,
		},
		{
			name: "Test if service interface is not found",
			fields: fields{
				BaseGenerator: func() BaseGenerator {
					b := BaseGenerator{}
					b.srcFile = jen.NewFilePath("")
					b.InitPg()
					f := fs.NewDefaultFs("")
					f.MkdirAll("test/pkg/service")
					f.WriteFile("test/pkg/service/service.go", `package service
					import "context"
					type HelloService interface {} `, true)
					b.fs = f
					return b
				}(),
				name: "test",
				file: func() *parser.File {
					f := fs.Get()
					s, _ := f.ReadFile("test/pkg/service/service.go")
					fl, _ := parser.NewFileParser().Parse([]byte(s))
					return fl
				}(),
				interfaceName: "TestService",
				transport:     "http",
				filePath:      "test/pkg/service/service.go",
				destPath:      "test/pkg/service",
				methods:       []string{},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &GenerateTransport{
				BaseGenerator:    tt.fields.BaseGenerator,
				name:             tt.fields.name,
				transport:        tt.fields.transport,
				interfaceName:    tt.fields.interfaceName,
				destPath:         tt.fields.destPath,
				methods:          tt.fields.methods,
				filePath:         tt.fields.filePath,
				file:             tt.fields.file,
				serviceInterface: tt.fields.serviceInterface,
			}
			if got := g.serviceFound(); got != tt.want {
				t.Errorf("GenerateTransport.serviceFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateTransport_removeBadMethods(t *testing.T) {
	type fields struct {
		BaseGenerator    BaseGenerator
		name             string
		transport        string
		interfaceName    string
		destPath         string
		methods          []string
		filePath         string
		file             *parser.File
		serviceInterface parser.Interface
	}
	tests := []struct {
		name   string
		fields fields
		want   []parser.Method
	}{
		{
			name: "Test if it does not remove wanted methods",
			fields: fields{
				BaseGenerator: func() BaseGenerator {
					b := BaseGenerator{}
					b.srcFile = jen.NewFilePath("")
					b.InitPg()
					f := fs.NewDefaultFs("")
					f.MkdirAll("test/pkg/service")
					f.WriteFile("test/pkg/service/service.go", `package service
					import "context"
					type TestService interface {
					Foo(ctx context.Context,a int)(r string, err error)
					} `, true)
					b.fs = f
					return b
				}(),
				name: "test",
				file: func() *parser.File {
					f := fs.Get()
					s, _ := f.ReadFile("test/pkg/service/service.go")
					fl, _ := parser.NewFileParser().Parse([]byte(s))
					return fl
				}(),
				serviceInterface: func() parser.Interface {
					f := fs.Get()
					s, _ := f.ReadFile("test/pkg/service/service.go")
					fl, _ := parser.NewFileParser().Parse([]byte(s))
					return fl.Interfaces[0]
				}(),
				interfaceName: "TestService",
				transport:     "http",
				filePath:      "test/pkg/service/service.go",
				destPath:      "test/pkg/service",
				methods:       []string{},
			},
			want: []parser.Method{
				parser.NewMethod(
					"Foo",
					parser.NamedTypeValue{},
					"",
					[]parser.NamedTypeValue{
						parser.NewNameType("ctx", "context.Context"),
						parser.NewNameType("a", "int"),
					},
					[]parser.NamedTypeValue{
						parser.NewNameType("r", "string"),
						parser.NewNameType("err", "error"),
					},
				),
			},
		},
		{
			name: "Test if it removes unwanted methods",
			fields: fields{
				BaseGenerator: func() BaseGenerator {
					b := BaseGenerator{}
					b.srcFile = jen.NewFilePath("")
					b.InitPg()
					f := fs.NewDefaultFs("")
					f.MkdirAll("test/pkg/service")
					f.WriteFile("test/pkg/service/service.go", `package service
					import "context"
					type TestService interface {
					Foo(ctx context.Context,a int)(r string, err error)
					Bar(a int)(r string, err error)
					foobar(a int)(r string, err error)
					BarFoo(ctx context.Context, a int)
					} `, true)
					b.fs = f
					return b
				}(),
				name: "test",
				file: func() *parser.File {
					f := fs.Get()
					s, _ := f.ReadFile("test/pkg/service/service.go")
					fl, _ := parser.NewFileParser().Parse([]byte(s))
					return fl
				}(),
				serviceInterface: func() parser.Interface {
					f := fs.Get()
					s, _ := f.ReadFile("test/pkg/service/service.go")
					fl, _ := parser.NewFileParser().Parse([]byte(s))
					return fl.Interfaces[0]
				}(),
				interfaceName: "TestService",
				transport:     "http",
				filePath:      "test/pkg/service/service.go",
				destPath:      "test/pkg/service",
				methods:       []string{},
			},
			want: []parser.Method{
				parser.NewMethod(
					"Foo",
					parser.NamedTypeValue{},
					"",
					[]parser.NamedTypeValue{
						parser.NewNameType("ctx", "context.Context"),
						parser.NewNameType("a", "int"),
					},
					[]parser.NamedTypeValue{
						parser.NewNameType("r", "string"),
						parser.NewNameType("err", "error"),
					},
				),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &GenerateTransport{
				BaseGenerator:    tt.fields.BaseGenerator,
				name:             tt.fields.name,
				transport:        tt.fields.transport,
				interfaceName:    tt.fields.interfaceName,
				destPath:         tt.fields.destPath,
				methods:          tt.fields.methods,
				filePath:         tt.fields.filePath,
				file:             tt.fields.file,
				serviceInterface: tt.fields.serviceInterface,
			}
			g.removeBadMethods()
			if !reflect.DeepEqual(g.serviceInterface.Methods, tt.want) {
				t.Errorf("After GenerateTransport.removeBadMethods(): Methods %v, want %v", g.serviceInterface.Methods, tt.want)
			}
		})
	}
}

func TestGenerateTransport_removeUnwantedMethods(t *testing.T) {
	type fields struct {
		BaseGenerator    BaseGenerator
		name             string
		transport        string
		interfaceName    string
		destPath         string
		methods          []string
		filePath         string
		file             *parser.File
		serviceInterface parser.Interface
	}
	tests := []struct {
		name   string
		fields fields
		want   []parser.Method
	}{
		{
			name: "Test if only selected methods are generated",
			fields: fields{
				BaseGenerator: func() BaseGenerator {
					b := BaseGenerator{}
					b.srcFile = jen.NewFilePath("")
					b.InitPg()
					f := fs.NewDefaultFs("")
					f.MkdirAll("test/pkg/service")
					f.WriteFile("test/pkg/service/service.go", `package service
					import "context"
					type TestService interface {
					Foo(ctx context.Context,a int)(r string, err error)
					Bar(ctx context.Context,a int)(r []string, err error)
					FooBar(ctx context.Context,a int)(r int, err error)
					} `, true)
					b.fs = f
					return b
				}(),
				name: "test",
				file: func() *parser.File {
					f := fs.Get()
					s, _ := f.ReadFile("test/pkg/service/service.go")
					fl, _ := parser.NewFileParser().Parse([]byte(s))
					return fl
				}(),
				serviceInterface: func() parser.Interface {
					f := fs.Get()
					s, _ := f.ReadFile("test/pkg/service/service.go")
					fl, _ := parser.NewFileParser().Parse([]byte(s))
					return fl.Interfaces[0]
				}(),
				interfaceName: "TestService",
				transport:     "http",
				filePath:      "test/pkg/service/service.go",
				destPath:      "test/pkg/service",
				methods:       []string{"Foo"},
			},
			want: []parser.Method{
				parser.NewMethod(
					"Foo",
					parser.NamedTypeValue{},
					"",
					[]parser.NamedTypeValue{
						parser.NewNameType("ctx", "context.Context"),
						parser.NewNameType("a", "int"),
					},
					[]parser.NamedTypeValue{
						parser.NewNameType("r", "string"),
						parser.NewNameType("err", "error"),
					},
				),
			},
		}, {
			name: "Test if only ot does not do anything if no methods selected",
			fields: fields{
				BaseGenerator: func() BaseGenerator {
					b := BaseGenerator{}
					b.srcFile = jen.NewFilePath("")
					b.InitPg()
					f := fs.NewDefaultFs("")
					f.MkdirAll("test/pkg/service")
					f.WriteFile("test/pkg/service/service.go", `package service
					import "context"
					type TestService interface {
					Foo(ctx context.Context,a int)(r string, err error)
					Bar(ctx context.Context,a int)(r string, err error)
					} `, true)
					b.fs = f
					return b
				}(),
				name: "test",
				file: func() *parser.File {
					f := fs.Get()
					s, _ := f.ReadFile("test/pkg/service/service.go")
					fl, _ := parser.NewFileParser().Parse([]byte(s))
					return fl
				}(),
				serviceInterface: func() parser.Interface {
					f := fs.Get()
					s, _ := f.ReadFile("test/pkg/service/service.go")
					fl, _ := parser.NewFileParser().Parse([]byte(s))
					return fl.Interfaces[0]
				}(),
				interfaceName: "TestService",
				transport:     "http",
				filePath:      "test/pkg/service/service.go",
				destPath:      "test/pkg/service",
				methods:       []string{},
			},
			want: []parser.Method{
				parser.NewMethod(
					"Foo",
					parser.NamedTypeValue{},
					"",
					[]parser.NamedTypeValue{
						parser.NewNameType("ctx", "context.Context"),
						parser.NewNameType("a", "int"),
					},
					[]parser.NamedTypeValue{
						parser.NewNameType("r", "string"),
						parser.NewNameType("err", "error"),
					},
				),
				parser.NewMethod(
					"Bar",
					parser.NamedTypeValue{},
					"",
					[]parser.NamedTypeValue{
						parser.NewNameType("ctx", "context.Context"),
						parser.NewNameType("a", "int"),
					},
					[]parser.NamedTypeValue{
						parser.NewNameType("r", "string"),
						parser.NewNameType("err", "error"),
					},
				),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &GenerateTransport{
				BaseGenerator:    tt.fields.BaseGenerator,
				name:             tt.fields.name,
				transport:        tt.fields.transport,
				interfaceName:    tt.fields.interfaceName,
				destPath:         tt.fields.destPath,
				methods:          tt.fields.methods,
				filePath:         tt.fields.filePath,
				file:             tt.fields.file,
				serviceInterface: tt.fields.serviceInterface,
			}
			g.removeUnwantedMethods()
			if !reflect.DeepEqual(g.serviceInterface.Methods, tt.want) {
				t.Errorf("After GenerateTransport.removeUnwantedMethods(): Methods %v, want %v", g.serviceInterface.Methods, tt.want)
			}
		})
	}
}

func Test_newGenerateHTTPTransport(t *testing.T) {
	type args struct {
		name             string
		gorillaMux       bool
		serviceInterface parser.Interface
		methods          []string
	}
	tests := []struct {
		name string
		args args
		want Gen
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newGenerateHTTPTransport(tt.args.name, tt.args.gorillaMux, tt.args.serviceInterface, tt.args.methods); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newGenerateHTTPTransport() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_generateHTTPTransport_Generate(t *testing.T) {
	type fields struct {
		BaseGenerator     BaseGenerator
		name              string
		methods           []string
		interfaceName     string
		destPath          string
		generateFirstTime bool
		file              *parser.File
		filePath          string
		serviceInterface  parser.Interface
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &generateHTTPTransport{
				BaseGenerator:     tt.fields.BaseGenerator,
				name:              tt.fields.name,
				methods:           tt.fields.methods,
				interfaceName:     tt.fields.interfaceName,
				destPath:          tt.fields.destPath,
				generateFirstTime: tt.fields.generateFirstTime,
				file:              tt.fields.file,
				filePath:          tt.fields.filePath,
				serviceInterface:  tt.fields.serviceInterface,
			}
			if err := g.Generate(); (err != nil) != tt.wantErr {
				t.Errorf("generateHTTPTransport.Generate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_newGenerateHTTPTransportBase(t *testing.T) {
	type args struct {
		name             string
		gorillaMux       bool
		serviceInterface parser.Interface
		methods          []string
		allMethods       []parser.Method
	}
	tests := []struct {
		name string
		args args
		want Gen
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newGenerateHTTPTransportBase(tt.args.name, tt.args.gorillaMux, tt.args.serviceInterface, tt.args.methods, tt.args.allMethods); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newGenerateHTTPTransportBase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_generateHTTPTransportBase_Generate(t *testing.T) {
	type fields struct {
		BaseGenerator    BaseGenerator
		name             string
		methods          []string
		allMethods       []parser.Method
		interfaceName    string
		destPath         string
		filePath         string
		file             *parser.File
		httpFilePath     string
		serviceInterface parser.Interface
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &generateHTTPTransportBase{
				BaseGenerator:    tt.fields.BaseGenerator,
				name:             tt.fields.name,
				methods:          tt.fields.methods,
				allMethods:       tt.fields.allMethods,
				interfaceName:    tt.fields.interfaceName,
				destPath:         tt.fields.destPath,
				filePath:         tt.fields.filePath,
				file:             tt.fields.file,
				httpFilePath:     tt.fields.httpFilePath,
				serviceInterface: tt.fields.serviceInterface,
			}
			if err := g.Generate(); (err != nil) != tt.wantErr {
				t.Errorf("generateHTTPTransportBase.Generate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_newGenerateGRPCTransportProto(t *testing.T) {
	type args struct {
		name             string
		serviceInterface parser.Interface
		methods          []string
	}
	tests := []struct {
		name string
		args args
		want Gen
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newGenerateGRPCTransportProto(tt.args.name, tt.args.serviceInterface, tt.args.methods); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newGenerateGRPCTransportProto() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_generateGRPCTransportProto_Generate(t *testing.T) {
	type fields struct {
		BaseGenerator     BaseGenerator
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
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &generateGRPCTransportProto{
				BaseGenerator:     tt.fields.BaseGenerator,
				name:              tt.fields.name,
				methods:           tt.fields.methods,
				interfaceName:     tt.fields.interfaceName,
				generateFirstTime: tt.fields.generateFirstTime,
				destPath:          tt.fields.destPath,
				protoSrc:          tt.fields.protoSrc,
				pbFilePath:        tt.fields.pbFilePath,
				compileFilePath:   tt.fields.compileFilePath,
				serviceInterface:  tt.fields.serviceInterface,
			}
			if err := g.Generate(); (err != nil) != tt.wantErr {
				t.Errorf("generateGRPCTransportProto.Generate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_generateGRPCTransportProto_getService(t *testing.T) {
	type fields struct {
		BaseGenerator     BaseGenerator
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
	tests := []struct {
		name   string
		fields fields
		want   *proto.Service
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &generateGRPCTransportProto{
				BaseGenerator:     tt.fields.BaseGenerator,
				name:              tt.fields.name,
				methods:           tt.fields.methods,
				interfaceName:     tt.fields.interfaceName,
				generateFirstTime: tt.fields.generateFirstTime,
				destPath:          tt.fields.destPath,
				protoSrc:          tt.fields.protoSrc,
				pbFilePath:        tt.fields.pbFilePath,
				compileFilePath:   tt.fields.compileFilePath,
				serviceInterface:  tt.fields.serviceInterface,
			}
			if got := g.getService(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("generateGRPCTransportProto.getService() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_generateGRPCTransportProto_generateRequestResponse(t *testing.T) {
	type fields struct {
		BaseGenerator     BaseGenerator
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
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &generateGRPCTransportProto{
				BaseGenerator:     tt.fields.BaseGenerator,
				name:              tt.fields.name,
				methods:           tt.fields.methods,
				interfaceName:     tt.fields.interfaceName,
				generateFirstTime: tt.fields.generateFirstTime,
				destPath:          tt.fields.destPath,
				protoSrc:          tt.fields.protoSrc,
				pbFilePath:        tt.fields.pbFilePath,
				compileFilePath:   tt.fields.compileFilePath,
				serviceInterface:  tt.fields.serviceInterface,
			}
			g.generateRequestResponse()
		})
	}
}

func Test_generateGRPCTransportProto_getServiceRPC(t *testing.T) {
	type fields struct {
		BaseGenerator     BaseGenerator
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
	type args struct {
		svc *proto.Service
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &generateGRPCTransportProto{
				BaseGenerator:     tt.fields.BaseGenerator,
				name:              tt.fields.name,
				methods:           tt.fields.methods,
				interfaceName:     tt.fields.interfaceName,
				generateFirstTime: tt.fields.generateFirstTime,
				destPath:          tt.fields.destPath,
				protoSrc:          tt.fields.protoSrc,
				pbFilePath:        tt.fields.pbFilePath,
				compileFilePath:   tt.fields.compileFilePath,
				serviceInterface:  tt.fields.serviceInterface,
			}
			g.getServiceRPC(tt.args.svc)
		})
	}
}

func Test_newGenerateGRPCTransportBase(t *testing.T) {
	type args struct {
		name             string
		serviceInterface parser.Interface
		methods          []string
		allMethods       []parser.Method
	}
	tests := []struct {
		name string
		args args
		want Gen
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newGenerateGRPCTransportBase(tt.args.name, tt.args.serviceInterface, tt.args.methods, tt.args.allMethods); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newGenerateGRPCTransportBase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_generateGRPCTransportBase_Generate(t *testing.T) {
	type fields struct {
		BaseGenerator    BaseGenerator
		name             string
		methods          []string
		allMethods       []parser.Method
		interfaceName    string
		destPath         string
		filePath         string
		file             *parser.File
		grpcFilePath     string
		serviceInterface parser.Interface
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &generateGRPCTransportBase{
				BaseGenerator:    tt.fields.BaseGenerator,
				name:             tt.fields.name,
				methods:          tt.fields.methods,
				allMethods:       tt.fields.allMethods,
				interfaceName:    tt.fields.interfaceName,
				destPath:         tt.fields.destPath,
				filePath:         tt.fields.filePath,
				file:             tt.fields.file,
				grpcFilePath:     tt.fields.grpcFilePath,
				serviceInterface: tt.fields.serviceInterface,
			}
			if err := g.Generate(); (err != nil) != tt.wantErr {
				t.Errorf("generateGRPCTransportBase.Generate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_newGenerateGRPCTransport(t *testing.T) {
	type args struct {
		name             string
		serviceInterface parser.Interface
		methods          []string
	}
	tests := []struct {
		name string
		args args
		want Gen
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newGenerateGRPCTransport(tt.args.name, tt.args.serviceInterface, tt.args.methods); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newGenerateGRPCTransport() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_generateGRPCTransport_Generate(t *testing.T) {
	type fields struct {
		BaseGenerator     BaseGenerator
		name              string
		methods           []string
		interfaceName     string
		destPath          string
		generateFirstTime bool
		file              *parser.File
		filePath          string
		serviceInterface  parser.Interface
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &generateGRPCTransport{
				BaseGenerator:     tt.fields.BaseGenerator,
				name:              tt.fields.name,
				methods:           tt.fields.methods,
				interfaceName:     tt.fields.interfaceName,
				destPath:          tt.fields.destPath,
				generateFirstTime: tt.fields.generateFirstTime,
				file:              tt.fields.file,
				filePath:          tt.fields.filePath,
				serviceInterface:  tt.fields.serviceInterface,
			}
			if err := g.Generate(); (err != nil) != tt.wantErr {
				t.Errorf("generateGRPCTransport.Generate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
