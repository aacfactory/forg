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
	BuiltinKind = TypeKind(iota + 1)
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

func newType(expr ast.Expr, scope *TypeScope) (typ *Type, err error) {
	var kind TypeKind
	path := ""
	name := ""
	switch expr.(type) {
	case *ast.Ident:

		break
	case *ast.SelectorExpr:

		break
	case *ast.StarExpr:

		break
	case *ast.StructType:

		break
	case *ast.ArrayType:

		break
	case *ast.MapType:

		break
	default:
		err = errors.Warning("forg: new type from ast expr failed").WithCause(errors.Warning(fmt.Sprintf("%s was not supported", reflect.TypeOf(expr))))
		return
	}
	typ = &Type{
		expr:        expr,
		Kind:        kind,
		Path:        path,
		Name:        name,
		Annotations: Annotations{},
		Paradigms:   make([]*TypeParadigm, 0, 1),
		Tags:        make(map[string]string),
		Elements:    make([]*Type, 0, 1),
	}
	return
}

type Type struct {
	expr        ast.Expr
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
	if typ.Paradigms != nil && len(typ.Paradigms) > 0 {
		p := ""
		for _, paradigm := range typ.Paradigms {
			p = p + "," + paradigm.Name
			if paradigm.Types != nil && len(paradigm.Types) > 0 {
				pts := ""
				for _, pt := range paradigm.Types {
					pts = pts + "|" + pt.Key()
				}
				if pts != "" {
					p = p + pts[1:]
				}
			}
		}
		if p != "" {
			key = key + "[" + p[1:] + "]"
		}
	}
	return
}

func (typ *Type) GetPath() (path string) {
	if typ.Path != "" {
		path = typ.Path
		return
	}
	if typ.Kind == BuiltinKind {
		return
	}
	switch typ.Kind {
	case PointerKind, ArrayKind:
		path = typ.Elements[0].GetPath()
		break
	case MapKind:
		path = typ.Elements[1].GetPath()
		break
	}
	return
}

func (typ *Type) GetParadigmPaths() (paths []string) {
	paths = make([]string, 0, 1)
	if typ.Paradigms != nil && len(typ.Paradigms) > 0 {
		for _, paradigm := range typ.Paradigms {
			if paradigm.Types != nil && len(paradigm.Types) > 0 {
				for _, t := range paradigm.Types {
					path := t.GetPath()
					if path != "" {
						paths = append(paths, path)
					}
				}
			}
		}
		return
	}
	if typ.Kind == BuiltinKind {
		return
	}
	switch typ.Kind {
	case PointerKind, ArrayKind:
		paths = typ.Elements[0].GetParadigmPaths()
		break
	case MapKind:
		paths = typ.Elements[1].GetParadigmPaths()
		break
	}
	return
}

type TypeScope struct {
	Mod     *Module
	Path    string
	Imports Imports
}

type Types struct {
	values sync.Map
	group  singleflight.Group
	mod    *Module
}

func (types *Types) parse(ctx context.Context, expr ast.Expr, scope *TypeScope) (typ *Type, err error) {
	bt, btOk := types.tryParseBuiltinType(expr, scope)
	if btOk {
		typ = bt
		return
	}
	// scan basic
	typ, err = newType(expr, scope)
	if err != nil {
		err = errors.Warning("forg: parse type failed").WithCause(err)
		return
	}
	key := typ.Key()
	cached := ctx.Value(key)
	if cached != nil {
		typ = cached.(*Type)
		return
	}
	stored, has := types.values.Load(key)
	if has {
		typ = stored.(*Type)
		return
	}
	v, doErr, _ := types.group.Do(key, func() (v interface{}, err error) {
		ctx = context.WithValue(ctx, key, typ)
		err = types.scanType(ctx, typ, scope)
		if err != nil {
			return
		}
		types.values.Store(key, key)
		v = typ
		return
	})
	if doErr != nil {
		err = errors.Warning("forg: parse type failed").WithMeta("key", key).WithCause(doErr)
		return
	}
	typ = v.(*Type)
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
			Kind:        BuiltinKind,
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
		case "encoding/json":
			switch structName {
			case "RawMessage":
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
		case "github.com/aacfactory/json":
			switch structName {
			case "RawMessage", "Object", "Array", "Date", "Time":
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
		case "github.com/aacfactory/fns-contrib/databases/sql/dal":
			switch structName {
			case "PageResult":
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

func (types *Types) scanType(ctx context.Context, typ *Type, scope *TypeScope) (err error) {
	switch typ.expr.(type) {
	case *ast.Ident:

		break
	case *ast.SelectorExpr:

		break
	case *ast.StarExpr:

		break
	case *ast.ArrayType:

		break
	case *ast.MapType:

		break
	default:
		err = errors.Warning("forg: unsupported expr").WithMeta("expr", reflect.TypeOf(typ.expr).String()).WithMeta("path", typ.Path).WithMeta("name", typ.Name)
		return
	}
	return
}

func isContextType(expr ast.Expr, scope *TypeScope) (ok bool) {
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
	importer, has := scope.Imports.Find(pkg)
	if !has {
		return
	}
	ok = importer.Path == "context"
	return
}

func isCodeErrorType(expr ast.Expr, scope *TypeScope) (ok bool) {
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
	importer, has := scope.Imports.Find(pkg)
	if !has {
		return
	}
	ok = importer.Path == "github.com/aacfactory/errors"
	return
}
