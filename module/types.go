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
	PointerKind
	ArrayKind
	MapKind
)

type TypeKind int

type TypeParadigm struct {
	Name  string
	Types []Type
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

func (typ *Type) GetTopPath() (path string) {
	switch typ.Kind {
	case StructKind:
		path = typ.Path
	case PointerKind, ArrayKind:
		path = typ.Elements[0].GetTopPath()
		break
	case MapKind:
		path = typ.Elements[1].GetTopPath()
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

	return
}

func (types *Types) parseTypeParadigms(ctx context.Context, params *ast.FieldList, scope *TypeScope) (paradigms []*TypeParadigm, err error) {

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
		// annotations

		// fields

		return
	})
	if doErr != nil {
		err = doErr
		return
	}
	typ = result.(*Type)
	return
}

func (types *Types) tryParseBuiltinType(expr ast.Expr, scope *TypeScope) (typ *Type, ok bool) {
	switch expr.(type) {
	case *ast.Ident:
		e := expr.(*ast.Ident)
		if e.Obj != nil {
			return
		}
		isBuiltin := e.Name == "string" ||
			e.Name == "bool" ||
			e.Name == "int" || e.Name == "int8" || e.Name == "int16" || e.Name == "int32" || e.Name == "int64" ||
			e.Name == "uint" || e.Name == "uint8" || e.Name == "uint16" || e.Name == "uint32" || e.Name == "uint64" ||
			e.Name == "float32" || e.Name == "float64" ||
			e.Name == "complex64" || e.Name == "complex128"
		if !isBuiltin {
			break
		}
		typ = &Type{
			Kind:        BasicKind,
			Path:        "",
			Name:        e.Name,
			Annotations: Annotations{},
			Paradigms:   make([]*TypeParadigm, 0, 1),
			Elements:    make([]*Type, 0, 1),
		}
		break
	case *ast.SelectorExpr:
		e := expr.(*ast.SelectorExpr)
		pkg := ""
		if e.X != nil {
			pkgExpr, isIdent := e.X.(*ast.Ident)
			if isIdent {
				pkg = pkgExpr.Name
			}
		}
		path := ""
		if pkg != "" && scope != nil {
			importer, has := scope.Imports.Find(pkg)
			if has {
				path = importer.Path
			}
		}
		structName := ""
		if e.Sel != nil {
			structName = e.Sel.Name
		}
		if path == "" || structName == "" {
			break
		}
		switch path {
		case "time":
			switch structName {
			case "Time", "Duration":
				typ = &Type{
					Kind:        BasicKind,
					Path:        path,
					Name:        structName,
					Annotations: Annotations{},
					Paradigms:   make([]*TypeParadigm, 0, 1),
					Elements:    make([]*Type, 0, 1),
				}
				break
			default:
				break
			}
			break
		case "encoding/json":
			switch structName {
			case "RawMessage":
				typ = &Type{
					Kind:        BasicKind,
					Path:        path,
					Name:        structName,
					Annotations: Annotations{},
					Paradigms:   make([]*TypeParadigm, 0, 1),
					Elements:    make([]*Type, 0, 1),
				}
				break
			default:
				break
			}
			break
		case "github.com/aacfactory/json":
			switch structName {
			case "RawMessage", "Date", "Time":
				typ = &Type{
					Kind:        BasicKind,
					Path:        path,
					Name:        structName,
					Annotations: Annotations{},
					Paradigms:   make([]*TypeParadigm, 0, 1),
					Elements:    make([]*Type, 0, 1),
				}
				break
			case "Object", "Array":
				typ = &Type{
					Kind:        BuiltinKind,
					Path:        path,
					Name:        structName,
					Annotations: Annotations{},
					Paradigms:   make([]*TypeParadigm, 0, 1),
					Elements:    make([]*Type, 0, 1),
				}
				break
			default:
				break
			}
			break
		case "github.com/aacfactory/errors":
			switch structName {
			case "CodeError":
				typ = &Type{
					Kind:        BuiltinKind,
					Path:        path,
					Name:        structName,
					Annotations: Annotations{},
					Paradigms:   make([]*TypeParadigm, 0, 1),
					Elements:    make([]*Type, 0, 1),
				}
				break
			default:
				break
			}
			break
		case "github.com/aacfactory/service":
			switch structName {
			case "Empty":
				typ = &Type{
					Kind:        BuiltinKind,
					Path:        path,
					Name:        structName,
					Annotations: Annotations{},
					Paradigms:   make([]*TypeParadigm, 0, 1),
					Elements:    make([]*Type, 0, 1),
				}
				break
			default:
				break
			}
			break
		case "github.com/aacfactory/fns-contrib/databases/sql":
			switch structName {
			case "Date", "Time":
				typ = &Type{
					Kind:        BasicKind,
					Path:        path,
					Name:        structName,
					Annotations: Annotations{},
					Paradigms:   make([]*TypeParadigm, 0, 1),
					Elements:    make([]*Type, 0, 1),
				}
				break
			default:
				break
			}
			break
		case "github.com/aacfactory/fns-contrib/databases/sql/dal":
			switch structName {
			case "Pager":
				typ = &Type{
					Kind:        BuiltinKind,
					Path:        path,
					Name:        structName,
					Annotations: Annotations{},
					Paradigms:   make([]*TypeParadigm, 0, 1),
					Elements:    make([]*Type, 0, 1),
				}
				// todo 泛型
				break
			default:
				break
			}
			break
		default:
			break
		}
		break
	default:
		break
	}
	ok = typ != nil
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
