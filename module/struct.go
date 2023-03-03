package module

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/forg/files"
	"go/ast"
	"golang.org/x/sync/singleflight"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Structs struct {
	mod    *Module
	values sync.Map
	group  singleflight.Group
}

func (sts *Structs) load(ctx context.Context, path string, name string) (v *Struct, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("forg: get struct failed").WithCause(ctx.Err())
		return
	}
	key := fmt.Sprintf("structs:%s.%s", path, name)
	vv, has := sts.values.Load(key)
	if has {
		v = vv.(*Struct)
		return
	}
	vv = ctx.Value(key)
	if vv != nil {
		v = vv.(*Struct)
		return
	}
	cached := ctx.Value(key)
	if cached != nil {
		v = vv.(*Struct)
		return
	}
	st, readErr, _ := sts.group.Do(key, func() (v interface{}, err error) {
		st := &Struct{
			mod:    sts.mod,
			Path:   path,
			Name:   name,
			Fields: make([]*StructField, 0, 1),
		}
		ctx = context.WithValue(ctx, key, st)
		parseErr := st.parse(ctx)
		if parseErr != nil {
			err = parseErr
			return
		}
		sts.values.Store(key, st)
		v = st
		return
	})
	if readErr != nil {
		err = errors.Warning("forg: get struct failed").WithMeta("name", key).WithCause(readErr)
		return
	}
	v = st.(*Struct)
	return
}

type StructField struct {
	Name        string
	Annotations Annotations
	Tags        map[string]string
	Element     *Element
}

type Struct struct {
	mod         *Module
	fileImports Imports
	Path        string
	Name        string
	Annotations Annotations
	Fields      []*StructField
}

func (st *Struct) parse(ctx context.Context) (err error) {
	if ctx.Err() != nil {
		err = errors.Warning("forg: parse struct failed").WithCause(ctx.Err()).
			WithMeta("path", st.Path).WithMeta("name", st.Name)
		return
	}
	typ, fileImports, typeErr := st.loadType(ctx)
	if typeErr != nil {
		err = errors.Warning("forg: parse struct failed").WithCause(typeErr).
			WithMeta("path", st.Path).WithMeta("name", st.Name)
		return
	}
	if typ == nil {
		err = errors.Warning("forg: parse struct failed").WithCause(errors.Warning("struct type was not found")).
			WithMeta("path", st.Path).WithMeta("name", st.Name)
		return
	}

	if typ.Fields == nil || typ.Fields.List == nil || len(typ.Fields.List) == 0 {
		return
	}

	for _, field := range typ.Fields.List {
		// todo 组合struct的解析
		// todo 多个name，以一个name来append一个field，有这些name的fields
		if field.Names == nil || len(field.Names) == 0 {
			// todo 组合struct的解析 ???
			continue
		}
		if len(field.Names) > 1 {
			continue
		}
		name := field.Names[0].Name
		if !ast.IsExported(name) {
			continue
		}
		tags := make(map[string]string)
		if field.Tag != nil && field.Tag.Value != "" {
			tags = parseFieldTag(field.Tag.Value)
		}
		if jsonName, hasJsonName := tags["json"]; hasJsonName && jsonName == "-" {
			continue
		}
		sf := &StructField{
			Name:        name,
			Annotations: nil,
			Tags:        tags,
			Element:     nil,
		}
		if field.Doc != nil && field.Doc.Text() != "" {
			annotations, parseAnnotationsErr := ParseAnnotations(field.Doc.Text())
			if parseAnnotationsErr != nil {
				err = errors.Warning("forg: parse struct failed").WithCause(parseAnnotationsErr).
					WithMeta("path", st.Path).WithMeta("name", st.Name).
					WithMeta("field", name)
				return
			}
			sf.Annotations = annotations
		} else {
			sf.Annotations = Annotations{}
		}
		element, elementErr := newElement(field.Type, st.mod, st.Path, fileImports)
		if elementErr != nil {
			err = errors.Warning("forg: parse struct failed").WithCause(elementErr).
				WithMeta("path", st.Path).WithMeta("name", st.Name).
				WithMeta("field", name)
			return
		}
		sf.Element = element
		st.Fields = append(st.Fields, sf)
	}
	return
}

func (st *Struct) loadType(ctx context.Context) (typ *ast.StructType, fileImports Imports, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("forg: get struct type from source file failed").WithCause(ctx.Err())
		return
	}
	modPath, modDir, matched := st.mod.matchMod(st.Path)
	if !matched {
		err = errors.Warning("forg: get struct type from source file failed").
			WithCause(errors.Warning("not found")).
			WithMeta("path", st.Path).WithMeta("name", st.Name)
		return
	}
	subDir, _ := strings.CutPrefix(st.Path, modPath)
	dir := filepath.ToSlash(filepath.Join(modDir, subDir))
	if !files.ExistFile(dir) {
		err = errors.Warning("forg: get struct type from source file failed").
			WithCause(errors.Warning("dir not found")).WithMeta("dir", dir)
		return
	}
	entries, readDirErr := os.ReadDir(dir)
	if readDirErr != nil {
		err = errors.Warning("forg: get struct type from source file failed").
			WithCause(readDirErr).WithMeta("dir", dir)
		return
	}
	if entries == nil || len(entries) == 0 {
		err = errors.Warning("forg: get struct type from source file failed").
			WithCause(errors.Warning("no entry in dir")).WithMeta("dir", dir)
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
			continue
		}
		path := filepath.ToSlash(filepath.Join(dir, entry.Name()))
		sf, parseSourceErr := files.ParseSource(path)
		if parseSourceErr != nil {
			err = errors.Warning("forg: get struct type from source file failed").
				WithCause(parseSourceErr).WithMeta("path", path)
			return
		}
		if sf.Decls == nil || len(sf.Decls) == 0 {
			continue
		}
		for _, decl := range sf.Decls {
			gen, genOk := decl.(*ast.GenDecl)
			if !genOk {
				continue
			}
			if gen.Specs == nil || len(gen.Specs) == 0 {
				continue
			}
			for _, s := range gen.Specs {
				spec, typOk := s.(*ast.TypeSpec)
				if !typOk {
					continue
				}
				structType, structTypeOk := spec.Type.(*ast.StructType)
				if !structTypeOk {
					return
				}
				if spec.Name.Name == st.Name {
					doc := ""
					if spec.Doc != nil && len(spec.Doc.Text()) != 0 {
						doc = spec.Doc.Text()
					} else if len(gen.Specs) == 1 && gen.Doc != nil && len(gen.Doc.Text()) != 0 {
						doc = gen.Doc.Text()
					}
					if doc != "" {
						annotations, parseAnnotationsErr := ParseAnnotations(doc)
						if parseAnnotationsErr != nil {
							err = errors.Warning("forg: get struct type from source file failed").
								WithCause(parseAnnotationsErr).WithMeta("path", path)
							return
						}
						st.Annotations = annotations
					} else {
						st.Annotations = Annotations{}
					}
					typ = structType
					fileImports = newImportsFromAstFileImports(sf.Imports)
					return
				}
			}
		}
	}
	return
}
