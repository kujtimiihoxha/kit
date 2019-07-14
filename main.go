package main

import (
	"fmt"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"kit/fs"
	"kit/generator"
)

func main() {
	//fmt.Println(generator.NewProject("test"))
	viper.Set("testFs", afero.NewBasePathFs(fs.AppFs(), "test"))
	//fmt.Println(generator.NewService("abc"))
	fmt.Println(generator.GenerateService("abc"))
}
