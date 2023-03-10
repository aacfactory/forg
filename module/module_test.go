package module_test

import (
	"context"
	"fmt"
	"github.com/aacfactory/forg/module"
	"testing"
)

func TestParseModule(t *testing.T) {
	path := "D:\\studio\\workspace\\go\\src\\tkh.com\\tkh\\go.mod"
	mod, createErr := module.New(path)
	if createErr != nil {
		t.Errorf("%+v", createErr)
		return
	}
	fmt.Println(mod)

}

func TestFunction_Parse(t *testing.T) {
	path := `D:\studio\workspace\go\src\github.com\aacfactory\fns-example\standalone\go.mod`
	mod, modErr := module.New(path)
	if modErr != nil {
		t.Errorf("%+v", modErr)
		return
	}
	fmt.Println(mod.String())
	services, servicesErr := mod.Services()
	if servicesErr != nil {
		t.Errorf("%+v", servicesErr)
		return
	}
	fmt.Println("services:", len(services))
	if len(services) == 0 {
		return
	}
	service := services[0]
	fmt.Println("service:", service.Name, "functions:", len(service.Functions), "components:", len(service.Components))
	if len(service.Functions) == 0 {
		return
	}
	fn := service.Functions[0]
	parseErr := fn.Parse(context.TODO())
	if parseErr != nil {
		t.Errorf("%+v", parseErr)
		return
	}
	fmt.Println("fn:", fn.Name(), fn.Param.String(), fn.Result.String())
}
