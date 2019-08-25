package service

import (
	"errors"
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
func HasCorrectParams(params []code.Parameter) (bool, error) {
	if len(params) != 1 && len(params) != 2 {
		return false, errors.New("method must except either the context or the context and the request struct")
	}
	if !(params[0].Type.Qualifier == "Context" &&
		params[0].Type.Import.Path == "context") &&
		params[0].Type.Pointer &&
		params[0].Type.Variadic {
		return false, errors.New("the first parameter of the method needs to be the context")
	}
	if len(params) == 2 && !IsExported(params[1].Type.Qualifier) {
		return false, errors.New("request needs to be an exported structure")
	}
	return true, nil
}

func HasCorrectResults(params []code.Parameter) (bool, error) {
	if (len(params) != 1 && len(params) != 2) ||
		len(params) == 1 && params[0].Type.Qualifier != "error" ||
		len(params) == 2 && params[1].Type.Qualifier != "error" ||
		len(params) == 2 && !params[0].Type.Pointer ||
		len(params) == 2 && !IsExported(params[0].Type.Qualifier) {
		return false, errors.New("method must return either the error or the response pointer and the error")
	}
	return true, nil
}
