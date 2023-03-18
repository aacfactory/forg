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

type Options struct {
	Workspace string
}

type Option func(options *Options) (err error)

func WithWorkspace(workspace string) Option {
	return func(options *Options) (err error) {
		workspace = strings.TrimSpace(workspace)
		if workspace == "" {
			err = errors.Warning("forg: workspace option is invalid")
			return
		}
		options.Workspace = workspace
		return
	}
}

func Load(dir string, options ...Option) (project *Project, err error) {
	opt := &Options{
		Workspace: "",
	}
	if options != nil && len(options) > 0 {
		for _, option := range options {
			optionErr := option(opt)
			if optionErr != nil {
				err = errors.Warning("forg: load project failed").WithCause(optionErr)
				return
			}
		}
	}
	if dir == "" {
		err = errors.Warning("forg: load project failed").WithCause(errors.Warning("project dir is nil"))
		return
	}
	moduleFilename := filepath.Join(dir, "go.mod")
	var mod *module.Module
	if opt.Workspace != "" {
		mod, err = module.NewWithWork(moduleFilename, opt.Workspace)
	} else {
		mod, err = module.New(moduleFilename)
	}
	if err != nil {
		err = errors.Warning("forg: load project failed").WithCause(err)
		return
	}
	project = &Project{
		Mod: mod,
	}
	return
}

type Project struct {
	Mod *module.Module
}

func (project *Project) Coding(ctx context.Context) (controller processes.ProcessController, err error) {
	parseErr := project.Mod.Parse(ctx)
	if parseErr != nil {
		err = errors.Warning("forg: project coding failed").WithCause(parseErr)
		return
	}
	services, servicesErr := project.Mod.Services()
	if servicesErr != nil {
		err = errors.Warning("forg: project coding failed").WithCause(servicesErr)
		return
	}
	process := processes.New()
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
	process.Add("services: deploying", codes.Unit(codes.NewDeploysFile(filepath.ToSlash(filepath.Join(project.Mod.Dir, "modules")), services)))
	controller = process
	return
}
