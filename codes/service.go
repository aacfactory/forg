package codes

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/forg/module"
	"github.com/aacfactory/gcg"
	"os"
	"path/filepath"
	"strings"
)

type ServiceFile struct {
	service *module.Service
}

func (s *ServiceFile) Name() (name string) {
	name = filepath.ToSlash(filepath.Join(s.service.Dir, "fns.go"))
	return
}

func (s *ServiceFile) Write(ctx context.Context) (err error) {
	file := gcg.NewFileWithoutNote(s.service.Path[strings.LastIndex(s.service.Path, "/")+1:])

	file.FileComments("NOTE: this file was been automatically generated, DONT EDIT IT\n")

	packages, importsErr := s.importsCode(ctx)
	if importsErr != nil {
		err = errors.Warning("forg: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(importsErr)
		return
	}
	if packages != nil && len(packages) > 0 {
		for _, importer := range packages {
			file.AddImport(importer)
		}
	}

	names, namesErr := s.constFunctionNamesCode(ctx)
	if namesErr != nil {
		err = errors.Warning("forg: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(namesErr)
		return
	}
	file.AddCode(names)

	proxies, proxiesErr := s.proxyFunctionsCode(ctx)
	if proxiesErr != nil {
		err = errors.Warning("forg: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(proxiesErr)
		return
	}
	file.AddCode(proxies)

	service, serviceErr := s.serviceCode(ctx)
	if serviceErr != nil {
		err = errors.Warning("forg: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(serviceErr)
		return
	}
	file.AddCode(service)

	writer, openErr := os.OpenFile(s.Name(), os.O_TRUNC|os.O_WRONLY, 0600)
	if openErr != nil {
		err = errors.Warning("forg: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(openErr)
		return
	}
	renderErr := file.Render(writer)
	if renderErr != nil {
		_ = writer.Close()
		err = errors.Warning("forg: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(renderErr)
		return
	}
	_ = writer.Close()
	return
}

func (s *ServiceFile) importsCode(ctx context.Context) (packages []*gcg.Package, err error) {

	return
}

func (s *ServiceFile) constFunctionNamesCode(ctx context.Context) (code gcg.Code, err error) {

	return
}

func (s *ServiceFile) proxyFunctionsCode(ctx context.Context) (code gcg.Code, err error) {

	return
}

func (s *ServiceFile) serviceCode(ctx context.Context) (code gcg.Code, err error) {

	return
}
