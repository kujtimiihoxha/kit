package service

import (
	"fmt"
	"github.com/go-services/annotation"
	"github.com/ozgio/strutil"
	"github.com/spf13/afero"
	"kit/fs"
	"kit/template"
	"strings"
)

type TransportType string

const (
	// transport types
	HTTPType TransportType = "http"
)

type MethodRoute struct {
	Name       string
	Methods    []string
	MethodsAll string
	Route      string
}

type HTTPTransport struct {
	MethodRoutes []MethodRoute
}

type MethodTransport interface {
	Generate(s Service, method Method, genFs afero.Fs) error
}

func NewHTTPTransport(httpAnnotations []annotation.Annotation) MethodTransport {
	transport := HTTPTransport{}
	for _, httpAnnotation := range httpAnnotations {
		if httpAnnotation.Name != "http" {
			continue
		}
		var methodsPrepared []string
		for _, method := range strings.Split(httpAnnotation.Get("methods").String(), ",") {
			methodsPrepared = append(methodsPrepared, strings.ToUpper(strings.TrimSpace(method)))
		}
		route := httpAnnotation.Get("route").String()
		if !strings.HasPrefix(route, "/") {
			route = "/" + route
		}
		transport.MethodRoutes = append(transport.MethodRoutes, MethodRoute{
			Name:       httpAnnotation.Get("name").String(),
			Methods:    methodsPrepared,
			MethodsAll: strings.Join(methodsPrepared, ", "),
			Route:    route  ,
		})
	}
	return transport
}

func (h HTTPTransport) Generate(svc Service, method Method, transportFs afero.Fs) error {
	transportFs = afero.NewBasePathFs(transportFs, "http")
	methodHttpCode, err := template.CompileGoFromPath(
		"/assets/templates/service/gen/transport/http/method.go.gotmpl",
		map[string]interface{}{
			"Service": svc,
			"Method":  method,
			"HTTP":    h,
		},
	)
	if err != nil {
		return err
	}
	return fs.WriteFile(
		fmt.Sprintf("%s.go", strutil.ToSnakeCase(method.Code.Name)),
		methodHttpCode,
		transportFs,
	)
}
