package module

import (
	"fmt"
	"github.com/aacfactory/cases"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/forg/files"
	"go/ast"
	"path/filepath"
	"sort"
	"strings"
)

type Component struct {
	Indent string
}

type Components []*Component

func (components Components) Len() int {
	return len(components)
}

func (components Components) Less(i, j int) bool {
	return components[i].Indent < components[j].Indent
}

func (components Components) Swap(i, j int) {
	components[i], components[j] = components[j], components[i]
	return
}

func tryLoadService(mod *Module, path string) (service *Service, has bool, err error) {
	f, readErr := mod.sources.ReadFile(path, "doc.go")
	if readErr != nil {
		err = errors.Warning("forg: parse service failed").WithCause(readErr).WithMeta("path", path).WithMeta("file", "doc.go")
		return
	}
	_, pkg := filepath.Split(path)
	if pkg != f.Name.Name {
		err = errors.Warning("forg: parse service failed").WithCause(errors.Warning("pkg must be same as dir name")).WithMeta("path", path).WithMeta("file", "doc.go")
		return
	}

	doc := f.Doc.Text()
	if doc == "" {
		return
	}
	annotations, parseAnnotationsErr := ParseAnnotations(doc)
	if parseAnnotationsErr != nil {
		err = errors.Warning("forg: parse service failed").WithCause(parseAnnotationsErr).WithMeta("path", path).WithMeta("file", "doc.go")
		return
	}
	name, hasName := annotations.Get("service")
	if !hasName {
		return
	}
	has = true
	_, hasInternal := annotations.Get("internal")
	title, _ := annotations.Get("title")
	Description, _ := annotations.Get("description")

	service = &Service{
		mod:         mod,
		Path:        path,
		Name:        strings.ToLower(name),
		Internal:    hasInternal,
		Title:       title,
		Description: Description,
		Functions:   make([]*Function, 0, 1),
		Components:  make([]*Component, 0, 1),
	}
	loadFunctionsErr := service.loadFunctions()
	if loadFunctionsErr != nil {
		err = errors.Warning("forg: parse service failed").WithCause(loadFunctionsErr).WithMeta("path", path).WithMeta("file", "doc.go")
		return
	}
	loadComponentsErr := service.loadComponents()
	if loadComponentsErr != nil {
		err = errors.Warning("forg: parse service failed").WithCause(loadComponentsErr).WithMeta("path", path).WithMeta("file", "doc.go")
		return
	}
	sort.Sort(service.Functions)
	sort.Sort(service.Components)
	return
}

type Services []*Service

func (services Services) Len() int {
	return len(services)
}

func (services Services) Less(i, j int) bool {
	return services[i].Name < services[j].Name
}

func (services Services) Swap(i, j int) {
	services[i], services[j] = services[j], services[i]
	return
}

type Service struct {
	mod         *Module
	Path        string
	Name        string
	Internal    bool
	Title       string
	Description string
	Functions   Functions
	Components  Components
}

func (service *Service) loadFunctions() (err error) {
	err = service.mod.sources.ReadDir(service.Path, func(file *ast.File, filename string) (err error) {
		if file.Decls == nil || len(file.Decls) == 0 {
			return
		}
		fileImports := newImportsFromAstFileImports(file.Imports)
		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if funcDecl.Recv != nil {
				continue
			}
			if funcDecl.Doc == nil {
				continue
			}
			doc := funcDecl.Doc.Text()
			if !strings.Contains(doc, "@fn") {
				continue
			}
			ident := funcDecl.Name.Name
			if ast.IsExported(ident) {
				err = errors.Warning("forg: parse func name failed").
					WithMeta("file", filename).
					WithMeta("func", ident).
					WithCause(errors.Warning("forg: func name must not be exported"))
				return
			}
			nameAtoms, parseNameErr := cases.LowerCamel().Parse(ident)
			if parseNameErr != nil {
				err = errors.Warning("forg: parse func name failed").
					WithMeta("file", filename).
					WithMeta("func", ident).
					WithCause(parseNameErr)
				return
			}
			proxyIdent := cases.Camel().Format(nameAtoms)
			constIdent := fmt.Sprintf("_%sFn", ident)
			annotations, parseAnnotationsErr := ParseAnnotations(doc)
			if parseAnnotationsErr != nil {
				err = errors.Warning("forg: parse func annotations failed").
					WithMeta("file", filename).
					WithMeta("func", ident).
					WithCause(parseAnnotationsErr)
				return
			}
			service.Functions = append(service.Functions, &Function{
				mod:             service.mod,
				hostFileImports: fileImports,
				hostServiceName: service.Name,
				path:            service.Path,
				decl:            funcDecl,
				Ident:           funcDecl.Name.Name,
				ConstIdent:      constIdent,
				ProxyIdent:      proxyIdent,
				Annotations:     annotations,
				Param:           nil,
				Result:          nil,
			})
		}
		return
	})
	return
}

func (service *Service) loadComponents() (err error) {
	componentsPath := fmt.Sprintf("%s/components", service.Path)
	dir, dirErr := service.mod.sources.destinationPath(componentsPath)
	if dirErr != nil {
		err = errors.Warning("forg: read service components dir failed").WithCause(dirErr).WithMeta("service", service.Path)
		return
	}
	if !files.ExistFile(dir) {
		return
	}
	readErr := service.mod.sources.ReadDir(componentsPath, func(file *ast.File, filename string) (err error) {
		if file.Decls == nil || len(file.Decls) == 0 {
			return
		}
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			specs := genDecl.Specs
			if specs == nil || len(specs) == 0 {
				continue
			}
			for _, spec := range specs {
				ts, tsOk := spec.(*ast.TypeSpec)
				if !tsOk {
					continue
				}
				doc := ""
				if ts.Doc == nil || ts.Doc.Text() == "" {
					if len(specs) == 1 && genDecl.Doc != nil && genDecl.Doc.Text() != "" {
						doc = genDecl.Doc.Text()
					}
				} else {
					doc = ts.Doc.Text()
				}
				if !strings.Contains(doc, "@component") {
					continue
				}
				ident := ts.Name.Name
				if !ast.IsExported(ident) {
					err = errors.Warning("forg: parse component name failed").
						WithMeta("file", filename).
						WithMeta("component", ident).
						WithCause(errors.Warning("forg: component name must be exported"))
					return
				}
				service.Components = append(service.Components, &Component{
					Indent: ident,
				})
			}
		}
		return
	})
	if readErr != nil {
		err = errors.Warning("forg: read service components dir failed").WithCause(readErr).WithMeta("service", service.Path)
		return
	}
	return
}
