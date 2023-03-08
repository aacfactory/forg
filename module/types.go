package module

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"go/ast"
	"golang.org/x/sync/singleflight"
	"reflect"
	"sync"
)

const (
	BasicKind   = TypeKind(iota + 1) // 基本类型，oas时不需要ref
	BuiltinKind                      // 内置类型，oas时需要ref，但不需要建component
	StructKind
	StructFieldKind
	PointerKind
	ArrayKind
	MapKind
	AnyKind
	ParadigmKind
)

type TypeKind int

type TypeParadigm struct {
	Name  string
	Types []*Type
}

var AnyType = &Type{
	Kind:        AnyKind,
	Path:        "",
	Name:        "",
	Annotations: nil,
	Paradigms:   nil,
	Tags:        nil,
	Elements:    nil,
}

type Type struct {
	Kind        TypeKind
	Path        string
	Name        string
	Annotations Annotations
	Paradigms   []*TypeParadigm
	Tags        map[string]string
	Elements    []*Type
}

func (typ *Type) Key() (key string) {
	key = fmt.Sprintf("%s:%s", typ.Path, typ.Name)
	return
}

func (typ *Type) GetTopPaths() (paths []string) {
	paths = make([]string, 0, 1)
	switch typ.Kind {
	case StructKind:
		paths = append(paths, typ.Path)
	case PointerKind, ArrayKind:
		paths = append(paths, typ.Elements[0].GetTopPaths()...)
		break
	case MapKind:
		paths = append(paths, typ.Elements[0].GetTopPaths()...)
		paths = append(paths, typ.Elements[1].GetTopPaths()...)
		break
	}
	return
}

type TypeScope struct {
	Path       string
	Mod        *Module
	Imports    Imports
	GenericDoc string
}

type Types struct {
	values sync.Map
	group  singleflight.Group
}

func (types *Types) parseType(ctx context.Context, spec *ast.TypeSpec, scope *TypeScope) (typ *Type, err error) {
	//
	switch spec.Type.(type) {
	case *ast.Ident:

		break
	case *ast.StructType:
		typ, err = types.parseStructType(ctx, spec, scope)
		break
	case *ast.ArrayType:

	case *ast.MapType:

	default:

	}
	return
}

func (types *Types) parseTypeParadigms(ctx context.Context, params *ast.FieldList, scope *TypeScope) (paradigms []*TypeParadigm, err error) {

	return
}

func (types *Types) parseParadigmType(ctx context.Context, param ast.Expr, scope *TypeScope) (paradigm *TypeParadigm, err error) {

	return
}

func (types *Types) parseStructType(ctx context.Context, spec *ast.TypeSpec, scope *TypeScope) (typ *Type, err error) {
	path := scope.Path
	name := spec.Name.Name
	st, typeOk := spec.Type.(*ast.StructType)
	if !typeOk {
		err = errors.Warning("forg: parse struct type failed").
			WithMeta("path", path).WithMeta("name", name).
			WithCause(errors.Warning("type of spec is not ast.StructType").WithMeta("type", reflect.TypeOf(spec.Type).String()))
		return
	}
	typ = &Type{
		Kind:        StructKind,
		Path:        path,
		Name:        name,
		Annotations: nil,
		Paradigms:   nil,
		Tags:        nil,
		Elements:    make([]*Type, 0, 1),
	}
	key := typ.Key()
	cached := ctx.Value(key)
	if cached != nil {
		typ = cached.(*Type)
		return
	}
	stored, loaded := types.values.Load(key)
	if loaded {
		typ = stored.(*Type)
		return
	}
	result, doErr, _ := types.group.Do(key, func() (v interface{}, err error) {
		ctx = context.WithValue(ctx, key, typ)
		// annotations
		doc := ""
		if spec.Doc != nil && spec.Doc.Text() != "" {
			doc = spec.Doc.Text()
		} else {
			doc = scope.GenericDoc
		}
		annotations, parseAnnotationsErr := ParseAnnotations(doc)
		if parseAnnotationsErr != nil {
			err = errors.Warning("forg: parse struct type failed").
				WithMeta("path", path).WithMeta("name", name).
				WithCause(parseAnnotationsErr)
			return
		}
		typ.Annotations = annotations
		// paradigms
		if spec.TypeParams != nil && spec.TypeParams.NumFields() > 0 {
			paradigms, parseParadigmsErr := types.parseTypeParadigms(ctx, spec.TypeParams, scope)
			if parseParadigmsErr != nil {
				err = errors.Warning("forg: parse struct type failed").
					WithMeta("path", path).WithMeta("name", name).
					WithCause(parseParadigmsErr)
				return
			}
			typ.Paradigms = paradigms
		}
		// fields
		// get name, annotations, tag from field
		// get element and paradigms from field.Type (*ast.IndexExpr or *ast.IndexListExpr contains paradigms)
		if st.Fields != nil && st.Fields.NumFields() > 0 {
			typ.Elements = make([]*Type, 0, 1)
			for i, field := range st.Fields.List {
				if field.Names != nil && len(field.Names) > 1 {
					err = errors.Warning("forg: parse struct type failed").
						WithMeta("path", path).WithMeta("name", name).
						WithCause(errors.Warning("forg: too many names of one field")).WithMeta("field_no", fmt.Sprintf("%d", i))
					return
				}
				if field.Names == nil || len(field.Names) == 0 {
					// compose
					if field.Type != nil {
						element, parseStructFieldTypeErr := types.parseStructFieldType(ctx, typ, field.Type, scope)
						if parseStructFieldTypeErr != nil {
							err = errors.Warning("forg: parse struct type failed").
								WithMeta("path", path).WithMeta("name", name).
								WithCause(parseStructFieldTypeErr).WithMeta("field_no", fmt.Sprintf("%d", i))
							return
						}
						typ.Elements = append(typ.Elements, &Type{
							Kind:        StructFieldKind,
							Path:        "",
							Name:        "",
							Annotations: nil,
							Paradigms:   nil,
							Tags:        nil,
							Elements:    []*Type{element},
						})
					} else {
						err = errors.Warning("forg: parse struct type failed").
							WithMeta("path", path).WithMeta("name", name).
							WithCause(errors.Warning("forg: unsupported field")).WithMeta("field_no", fmt.Sprintf("%d", i))
						return
					}
					return
				}
				ft := &Type{
					Kind:        StructFieldKind,
					Path:        "",
					Name:        "",
					Annotations: nil,
					Paradigms:   nil,
					Tags:        nil,
					Elements:    nil,
				}
				// name
				ft.Name = field.Names[0].Name
				// tag
				if field.Tag != nil && field.Tag.Value != "" {
					ft.Tags = parseFieldTag(field.Tag.Value)
				}
				// annotations
				if field.Doc != nil && field.Doc.Text() != "" {
					fieldAnnotations, parseFieldAnnotationsErr := ParseAnnotations(field.Doc.Text())
					if parseFieldAnnotationsErr != nil {
						err = errors.Warning("forg: parse struct type failed").
							WithMeta("path", path).WithMeta("name", name).
							WithCause(parseFieldAnnotationsErr).
							WithMeta("field_no", fmt.Sprintf("%d", i)).
							WithMeta("field", ft.Name)
						return
					}
					ft.Annotations = fieldAnnotations
				}
				// paradigms and element
				element, parseStructFieldTypeErr := types.parseStructFieldType(ctx, typ, field.Type, scope)
				if parseStructFieldTypeErr != nil {
					err = errors.Warning("forg: parse struct type failed").
						WithMeta("path", path).WithMeta("name", name).
						WithCause(parseStructFieldTypeErr).
						WithMeta("field_no", fmt.Sprintf("%d", i)).
						WithMeta("field", ft.Name)
					return
				}
				ft.Elements = []*Type{element}
				typ.Elements = append(typ.Elements, ft)
			}
		}
		return
	})
	if doErr != nil {
		err = doErr
		return
	}
	typ = result.(*Type)
	return
}

func (types *Types) parseStructFieldType(ctx context.Context, st *Type, field ast.Expr, scope *TypeScope) (element *Type, err error) {
	if ctx.Err() != nil {
		err = ctx.Err()
		return
	}
	switch field.(type) {
	case *ast.Ident:
		expr := field.(*ast.Ident)
		isBuiltin := expr.Name == "string" ||
			expr.Name == "bool" ||
			expr.Name == "int" || expr.Name == "int8" || expr.Name == "int16" || expr.Name == "int32" || expr.Name == "int64" ||
			expr.Name == "uint" || expr.Name == "uint8" || expr.Name == "uint16" || expr.Name == "uint32" || expr.Name == "uint64" ||
			expr.Name == "float32" || expr.Name == "float64" ||
			expr.Name == "complex64" || expr.Name == "complex128"
		if isBuiltin {
			element = &Type{
				Kind:        BasicKind,
				Path:        "",
				Name:        expr.Name,
				Annotations: Annotations{},
				Paradigms:   make([]*TypeParadigm, 0, 1),
				Elements:    make([]*Type, 0, 1),
			}
			break
		}
		// paradigms
		if st.Paradigms != nil && len(st.Paradigms) > 0 {
			for _, paradigm := range st.Paradigms {
				if paradigm.Name == expr.Name {
					element = &Type{
						Kind:        ParadigmKind,
						Path:        "",
						Name:        expr.Name,
						Annotations: nil,
						Paradigms:   nil,
						Tags:        nil,
						Elements:    paradigm.Types,
					}
					break
				}
			}
		}
		if expr.Obj == nil {
			// not in same file
			element, err = scope.Mod.ParseType(ctx, scope.Path, expr.Name)
			break
		}
		// in same file
		if expr.Obj.Kind != ast.Typ || expr.Obj.Decl == nil {
			err = errors.Warning("forg: kind of field object must be type")
			break
		}
		spec, isTypeSpec := expr.Obj.Decl.(*ast.TypeSpec)
		if !isTypeSpec {
			err = errors.Warning("forg: kind of field object must be type").WithMeta("decl", reflect.TypeOf(expr.Obj.Decl).String())
			break
		}
		element, err = types.parseType(ctx, spec, scope)
		break
	case *ast.SelectorExpr:
		expr := field.(*ast.SelectorExpr)
		ident, isIdent := expr.X.(*ast.Ident)
		if !isIdent {
			err = errors.Warning("forg: x type of selector field must be ident").WithMeta("selector_x", reflect.TypeOf(expr.X).String())
			break
		}
		// path
		importer, hasImporter := scope.Imports.Find(ident.Name)
		if !hasImporter {
			err = errors.Warning("forg: import of selector field was not found").WithMeta("import", ident.Name)
			break
		}
		// name
		name := expr.Sel.Name
		builtin, isBuiltin := tryGetBuiltinType(importer.Path, name)
		if isBuiltin {
			element = builtin
			break
		}
		// find in mod
		element, err = scope.Mod.ParseType(ctx, importer.Path, expr.Sel.Name)
		break
	case *ast.StarExpr:
		expr := field.(*ast.StarExpr)
		starElement, parseStarErr := types.parseStructFieldType(ctx, st, expr.X, scope)
		if parseStarErr != nil {
			err = parseStarErr
			break
		}
		element = &Type{
			Kind:        PointerKind,
			Path:        "",
			Name:        "",
			Annotations: nil,
			Paradigms:   nil,
			Tags:        nil,
			Elements:    []*Type{starElement},
		}
		break
	case *ast.ArrayType:
		expr := field.(*ast.ArrayType)
		arrayElement, parseArrayErr := types.parseStructFieldType(ctx, st, expr.Elt, scope)
		if parseArrayErr != nil {
			err = parseArrayErr
			break
		}
		element = &Type{
			Kind:        ArrayKind,
			Path:        "",
			Name:        "",
			Annotations: nil,
			Paradigms:   nil,
			Tags:        nil,
			Elements:    []*Type{arrayElement},
		}
		break
	case *ast.MapType:
		expr := field.(*ast.MapType)
		keyElement, parseKeyErr := types.parseStructFieldType(ctx, st, expr.Key, scope)
		if parseKeyErr != nil {
			err = parseKeyErr
			break
		}
		if keyElement.Kind != BasicKind {
			err = errors.Warning("forg: key kind of map kind field must be basic")
			break
		}
		valueElement, parseValueErr := types.parseStructFieldType(ctx, st, expr.Value, scope)
		if parseValueErr != nil {
			err = parseValueErr
			break
		}
		element = &Type{
			Kind:        MapKind,
			Path:        "",
			Name:        "",
			Annotations: nil,
			Paradigms:   nil,
			Tags:        nil,
			Elements:    []*Type{keyElement, valueElement},
		}
		break
	case *ast.IndexExpr:
		expr := field.(*ast.IndexExpr)
		paradigmType, parseParadigmTypeErr := types.parseStructFieldType(ctx, st, expr.Index, scope)
		if parseParadigmTypeErr != nil {
			err = parseParadigmTypeErr
			break
		}
		element, err = types.parseStructFieldType(ctx, st, expr.X, scope)
		if err != nil {
			break
		}
		element.Paradigms = []*TypeParadigm{{
			Name:  "",
			Types: []*Type{paradigmType},
		}}
		break
	case *ast.IndexListExpr:
		expr := field.(*ast.IndexListExpr)
		paradigmTypes := make([]*Type, 0, 1)
		for _, index := range expr.Indices {
			paradigmType, parseParadigmTypeErr := types.parseStructFieldType(ctx, st, index, scope)
			if parseParadigmTypeErr != nil {
				err = parseParadigmTypeErr
				break
			}
			paradigmTypes = append(paradigmTypes, paradigmType)
		}
		element, err = types.parseStructFieldType(ctx, st, expr.X, scope)
		if err != nil {
			break
		}
		paradigms := make([]*TypeParadigm, 0, 1)
		for _, paradigmType := range paradigmTypes {
			paradigms = append(paradigms, &TypeParadigm{
				Name:  "",
				Types: []*Type{paradigmType},
			})
		}
		break
	default:
		err = errors.Warning("forg: unsupported field type").WithMeta("type", reflect.TypeOf(field).String())
		return
	}
	return
}

func isContextType(expr ast.Expr, imports Imports) (ok bool) {
	e, isSelector := expr.(*ast.SelectorExpr)
	if !isSelector {
		return
	}
	if e.X == nil {
		return
	}
	ident, isIdent := e.X.(*ast.Ident)
	if !isIdent {
		return
	}
	pkg := ident.Name
	if pkg == "" {
		return
	}
	if e.Sel == nil {
		return
	}
	ok = e.Sel.Name == "Context"
	if !ok {
		return
	}
	importer, has := imports.Find(pkg)
	if !has {
		return
	}
	ok = importer.Path == "context"
	return
}

func isCodeErrorType(expr ast.Expr, imports Imports) (ok bool) {
	e, isSelector := expr.(*ast.SelectorExpr)
	if !isSelector {
		return
	}
	if e.X == nil {
		return
	}
	ident, isIdent := e.X.(*ast.Ident)
	if !isIdent {
		return
	}
	pkg := ident.Name
	if pkg == "" {
		return
	}
	if e.Sel == nil {
		return
	}
	ok = e.Sel.Name == "CodeError"
	if !ok {
		return
	}
	importer, has := imports.Find(pkg)
	if !has {
		return
	}
	ok = importer.Path == "github.com/aacfactory/errors"
	return
}
