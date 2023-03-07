package module

import (
	"context"
	"github.com/aacfactory/errors"
	"go/ast"
	"strings"
	"time"
)

type FunctionField struct {
	Name      string
	Paradigms []*TypeParadigm
	Type      *Type
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
		typePath := f.Param.Type.GetTopPath()
		if typePath != "" {
			v.Add(&Import{
				Path:  typePath,
				Alias: "",
			})
		}
		if f.Param.Paradigms != nil && len(f.Param.Paradigms) > 0 {
			for _, paradigm := range f.Param.Paradigms {
				if paradigm.Types != nil && len(paradigm.Types) > 0 {
					for _, tp := range paradigm.Types {
						paradigmPath := tp.GetTopPath()
						if paradigmPath != "" {
							v.Add(&Import{
								Path:  paradigmPath,
								Alias: "",
							})
						}
					}
				}
			}
		}
	}
	if f.Result != nil {
		typePath := f.Result.Type.GetTopPath()
		if typePath != "" {
			v.Add(&Import{
				Path:  typePath,
				Alias: "",
			})
		}
		if f.Result.Paradigms != nil && len(f.Result.Paradigms) > 0 {
			for _, paradigm := range f.Result.Paradigms {
				if paradigm.Types != nil && len(paradigm.Types) > 0 {
					for _, tp := range paradigm.Types {
						paradigmPath := tp.GetTopPath()
						if paradigmPath != "" {
							v.Add(&Import{
								Path:  paradigmPath,
								Alias: "",
							})
						}
					}
				}
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
	typ, paradigms, parseTypeErr := f.parseFieldType(ctx, field.Type)
	if parseTypeErr != nil {
		err = parseTypeErr
		return
	}
	v = &FunctionField{
		Name:      name,
		Paradigms: paradigms,
		Type:      typ,
	}
	return
}

func (f *Function) parseFieldType(ctx context.Context, e ast.Expr) (typ *Type, paradigms []*TypeParadigm, err error) {
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
		if decl.TypeParams != nil && decl.TypeParams.NumFields() > 0 {
			paradigms, err = f.mod.types.parseTypeParadigms(ctx, decl.TypeParams, &TypeScope{
				Path:       f.path,
				Mod:        f.mod,
				Imports:    f.imports,
				GenericDoc: "",
			})
			if err != nil {
				err = errors.Warning("forg: parse paradigms field failed").WithCause(err)
				return
			}
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

		typ = &Type{
			Kind:        ArrayKind,
			Path:        "",
			Name:        "",
			Annotations: nil,
			Paradigms:   nil,
			Tags:        nil,
			Elements:    []*Type{},
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
