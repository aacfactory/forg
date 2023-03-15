package codes

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/forg/module"
	"github.com/aacfactory/gcg"
	"os"
	"path/filepath"
)

func NewServicesFile(dir string, services module.Services) (file CodeFile) {
	file = &ServicesFile{
		filename: filepath.ToSlash(filepath.Join(dir, "services.go")),
		services: services,
	}
	return
}

type ServicesFile struct {
	filename string
	services module.Services
}

func (s *ServicesFile) Name() (name string) {
	name = s.filename
	return
}

func (s *ServicesFile) Write(ctx context.Context) (err error) {
	if s.filename == "" {
		return
	}
	if ctx.Err() != nil {
		err = errors.Warning("forg: services write failed").
			WithMeta("services", s.Name()).
			WithCause(ctx.Err())
		return
	}

	file := gcg.NewFileWithoutNote("modules")
	file.FileComments("NOTE: this file was been automatically generated, DONT EDIT IT\n")

	fn := gcg.Func()
	fn.Name("services")
	fn.AddResult("v", gcg.Token("[]service.Service", gcg.NewPackage("github.com/aacfactory/fns/service")))
	body := gcg.Statements()
	if s.services != nil && s.services.Len() > 0 {
		body.Token("v = append(").Line()
		body.Tab().Token("v").Symbol(",").Line()
		for _, service := range s.services {
			body.Tab().Token(fmt.Sprintf("%s.Service()", service.PathIdent), gcg.NewPackage(service.Path)).Symbol(",").Line()
		}
		body.Token(")").Line()
	}
	body.Return().Line()
	fn.Body(body)
	file.AddCode(fn.Build())

	writer, openErr := os.OpenFile(s.Name(), os.O_TRUNC|os.O_RDWR|os.O_SYNC, 0600)
	if openErr != nil {
		err = errors.Warning("forg: code file write failed").
			WithMeta("kind", "services").WithMeta("file", s.Name()).
			WithCause(openErr)
		return
	}
	renderErr := file.Render(writer)
	if renderErr != nil {
		_ = writer.Close()
		err = errors.Warning("forg: code file write failed").
			WithMeta("kind", "services").WithMeta("file", s.Name()).
			WithCause(renderErr)
		return
	}
	_ = writer.Close()
	return
}
