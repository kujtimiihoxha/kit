package fs

import (
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/Songmu/prompter"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

type FileSystem interface {
	init(dir string)
	ReadFile(path string) (string, error)
	WriteFile(path string, data string, force bool) error
	Mkdir(path string) error
	MkdirAll(path string) error
	FilePathSeparator() string
	Exists(path string) (bool, error)
}

var defaultFs *DefaultFs

type DefaultFs struct {
	Fs afero.Fs
}

func (f *DefaultFs) init(dir string) {
	var inFs afero.Fs
	if viper.GetBool("gk_testing") {
		inFs = afero.NewMemMapFs()
	} else {
		if viper.GetString("gk_folder") != "" {
			inFs = afero.NewBasePathFs(afero.NewOsFs(), viper.GetString("gk_folder"))
		} else {
			inFs = afero.NewOsFs()
		}
	}
	if dir != "" {
		f.Fs = afero.NewBasePathFs(inFs, dir)
	} else {
		f.Fs = inFs
	}
}

func (f *DefaultFs) ReadFile(path string) (string, error) {
	d, err := afero.ReadFile(f.Fs, path)
	return string(d), err
}

func (f *DefaultFs) WriteFile(path string, data string, force bool) error {

	if b, _ := f.Exists(path); b && !(viper.GetBool("gk_force_override") || force) {
		s, _ := f.ReadFile(path)
		if s == data {
			logrus.Warnf("`%s` exists and is identical it will be ignored", path)
			return nil
		}
		b := prompter.YN(fmt.Sprintf("`%s` already exists do you want to override it ?", path), false)
		if !b {
			return nil
		}
	}
	return afero.WriteFile(f.Fs, path, []byte(data), os.ModePerm)
}

func (f *DefaultFs) Mkdir(path string) error {
	return f.Fs.Mkdir(path, os.ModePerm)
}

func (f *DefaultFs) MkdirAll(path string) error {
	return f.Fs.MkdirAll(path, os.ModePerm)
}
func (f *DefaultFs) FilePathSeparator() string {
	return afero.FilePathSeparator
}
func (f *DefaultFs) Exists(path string) (bool, error) {
	return afero.Exists(f.Fs, path)
}
func NewDefaultFs(dir string) *DefaultFs {
	dfs := &DefaultFs{}
	dfs.init(dir)
	defaultFs = dfs
	return dfs
}

func Get() *DefaultFs {
	if defaultFs == nil {
		return NewDefaultFs("")
	} else {
		return defaultFs
	}
}
