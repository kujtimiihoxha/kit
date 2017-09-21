package generator

import (
	"fmt"
	"path"
	"testing"

	"github.com/kujtimiihoxha/kit/fs"
	"github.com/kujtimiihoxha/kit/utils"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
)

func TestGenerateServiceEndpointsBase_Generate(t *testing.T) {
	svcName := "test"
	setDefaults()
	g := newGenerateServiceEndpointsBase(svcName, getTestServiceInterface(svcName))
	g.Generate()
	dest := fmt.Sprintf(viper.GetString("gk_endpoint_path_format"), utils.ToLowerSnakeCase(svcName))
	filePath := path.Join(dest, viper.GetString("gk_endpoint_base_file_name"))

	fl, err := fs.Get().ReadFile(filePath)
	Convey("Test if generator generates file without errors", t, func() {
		So(err, ShouldBeNil)
		Convey("Test if Endpoints struct is created how it should", func() {
			So(fl, ShouldContainSubstring, `type Endpoints struct {
	FooEndpoint endpoint.Endpoint
}`)
		})
		Convey("Test if New method is created how it should", func() {
			So(fl, ShouldContainSubstring, `func New(s service.TestService, mdw map[string][]endpoint.Middleware) Endpoints {
	eps := Endpoints{FooEndpoint: MakeFooEndpoint(s)}
	for _, m := range mdw["Foo"] {
		eps.FooEndpoint = m(eps.FooEndpoint)
	}
	return eps
}`)
		})
	})
}
