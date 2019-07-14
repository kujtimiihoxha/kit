package template

import (
	"bytes"
	"fmt"
	"golang.org/x/tools/imports"
	"io/ioutil"
	"text/template"
)

var FS *FileSystem

func CompileFromPath(tplPath string, data interface{}) (string, error) {
	file, err := FS.Open(tplPath)
	if err != nil {
		return "", err
	}
	buf, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}
	t := template.Must(template.New("template").Funcs(CustomFunctions).Parse(string(buf)))
	templateBuffer := bytes.NewBufferString("")
	err = t.Execute(templateBuffer, data)
	if err != nil {
		return "", err
	}
	return templateBuffer.String(), err
}
func CompileGoFromPath(tplPath string, data interface{}) (string, error) {
	src, err := CompileFromPath(tplPath, data)
	fmt.Println(src, err)
	prettyCode, err := imports.Process("template.go", []byte(src), nil)
	return string(prettyCode), err
}

func FromPath(tplPath string) (string, error) {
	file, err := FS.Open(tplPath)
	if err != nil {
		return "", err
	}
	buf, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}
