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
				result = fmt.Sprintf("%s: parse succeed", function.Name())
				return
			})
		}
		serviceCodeFileUnits = append(serviceCodeFileUnits, codes.Unit(codes.NewServiceFile(service)))
	}
	serviceCodeFileUnits = append(serviceCodeFileUnits, codes.Unit(codes.NewServicesFile(filepath.ToSlash(filepath.Join(mod.Dir, "modules")), services)))
	process.Add("parsing", processes.ParallelUnits(functionParseUnits...))
	process.Add("writing", processes.ParallelUnits(serviceCodeFileUnits...))
	return
}
