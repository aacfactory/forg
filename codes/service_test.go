package codes_test

import (
	"context"
	"github.com/aacfactory/forg/codes"
	"github.com/aacfactory/forg/module"
	"testing"
)

func TestNewServiceFile(t *testing.T) {
	ctx := context.TODO()
	path := `D:\studio\workspace\go\src\github.com\aacfactory\fns-example\standalone\go.mod`
	mod, modErr := module.New(path)
	if modErr != nil {
		t.Errorf("%+v", modErr)
		return
	}
	parseErr := mod.Parse(ctx)
	if parseErr != nil {
		t.Errorf("%+v", parseErr)
		return
	}
	services, servicesErr := mod.Services()
	if servicesErr != nil {
		t.Errorf("%+v", servicesErr)
		return
	}
	for _, service := range services {
		functions := service.Functions
		for _, function := range functions {
			parseFnErr := function.Parse(ctx)
			if parseFnErr != nil {
				t.Errorf("%+v", parseFnErr)
				return
			}
		}
	}
	for _, service := range services {
		sf := codes.NewServiceFile(service)
		writeErr := sf.Write(ctx)
		if writeErr != nil {
			t.Errorf("%+v", writeErr)
			return
		}
	}

	deploys := codes.NewDeploysFile(`D:\studio\workspace\go\src\github.com\aacfactory\fns-example\standalone\modules`, services)
	writeErr := deploys.Write(ctx)
	if writeErr != nil {
		t.Errorf("%+v", writeErr)
		return
	}
}
