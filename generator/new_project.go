package generator

import (
	"github.com/spf13/afero"
	"kit/fs"
	"kit/template"
	"strings"

	"github.com/ozgio/strutil"
)

func NewProject(name string) error {
	appFs := fs.AppFs()

	// we should remove the '_' because of this guide https://blog.golang.org/package-names
	moduleName := strings.ReplaceAll(strutil.ToSnakeCase(name), "_", "")

	if err := fs.CreateFolder(moduleName, appFs); err != nil {
		return err
	}

	gomod, err := template.CompileFromPath("/assets/templates/project/go.mod.gotmpl", map[string]string{
		"ProjectModule": moduleName,
	})
	if err != nil {
		return err
	}
	projectFs := afero.NewBasePathFs(appFs, moduleName)

	gitignore, err := template.FromPath("/assets/project/gitignore")
	if err != nil {
		return err
	}
	kitJson, err := template.CompileFromPath("/assets/templates/project/kit.json.gotmpl", map[string]string{
		"ProjectModule": moduleName,
	})
	if err != nil {
		return err
	}
	if err := fs.CreateFile(".gitignore", gitignore, projectFs); err != nil {
		return err
	}
	if err := fs.CreateFile("go.mod", gomod, projectFs); err != nil {
		return err
	}
	if err := fs.CreateFile("kit.json", kitJson, projectFs); err != nil {
		return err
	}

	return nil
}
