package template

import (
	"github.com/go-services/code"
	"github.com/ozgio/strutil"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"
)

var CustomFunctions = template.FuncMap{
	"toLowerFirst": ToLowerFirst,
	"toTitle":      toTitle,
	"paramsString": paramsString,
}

func paramsString(params []code.Parameter) string {
	var paramsStrings []string
	for _, p := range params {
		paramsStrings = append(paramsStrings, p.String())
	}
	return strings.Join(paramsStrings, ", ")
}

func ToLowerFirst(text string) string {
	if len(text) > 0 {
		r, size := utf8.DecodeRuneInString(text)
		if r != utf8.RuneError || size > 1 {
			lo := unicode.ToLower(r)
			if lo != r {
				text = string(lo) + text[size:]
			}
		}
	}
	return text
}

func toTitle(text string) string {
	return strings.Title(strutil.ToCamelCase(text))
}
