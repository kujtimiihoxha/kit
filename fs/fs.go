package fs

import (
	"fmt"
	"os"

	"github.com/Songmu/prompter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

var defaultFs *KitFs

// KitFs wraps an afero.Fs
type KitFs struct {
	Fs afero.Fs
}

func (f *KitFs) init(dir string) {
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

// ReadFile reads the file from `path` and returns the content in string format
// or returns an error if it occurs.
func (f *KitFs) ReadFile(path string) (string, error) {
	d, err := afero.ReadFile(f.Fs, path)
	return string(d), err
}

// WriteFile writs a file to the `path` with `data` as content, if `force` is set
// to true it will override the file if it already exists.
func (f *KitFs) WriteFile(path string, data string, force bool) error {
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

// Mkdir creates a directory.
func (f *KitFs) Mkdir(dir string) error {
	return f.Fs.Mkdir(dir, os.ModePerm)
}

// MkdirAll creates a directory and its parents if they don't exist.
func (f *KitFs) MkdirAll(path string) error {
	return f.Fs.MkdirAll(path, os.ModePerm)
}

// Exists returns true,nil if the dir/file exists or false,nil if
// the dir/file does not exist, it will return an error if something
// went wrong.
func (f *KitFs) Exists(path string) (bool, error) {
	return afero.Exists(f.Fs, path)
}

// NewDefaultFs creates a KitFs with `dir` as root.
func NewDefaultFs(dir string) *KitFs {
	dfs := &KitFs{}
	dfs.init(dir)
	defaultFs = dfs
	return dfs
}

// Get returns a new KitFs if it was not initiated before or
// it returns the existing defaultFs if it is initiated.
func Get() *KitFs {
	if defaultFs == nil {
		return NewDefaultFs("")
	}
	return defaultFs
}
