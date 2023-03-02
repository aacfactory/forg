package module

import (
	"fmt"
	"github.com/aacfactory/cases"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/forg/files"
	"go/ast"
	"os"
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

func tryLoadService(mod *Module, filename string) (service *Service, has bool, err error) {
	f, parseErr := files.ParseSource(filename)
	if parseErr != nil {
		err = errors.Warning("forg: parse service failed").WithCause(parseErr).WithMeta("file", filename)
		return
	}
	doc := f.Doc.Text()
	if doc == "" {
		return
	}
	annotations, parseAnnotationsErr := ParseAnnotations(doc)
	if parseAnnotationsErr != nil {
		err = errors.Warning("forg: parse service failed").WithCause(parseAnnotationsErr).WithMeta("file", filename)
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
		Dir:         filepath.ToSlash(filepath.Dir(filename)),
		Name:        strings.ToLower(name),
		Internal:    hasInternal,
		Title:       title,
		Description: Description,
		Functions:   make([]*Function, 0, 1),
		Components:  make([]*Component, 0, 1),
	}
	loadFunctionsErr := service.loadFunctions()
	if loadFunctionsErr != nil {
		err = errors.Warning("forg: parse service failed").WithCause(loadFunctionsErr).WithMeta("file", filename)
		return
	}
	loadComponentsErr := service.loadComponents()
	if loadComponentsErr != nil {
		err = errors.Warning("forg: parse service failed").WithCause(loadComponentsErr).WithMeta("file", filename)
		return
	}
	sort.Sort(service.Functions)
	sort.Sort(service.Components)
	return
}

type Service struct {
	mod         *Module
	Dir         string
	Name        string
	Internal    bool
	Title       string
	Description string
	Functions   Functions
	Components  Components
}

func (service *Service) loadFunctions() (err error) {
	entries, readDirErr := os.ReadDir(service.Dir)
	if readDirErr != nil {
		err = errors.Warning("forg: read service dir failed").WithMeta("dir", service.Dir).WithCause(readDirErr)
		return
	}
	if entries == nil || len(entries) == 0 {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "doc.go" {
			continue
		}
		if filepath.Ext(entry.Name()) != ".go" {
			continue
		}
		filename := filepath.Join(service.Dir, entry.Name())
		file, parseErr := files.ParseSource(filename)
		if parseErr != nil {
			err = errors.Warning("forg: parse go source file failed").WithMeta("file", filename).WithCause(parseErr)
			return
		}
		if file.Decls == nil || len(file.Decls) == 0 {
			continue
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
			name := funcDecl.Name.Name
			if name[0] < 'a' || name[0] > 'z' {
				err = errors.Warning("forg: parse func name failed").
					WithMeta("file", filename).
					WithMeta("func", funcDecl.Name.Name).
					WithCause(errors.Warning("forg: func name must be lower camel format"))
				return
			}
			nameAtoms, parseNameErr := cases.LowerCamel().Parse(name)
			if parseNameErr != nil {
				err = errors.Warning("forg: parse func name failed").
					WithMeta("file", filename).
					WithMeta("func", funcDecl.Name.Name).
					WithCause(parseNameErr)
				return
			}
			proxyIdent := cases.Camel().Format(nameAtoms)
			constIdent := fmt.Sprintf("_%sFn", name)
			annotations, parseAnnotationsErr := ParseAnnotations(doc)
			if parseAnnotationsErr != nil {
				err = errors.Warning("forg: parse func annotations failed").
					WithMeta("file", filename).
					WithMeta("func", funcDecl.Name.Name).
					WithCause(parseAnnotationsErr)
				return
			}
			service.Functions = append(service.Functions, &Function{
				mod:             service.mod,
				hostFileImports: fileImports,
				decl:            funcDecl,
				Ident:           funcDecl.Name.Name,
				ConstIdent:      constIdent,
				ProxyIdent:      proxyIdent,
				Annotations:     annotations,
				Param:           nil,
				Result:          nil,
			})
		}
	}
	return
}

func (service *Service) loadComponents() (err error) {
	componentsDir := filepath.Join(service.Dir, "components")
	if !files.ExistFile(componentsDir) {
		return
	}
	entries, readDirErr := os.ReadDir(componentsDir)
	if readDirErr != nil {
		err = errors.Warning("forg: read service components dir failed").WithMeta("dir", componentsDir).WithCause(readDirErr)
		return
	}
	if entries == nil || len(entries) == 0 {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "doc.go" {
			continue
		}
		if filepath.Ext(entry.Name()) != ".go" {
			continue
		}
		filename := filepath.Join(componentsDir, entry.Name())
		file, parseErr := files.ParseSource(filename)
		if parseErr != nil {
			err = errors.Warning("forg: parse go source file failed").WithMeta("file", componentsDir).WithCause(parseErr)
			return
		}
		if file.Decls == nil || len(file.Decls) == 0 {
			continue
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
				if ident[0] < 'A' || ident[0] > 'Z' {
					err = errors.Warning("forg: parse component name failed").
						WithMeta("file", filename).
						WithMeta("component", ident).
						WithCause(errors.Warning("forg: component name must be upper camel format"))
					return
				}
				service.Components = append(service.Components, &Component{
					Indent: ident,
				})
			}
		}

	}
	return
}
