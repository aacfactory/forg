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
	IdentKind
	InterfaceKind
	StructKind
	StructFieldKind
	PointerKind
	ArrayKind
	MapKind
	AnyKind
	ParadigmKind
	referenceKind
)

type TypeKind int

type TypeParadigm struct {
	Name  string
	Types []*Type
}

func (tp *TypeParadigm) String() (v string) {
	types := ""
	if tp.Types != nil && len(tp.Types) > 0 {
		for _, typ := range tp.Types {
			types = types + "| " + typ.String()
		}
		if types != "" {
			types = types[2:]
		}
	}
	v = fmt.Sprintf("[%s %s]", tp.Name, types)
	return
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

func (typ *Type) String() (v string) {
	switch typ.Kind {
	case BasicKind:
		v = typ.Name
		break
	case BuiltinKind, IdentKind, InterfaceKind, StructKind, StructFieldKind, PointerKind:
		v = typ.Key()
		break
	case ArrayKind:
		v = fmt.Sprintf("[]%s", typ.Elements[0].Key())
		break
	case MapKind:
		v = fmt.Sprintf("map[%s]%s", typ.Elements[0].String(), typ.Elements[1].String())
		break
	case AnyKind:
		v = "any"
		break
	case ParadigmKind:
		elements := ""
		for _, element := range typ.Elements {
			elements = elements + "| " + element.String()
		}
		if elements != "" {
			elements = elements[2:]
		}
		v = fmt.Sprintf("[%s %s]", typ.Name, elements)
		break
	}
	return
}

func (typ *Type) Key() (key string) {
	key = formatTypeKey(typ.Path, typ.Name)
	return
}

func formatTypeKey(path string, name string) (key string) {
	key = fmt.Sprintf("%s:%s", path, name)
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

func (typ *Type) Basic() (name string, ok bool) {
	if typ.Kind == BasicKind {
		name = typ.Name
		ok = true
		return
	}
	if typ.Kind == IdentKind {
		name, ok = typ.Elements[0].Basic()
		return
	}
	return
}

func (typ *Type) warpReference(types *Types) {
	if typ.Elements != nil && len(typ.Elements) > 0 {
		for i, element := range typ.Elements {
			if element.Kind == referenceKind {
				ref, has := types.values.Load(typ.Key())
				if has {
					typ.Elements[i] = ref.(*Type)
					element = ref.(*Type)
				}
			}
			element.warpReference(types)
		}
	}
	if typ.Paradigms != nil && len(typ.Paradigms) > 0 {
		for _, paradigm := range typ.Paradigms {
			if paradigm.Types != nil && len(paradigm.Types) > 0 {
				for i, pt := range paradigm.Types {
					if pt.Kind == referenceKind {
						ref, has := types.values.Load(typ.Key())
						if has {
							typ.Elements[i] = ref.(*Type)
							pt = ref.(*Type)
						}
					}
					pt.warpReference(types)
				}
			}
		}
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
	path := scope.Path
	name := spec.Name.Name

	key := formatTypeKey(path, name)

	processing := ctx.Value(key)
	if processing != nil {
		typ = &Type{
			Kind:        referenceKind,
			Path:        path,
			Name:        name,
			Annotations: nil,
			Paradigms:   nil,
			Tags:        nil,
			Elements:    nil,
		}
		return
	}

	result, doErr, _ := types.group.Do(key, func() (v interface{}, err error) {
		stored, loaded := types.values.Load(key)
		if loaded {
			v = stored.(*Type)
			return
		}
		ctx = context.WithValue(ctx, key, "processing")
		var result *Type
		switch spec.Type.(type) {
		case *ast.Ident:
			identType, parseIdentTypeErr := types.parseExpr(ctx, spec.Type, scope)
			if parseIdentTypeErr != nil {
				err = errors.Warning("forg: parse ident type spec failed").
					WithMeta("path", path).WithMeta("name", name).
					WithCause(parseIdentTypeErr)
				break
			}
			result = &Type{
				Kind:        IdentKind,
				Path:        path,
				Name:        name,
				Annotations: nil,
				Paradigms:   nil,
				Tags:        nil,
				Elements:    []*Type{identType},
			}
			break
		case *ast.StructType:
			result, err = types.parseStructType(ctx, spec, scope)
			break
		case *ast.InterfaceType:
			result = &Type{
				Kind:        InterfaceKind,
				Path:        path,
				Name:        name,
				Annotations: nil,
				Paradigms:   nil,
				Tags:        nil,
				Elements:    nil,
			}
			break
		case *ast.ArrayType:
			arrayType := spec.Type.(*ast.ArrayType)
			arrayElementType, parseArrayElementTypeErr := types.parseExpr(ctx, arrayType, scope)
			if parseArrayElementTypeErr != nil {
				err = errors.Warning("forg: parse array type spec failed").
					WithMeta("path", path).WithMeta("name", name).
					WithCause(parseArrayElementTypeErr)
				break
			}
			result = &Type{
				Kind:        ArrayKind,
				Path:        path,
				Name:        name,
				Annotations: nil,
				Paradigms:   nil,
				Tags:        nil,
				Elements:    []*Type{arrayElementType},
			}
			break
		case *ast.MapType:
			mapType := spec.Type.(*ast.MapType)
			keyElement, parseKeyErr := types.parseExpr(ctx, mapType.Key, scope)
			if parseKeyErr != nil {
				err = errors.Warning("forg: parse array type spec failed").
					WithMeta("path", path).WithMeta("name", name).
					WithCause(parseKeyErr)
				break
			}
			if _, basic := keyElement.Basic(); !basic {
				err = errors.Warning("forg: parse array type spec failed").
					WithMeta("path", path).WithMeta("name", name).
					WithCause(errors.Warning("forg: key kind of map kind field must be basic"))
				break
			}
			valueElement, parseValueErr := types.parseExpr(ctx, mapType.Value, scope)
			if parseValueErr != nil {
				err = errors.Warning("forg: parse array type spec failed").
					WithMeta("path", path).WithMeta("name", name).
					WithCause(parseValueErr)
				break
			}
			result = &Type{
				Kind:        MapKind,
				Path:        path,
				Name:        name,
				Annotations: nil,
				Paradigms:   nil,
				Tags:        nil,
				Elements:    []*Type{keyElement, valueElement},
			}
			break
		default:
			err = errors.Warning("forg: unsupported type spec").WithMeta("path", path).WithMeta("name", name)
			break
		}
		if err != nil {
			return
		}
		types.values.Store(key, result)
		// warp referenceKind
		result.warpReference(types)
		v = result
		return
	})
	if doErr != nil {
		err = doErr
		return
	}
	typ = result.(*Type)
	return
}

func (types *Types) parseTypeParadigms(ctx context.Context, params *ast.FieldList, scope *TypeScope) (paradigms []*TypeParadigm, err error) {
	paradigms = make([]*TypeParadigm, 0, 1)
	for _, param := range params.List {
		paradigm, paradigmErr := types.parseTypeParadigm(ctx, param, scope)
		if paradigmErr != nil {
			err = paradigmErr
			return
		}
		paradigms = append(paradigms, paradigm)
	}
	return
}

func (types *Types) parseTypeParadigm(ctx context.Context, param *ast.Field, scope *TypeScope) (paradigm *TypeParadigm, err error) {
	if param.Names != nil && len(param.Names) > 1 {
		err = errors.Warning("forg: parse paradigm failed").WithCause(errors.Warning("too many names"))
		return
	}
	name := ""
	if param.Names != nil {
		name = param.Names[0].Name
	}
	paradigm = &TypeParadigm{
		Name:  name,
		Types: make([]*Type, 0, 1),
	}
	if param.Type == nil {
		return
	}

	switch param.Type.(type) {
	case *ast.BinaryExpr:
		exprs := types.parseTypeParadigmBinaryExpr(param.Type.(*ast.BinaryExpr))
		for _, expr := range exprs {
			typ, parseTypeErr := types.parseExpr(ctx, expr, scope)
			if parseTypeErr != nil {
				err = errors.Warning("forg: parse paradigm failed").WithMeta("name", name).WithCause(parseTypeErr)
				return
			}
			paradigm.Types = append(paradigm.Types, typ)
		}
		break
	default:
		typ, parseTypeErr := types.parseExpr(ctx, param.Type, scope)
		if parseTypeErr != nil {
			err = errors.Warning("forg: parse paradigm failed").WithMeta("name", name).WithCause(parseTypeErr)
			return
		}
		paradigm.Types = append(paradigm.Types, typ)
		break
	}
	return
}

func (types *Types) parseTypeParadigmBinaryExpr(bin *ast.BinaryExpr) (exprs []ast.Expr) {
	exprs = make([]ast.Expr, 0, 1)
	xBin, isXBin := bin.X.(*ast.BinaryExpr)
	if isXBin {
		exprs = append(exprs, types.parseTypeParadigmBinaryExpr(xBin)...)
	} else {
		exprs = append(exprs, bin.X)
	}
	yBin, isYBin := bin.Y.(*ast.BinaryExpr)
	if isYBin {
		exprs = append(exprs, types.parseTypeParadigmBinaryExpr(yBin)...)
	} else {
		exprs = append(exprs, bin.Y)
	}
	return
}

func (types *Types) parseExpr(ctx context.Context, x ast.Expr, scope *TypeScope) (typ *Type, err error) {
	switch x.(type) {
	case *ast.Ident:
		expr := x.(*ast.Ident)
		if expr.Obj == nil {
			if expr.Name == "any" {
				typ = AnyType
				break
			}
			isBasic := expr.Name == "string" ||
				expr.Name == "bool" ||
				expr.Name == "int" || expr.Name == "int8" || expr.Name == "int16" || expr.Name == "int32" || expr.Name == "int64" ||
				expr.Name == "uint" || expr.Name == "uint8" || expr.Name == "uint16" || expr.Name == "uint32" || expr.Name == "uint64" ||
				expr.Name == "float32" || expr.Name == "float64" ||
				expr.Name == "complex64" || expr.Name == "complex128"
			if isBasic {
				typ = &Type{
					Kind:        BasicKind,
					Path:        "",
					Name:        expr.Name,
					Annotations: Annotations{},
					Paradigms:   make([]*TypeParadigm, 0, 1),
					Elements:    make([]*Type, 0, 1),
				}
				break
			} else {
				err = errors.Warning("forg: unsupported ident expr").WithMeta("ident", expr.Name)
				break
			}
		}
		if expr.Obj.Kind != ast.Typ || expr.Obj.Decl == nil {
			err = errors.Warning("forg: kind of ident expr object must be type and decl must not be nil")
			break
		}
		switch expr.Obj.Decl.(type) {
		case *ast.Field:
			// paradigms
			field := expr.Obj.Decl.(*ast.Field)
			paradigm, parseParadigmsErr := types.parseTypeParadigm(ctx, field, scope)
			if parseParadigmsErr != nil {
				err = parseParadigmsErr
				break
			}
			typ = &Type{
				Kind:        ParadigmKind,
				Path:        "",
				Name:        paradigm.Name,
				Annotations: nil,
				Paradigms:   nil,
				Tags:        nil,
				Elements:    paradigm.Types,
			}
			break
		case *ast.TypeSpec:
			// type
			spec := expr.Obj.Decl.(*ast.TypeSpec)
			typ, err = scope.Mod.ParseType(ctx, scope.Path, spec.Name.Name)
			break
		default:
			err = errors.Warning("forg: unsupported ident expr object decl").WithMeta("decl", reflect.TypeOf(expr.Obj.Decl).String())
			break
		}
		break
	case *ast.InterfaceType:
		typ = AnyType
		break
	case *ast.SelectorExpr:
		expr := x.(*ast.SelectorExpr)
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
			typ = builtin
			break
		}
		// find in mod
		typ, err = scope.Mod.ParseType(ctx, importer.Path, expr.Sel.Name)
		break
	case *ast.StarExpr:
		expr := x.(*ast.StarExpr)
		starElement, parseStarErr := types.parseExpr(ctx, expr.X, scope)
		if parseStarErr != nil {
			err = parseStarErr
			break
		}
		typ = &Type{
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
		expr := x.(*ast.ArrayType)
		arrayElement, parseArrayErr := types.parseExpr(ctx, expr.Elt, scope)
		if parseArrayErr != nil {
			err = parseArrayErr
			break
		}
		typ = &Type{
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
		expr := x.(*ast.MapType)
		keyElement, parseKeyErr := types.parseExpr(ctx, expr.Key, scope)
		if parseKeyErr != nil {
			err = parseKeyErr
			break
		}
		if _, basic := keyElement.Basic(); !basic {
			err = errors.Warning("forg: key kind of map kind field must be basic")
			break
		}
		valueElement, parseValueErr := types.parseExpr(ctx, expr.Value, scope)
		if parseValueErr != nil {
			err = parseValueErr
			break
		}
		typ = &Type{
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
		expr := x.(*ast.IndexExpr)
		paradigmType, parseParadigmTypeErr := types.parseExpr(ctx, expr.Index, scope)
		if parseParadigmTypeErr != nil {
			err = parseParadigmTypeErr
			break
		}
		typ, err = types.parseExpr(ctx, expr.X, scope)
		if err != nil {
			break
		}
		typ.Paradigms = []*TypeParadigm{{
			Name:  "",
			Types: []*Type{paradigmType},
		}}
		break
	case *ast.IndexListExpr:
		expr := x.(*ast.IndexListExpr)
		paradigmTypes := make([]*Type, 0, 1)
		for _, index := range expr.Indices {
			paradigmType, parseParadigmTypeErr := types.parseExpr(ctx, index, scope)
			if parseParadigmTypeErr != nil {
				err = parseParadigmTypeErr
				break
			}
			paradigmTypes = append(paradigmTypes, paradigmType)
		}
		typ, err = types.parseExpr(ctx, expr.X, scope)
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
		typ.Paradigms = paradigms
		break
	default:
		err = errors.Warning("forg: unsupported field type").WithMeta("type", reflect.TypeOf(x).String())
		return
	}
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
		Elements:    nil,
	}
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
	// elements
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
					fieldElementType, parseFieldElementTypeErr := types.parseExpr(ctx, field.Type, scope)
					if parseFieldElementTypeErr != nil {
						err = errors.Warning("forg: parse struct type failed").
							WithMeta("path", path).WithMeta("name", name).
							WithCause(parseFieldElementTypeErr).WithMeta("field_no", fmt.Sprintf("%d", i))
						return
					}
					typ.Elements = append(typ.Elements, &Type{
						Kind:        StructFieldKind,
						Path:        "",
						Name:        "",
						Annotations: nil,
						Paradigms:   nil,
						Tags:        nil,
						Elements:    []*Type{fieldElementType},
					})
				} else {
					err = errors.Warning("forg: parse struct type failed").
						WithMeta("path", path).WithMeta("name", name).
						WithCause(errors.Warning("forg: unsupported field")).WithMeta("field_no", fmt.Sprintf("%d", i))
					return
				}
				return
			}
			if !ast.IsExported(field.Names[0].Name) {
				continue
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
			// element
			fieldElementType, parseFieldElementTypeErr := types.parseExpr(ctx, field.Type, scope)
			if parseFieldElementTypeErr != nil {
				err = errors.Warning("forg: parse struct type failed").
					WithMeta("path", path).WithMeta("name", name).
					WithCause(parseFieldElementTypeErr).
					WithMeta("field_no", fmt.Sprintf("%d", i)).
					WithMeta("field", ft.Name)
				return
			}
			ft.Elements = []*Type{fieldElementType}
			typ.Elements = append(typ.Elements, ft)
		}
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
