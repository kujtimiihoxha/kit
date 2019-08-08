package generator

import (
	"github.com/go-services/code"
	"github.com/go-services/source"
	"strings"
	"unicode"
)

func isUpper(ch rune) bool {
	return ch == unicode.ToUpper(ch)
}
func fixMethodTypes(method *source.InterfaceMethod) {
	mthCode := method.Code().(*code.InterfaceMethod)
	for inx, param := range mthCode.Params {
		if param.Type.Import == nil && isUpper(rune(param.Type.Qualifier[0])) {
			mthCode.Params[inx].Type.Import = code.NewImport("service", "")
		}
	}
	for inx, result := range mthCode.Results {
		if result.Type.Import == nil && isUpper(rune(result.Type.Qualifier[0])) {
			mthCode.Results[inx].Type.Import = code.NewImport("service", "")
		}
	}
}

func routeMethods(methods string) string {
	var methodsPrepared []string
	for _, method := range strings.Split(methods, ",") {
		methodsPrepared = append(methodsPrepared, strings.ToUpper(method))
	}
	return strings.Join(methodsPrepared, ",")
}
func findMethodRoutes(method source.InterfaceMethod) []map[string]string {
	var methodRoutes []map[string]string
	for _, annotation := range method.Annotations() {
		if annotation.Name == "http" {
			methodRoutes = append(methodRoutes, map[string]string{
				"Name":    annotation.Get("name").String(),
				"Route":   annotation.Get("route").String(),
				"Methods": routeMethods(annotation.Get("methods").String()),
			})
		}
	}
	return methodRoutes
}
