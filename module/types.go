package module

import (
	"context"
	"fmt"
	"go/ast"
	"golang.org/x/sync/singleflight"
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

func (typ *Type) GetTopPaths() (paths []string) {
	switch typ.Kind {
	case StructKind:
		paths = make([]string, 0, 1)
		if typ.Path != "" {
			paths = append(paths, typ.Path)
		}
		if typ.Paradigms != nil && len(typ.Paradigms) > 0 {
			for _, paradigm := range typ.Paradigms {
				if paradigm.Types != nil && len(paradigm.Types) > 0 {
					for _, pt := range paradigm.Types {
						paradigmPaths := pt.GetTopPaths()
						if paradigmPaths != nil {
							paths = append(paths, paradigmPaths...)
						}
					}
				}
			}
		}
	case PointerKind, ArrayKind:
		paths = typ.Elements[0].GetTopPaths()
		break
	case MapKind:
		paths = typ.Elements[1].GetTopPaths()
		break
	}
	return
}

func NewTypeScope(path string, imports Imports) (scope *TypeScope) {

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

func (types *Types) parseStructType(ctx context.Context, name string, st *ast.StructType, scope *TypeScope) (typ *Type, err error) {

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
