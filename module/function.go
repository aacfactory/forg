package module

import (
	"context"
	"github.com/aacfactory/errors"
	"go/ast"
	"strings"
	"time"
)

type FunctionField struct {
	Type *Type
	Name string
}

type Function struct {
	mod             *Module
	hostServiceName string
	path            string
	filename        string
	file            *ast.File
	imports         Imports
	decl            *ast.FuncDecl
	Ident           string
	ConstIdent      string
	ProxyIdent      string
	Annotations     map[string]string
	Param           *FunctionField
	Result          *FunctionField
}

func (f *Function) Name() (name string) {
	name = f.Annotations["fn"]
	return
}

func (f *Function) Internal() (ok string) {
	ok = f.Annotations["internal"]
	return
}

func (f *Function) Title() (title string) {
	title = f.Annotations["title"]
	title = strings.TrimSpace(title)
	if title == "" {
		title = f.Name()
	}
	return
}

func (f *Function) Description() (description string) {
	description = f.Annotations["description"]
	return
}

func (f *Function) Validation() (ok bool) {
	_, ok = f.Annotations["validation"]
	return
}

func (f *Function) Authorization() (ok bool) {
	_, ok = f.Annotations["authorization"]
	return
}

func (f *Function) Permission() (ok bool) {
	_, ok = f.Annotations["permission"]
	return
}

func (f *Function) Deprecated() (ok bool) {
	_, ok = f.Annotations["deprecated"]
	return
}

func (f *Function) Barrier() (ok bool) {
	_, ok = f.Annotations["barrier"]
	return
}

func (f *Function) Timeout() (timeout time.Duration, has bool, err error) {
	s := ""
	s, has = f.Annotations["timeout"]
	if has {
		timeout, err = time.ParseDuration(s)
	}
	return
}

func (f *Function) SQL() (name string, has bool) {
	name, has = f.Annotations["sql"]
	return
}

func (f *Function) Transactional() (has bool) {
	_, has = f.Annotations["transactional"]
	return
}

func (f *Function) FieldImports() (v Imports) {
	v = Imports{}
	if f.Param != nil {
		paths := f.Param.Type.GetTopPaths()
		if paths != nil && len(paths) > 0 {
			for _, path := range paths {
				v.Add(&Import{
					Path:  path,
					Alias: "",
				})
			}
		}
	}
	if f.Result != nil {
		paths := f.Result.Type.GetTopPaths()
		if paths != nil && len(paths) > 0 {
			for _, path := range paths {
				v.Add(&Import{
					Path:  path,
					Alias: "",
				})
			}
		}
	}
	return
}

func (f *Function) Parse(ctx context.Context) (err error) {
	if ctx.Err() != nil {
		err = errors.Warning("forg: parse function failed").WithCause(ctx.Err()).
			WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
		return
	}
	if f.decl.Type.TypeParams != nil && f.decl.Type.TypeParams.List != nil && len(f.decl.Type.TypeParams.List) > 0 {
		err = errors.Warning("forg: parse function failed").WithCause(errors.Warning("function can not use paradigm")).
			WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
		return
	}

	// params
	params := f.decl.Type.Params
	if params == nil || params.List == nil || len(params.List) == 0 || len(params.List) > 2 {
		err = errors.Warning("forg: parse function failed").WithCause(errors.Warning("params length must be one or two")).
			WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
		return
	}
	if !isContextType(params.List[0].Type, f.imports) {
		err = errors.Warning("forg: parse function failed").WithCause(errors.Warning("first param must be context.Context")).
			WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
		return
	}
	if len(params.List) == 2 {
		param, parseParamErr := f.parseField(ctx, params.List[1])
		if parseParamErr != nil {
			err = errors.Warning("forg: parse function failed").WithCause(parseParamErr).
				WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
			return
		}
		f.Param = param
	}
	// results
	results := f.decl.Type.Results
	if results == nil || results.List == nil || len(results.List) == 0 || len(results.List) > 2 {
		err = errors.Warning("forg: parse function failed").WithCause(errors.Warning("results length must be one or two")).
			WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
		return
	}
	if len(results.List) == 1 {
		if !isCodeErrorType(results.List[0].Type, f.imports) {
			err = errors.Warning("forg: parse function failed").WithCause(errors.Warning("the last results must github.com/aacfactory/errors.CodeError")).
				WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
			return
		}
	} else {
		if !isCodeErrorType(results.List[1].Type, f.imports) {
			err = errors.Warning("forg: parse function failed").WithCause(errors.Warning("the last results must github.com/aacfactory/errors.CodeError")).
				WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
			return
		}
		result, parseResultErr := f.parseField(ctx, results.List[0])
		if parseResultErr != nil {
			err = errors.Warning("forg: parse function failed").WithCause(parseResultErr).
				WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
			return
		}
		f.Result = result
	}
	return
}

func (f *Function) parseField(ctx context.Context, field *ast.Field) (v *FunctionField, err error) {
	if len(field.Names) != 1 {
		err = errors.Warning("forg: field must has only one name")
		return
	}
	name := field.Names[0].Name
	typ, parseTypeErr := f.parseFieldType(ctx, field.Type)
	if parseTypeErr != nil {
		err = parseTypeErr
		return
	}
	v = &FunctionField{
		Type: typ,
		Name: name,
	}
	return
}

func (f *Function) parseFieldType(ctx context.Context, e ast.Expr) (typ *Type, err error) {
	switch e.(type) {
	case *ast.Ident:
		expr := e.(*ast.Ident)
		if expr.Obj == nil || expr.Obj.Decl == nil {
			// builtin
			err = errors.Warning("forg: field type only support value object or array")
			return
		}
		// type in same file
		decl, isTypeDecl := expr.Obj.Decl.(*ast.TypeSpec)
		if !isTypeDecl {
			err = errors.Warning("forg: field type only support value object or array")
			return
		}
		switch decl.Type.(type) {
		case *ast.StructType, *ast.ArrayType:
			typ, err = f.mod.ParseType(ctx, f.path, decl.Name.Name)
			if err != nil {
				return
			}
			break
		default:
			err = errors.Warning("forg: field type only support value object or array")
			return
		}
		break
	case *ast.SelectorExpr:
		expr := e.(*ast.SelectorExpr)
		ident, identOk := expr.X.(*ast.Ident)
		if !identOk {
			err = errors.Warning("forg: found selector field but x of expr is not indent")
			return
		}
		selectorImport, hasSelectorImport := f.imports.Find(ident.Name)
		if !hasSelectorImport {
			err = errors.Warning("forg: found selector field but can not found importer about it")
			return
		}
		selectorName := expr.Sel.Name
		typ, err = f.mod.ParseType(ctx, selectorImport.Path, selectorName)
		if err != nil {
			return
		}
		break
	case *ast.ArrayType:
		elementType, parseElementErr := f.parseFieldType(ctx, e.(*ast.ArrayType).Elt)
		if parseElementErr != nil {
			err = parseElementErr
			return
		}
		typ = &Type{
			Kind:        ArrayKind,
			Path:        "",
			Name:        "",
			Annotations: nil,
			Paradigms:   nil,
			Tags:        nil,
			Elements:    []*Type{elementType},
		}
		break
	default:
		err = errors.Warning("forg: field type only support value object or array")
		return
	}
	return
}

type Functions []*Function

func (fns Functions) Len() int {
	return len(fns)
}

func (fns Functions) Less(i, j int) bool {
	return fns[i].Ident < fns[j].Ident
}

func (fns Functions) Swap(i, j int) {
	fns[i], fns[j] = fns[j], fns[i]
	return
}
