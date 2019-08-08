package fs

import (
	"fmt"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

var fs afero.Fs

// AppFs returns the current location file system.
//  if we are in testing mode it returns a memory
func AppFs() afero.Fs {
	if viper.Get("testFs") != nil {
		return viper.Get("testFs").(afero.Fs)
	}
	if fs == nil {
		fs = afero.NewOsFs()
	}
	return fs
}

func CreateFolder(path string, fs afero.Fs) error {
	b, err := afero.Exists(fs, path)
	if err != nil {
		return err
	} else if b {
		return fmt.Errorf("folder with the name `%s` already exists", path)
	}
	return fs.Mkdir(path, 0755)
}

func CreateFile(path, data string, fs afero.Fs) error {
	return afero.WriteFile(fs, path, []byte(data), 0644)
}

func ReadFile(path string, fs afero.Fs) (string, error) {
	b, err := afero.ReadFile(fs, path)
	return string(b), err
}
