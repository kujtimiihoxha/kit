package service

import (
	"github.com/go-services/code"
	"unicode"
	"unicode/utf8"
)

func IsExported(name string) bool {
	ch, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(ch)
}

func HasCorrectParamNumber(params []code.Parameter) bool {
	return len(params) > 0 && len(params) <= 2
}
func HasCorrectParams(params []code.Parameter) bool {
	return params[0].Type.Qualifier == "Context" &&
		params[0].Type.Import.Path == "context" &&
		!params[0].Type.Pointer &&
		!params[0].Type.Variadic
}

func HasCorrectResults(params []code.Parameter) bool {
	if len(params) == 1 {
		return params[0].Type.Qualifier == "error"
	} else {
		return false
	}
}
