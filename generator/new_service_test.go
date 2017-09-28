package generator

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNewNewService(t *testing.T) {
	setDefaults()
	g := NewNewService("test").(*NewService)
	err := g.Generate()
	Convey("Test if generator generates the service without errors", t, func() {
		So(err, ShouldBeNil)
		Convey("Test if file destination and file path are right", func() {
			So(g.destPath, ShouldEqual, "test/pkg/service")
			So(g.filePath, ShouldEqual, "test/pkg/service/service.go")
		})
		Convey("Test if file is generated", func() {
			f, _ := g.fs.ReadFile("test/pkg/service/service.go")
			So(f, ShouldContainSubstring, "TestService")
		})
	})
}
