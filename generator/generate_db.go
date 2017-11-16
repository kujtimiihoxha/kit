package generator

import (
	"fmt"
	"path"

	"github.com/kujtimiihoxha/kit/fs"
	"github.com/kujtimiihoxha/kit/utils"
	"github.com/spf13/viper"
)

type generateDB struct {
	BaseGenerator
	name     string
	destPath string
	filePath string
}

func NewGenerateDB(name string) Gen {
	t := &generateDB{
		destPath: fmt.Sprintf(viper.GetString("gk_db_path_format"), utils.ToLowerSnakeCase(name)),
	}
	t.filePath = path.Join(t.destPath, viper.GetString("gk_db_file_name"))
	t.fs = fs.Get()
	return t
}
func (g *generateDB) Generate() (err error) {
	err = g.CreateFolderStructure(g.destPath)
	if err != nil {
		return err
	}
	if b, err := g.fs.Exists(g.filePath); err != nil {
		return err
	} else if b {
		return nil
	}
	tmp := `
package db

import (
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/kujtimiihoxha/shqip-core/config"
)

var session *gorm.DB

func Session() *gorm.DB {
	if session != nil {
		return session
	}
	s, err := gorm.Open("mysql", connectionString())
	if err != nil {
		panic(err)
	}
	session = s
	return session
}

func connectionString() string {
	cf := config.Get()
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", cf.DBUser, cf.DBPassword, cf.DBHost, cf.DBPort, cf.DBName)
}

func Close() error {
	if session != nil {
		return session.Close()
	}
	return nil
}
`
	tmp, err = utils.GoImportsSource(g.destPath, tmp)
	if err != nil {
		return err
	}
	return g.fs.WriteFile(g.filePath, tmp, true)
	return nil
}
