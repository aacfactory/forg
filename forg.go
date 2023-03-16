package forg

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/forg/codes"
	"github.com/aacfactory/forg/module"
	"github.com/aacfactory/forg/processes"
	"path/filepath"
	"strings"
)

func New(path string) (process *processes.Process, err error) {
	path = strings.TrimSpace(path)
	if path == "" {
		err = errors.Warning("forg: new failed").WithCause(errors.Warning("forg: path is required"))
		return
	}
	mod, modErr := module.New(path)
	if modErr != nil {
		err = errors.Warning("forg: new failed").WithCause(modErr)
		return
	}
	services, servicesErr := mod.Services()
	if servicesErr != nil {
		err = errors.Warning("forg: new failed").WithCause(servicesErr)
		return
	}
	process = processes.New()
	functionParseUnits := make([]processes.Unit, 0, 1)
	serviceCodeFileUnits := make([]processes.Unit, 0, 1)
	for _, service := range services {
		for _, function := range service.Functions {
			functionParseUnits = append(functionParseUnits, func(ctx context.Context) (result interface{}, err error) {
				err = function.Parse(ctx)
				if err != nil {
					return
				}
				result = fmt.Sprintf("%s/%s: parse succeed", function.HostServiceName(), function.Name())
				return
			})
		}
		serviceCodeFileUnits = append(serviceCodeFileUnits, codes.Unit(codes.NewServiceFile(service)))
	}
	process.Add("services: parsing", functionParseUnits...)
	process.Add("services: writing", serviceCodeFileUnits...)
	process.Add("deploys : writing", codes.Unit(codes.NewDeploysFile(filepath.ToSlash(filepath.Join(mod.Dir, "modules")), services)))
	return
}
