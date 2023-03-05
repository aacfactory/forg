package module

import (
	"context"
	"github.com/aacfactory/errors"
	"go/ast"
	"strings"
	"time"
)

type FunctionField struct {
	Paths []string
	Type  *Type
	Name  string
}

type Function struct {
	mod             *Module
	hostFileImports Imports
	hostServiceName string
	path            string
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

func (f *Function) Imports() (v Imports) {
	v = Imports{}
	if f.Param != nil && f.Param.Paths != nil && len(f.Param.Paths) > 0 {
		for _, p := range f.Param.Paths {
			v.Add(&Import{
				Path:  p,
				Alias: "",
			})
		}
	}
	if f.Result != nil && f.Result.Paths != nil && len(f.Result.Paths) > 0 {
		for _, p := range f.Result.Paths {
			v.Add(&Import{
				Path:  p,
				Alias: "",
			})
		}
	}
	return
}

func (f *Function) Parse(ctx context.Context) (err error) {
	if ctx.Err() != nil {
		err = errors.Warning("forg: parse function failed").WithCause(ctx.Err()).
			WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident)
		return
	}
	if f.decl.Type.TypeParams != nil && f.decl.Type.TypeParams.List != nil && len(f.decl.Type.TypeParams.List) > 0 {
		err = errors.Warning("forg: parse function failed").WithCause(errors.Warning("function can not use paradigm")).
			WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident)
		return
	}
	scope := &TypeScope{
		Path:    f.path,
		Imports: f.hostFileImports,
	}
	// params
	params := f.decl.Type.Params
	if params == nil || params.List == nil || len(params.List) == 0 || len(params.List) > 2 {
		err = errors.Warning("forg: parse function failed").WithCause(errors.Warning("params length must be one or two")).
			WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident)
		return
	}
	if !isContextType(params.List[0].Type, scope) {
		err = errors.Warning("forg: parse function failed").WithCause(errors.Warning("first param must be context.Context")).
			WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident)
		return
	}
	if len(params.List) == 2 {
		// todo fn的参数和结果都先自己把expr拿出，然后在types去parse（expr+scope）
		paramType, paramTypeErr := f.mod.types.parse(ctx, params.List[1].Type, scope)
		if paramTypeErr != nil {
			err = errors.Warning("forg: parse function failed").WithCause(paramTypeErr).
				WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident)
			return
		}
		if paramType.Kind != StructKind && paramType.Kind != ArrayKind {
			err = errors.Warning("forg: parse function failed").WithCause(errors.Warning("second param must be struct kind or array kind")).
				WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident)
			return
		}
		f.Param = &FunctionField{
			Paths: make([]string, 0, 1),
			Type:  paramType,
			Name:  params.List[1].Names[0].Name,
		}
		paramPath := paramType.GetPath()
		if paramPath != "" {
			f.Param.Paths = append(f.Param.Paths, paramPath)
		}
		paramParadigmPaths := paramType.GetParadigmPaths()
		if paramParadigmPaths != nil && len(paramParadigmPaths) > 0 {
			f.Param.Paths = append(f.Param.Paths, paramParadigmPaths...)
		}
	}
	// results
	results := f.decl.Type.Results
	if results == nil || results.List == nil || len(results.List) == 0 || len(results.List) > 2 {
		err = errors.Warning("forg: parse function failed").WithCause(errors.Warning("results length must be one or two")).
			WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident)
		return
	}
	if len(results.List) == 1 {
		if !isCodeErrorType(results.List[0].Type, scope) {
			err = errors.Warning("forg: parse function failed").WithCause(errors.Warning("the last results must github.com/aacfactory/errors.CodeError")).
				WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident)
			return
		}
	} else {
		if !isCodeErrorType(results.List[1].Type, scope) {
			err = errors.Warning("forg: parse function failed").WithCause(errors.Warning("the last results must github.com/aacfactory/errors.CodeError")).
				WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident)
			return
		}
		resultType, resultTypeErr := f.mod.types.parse(ctx, results.List[0].Type, scope)
		if resultTypeErr != nil {
			err = errors.Warning("forg: parse function failed").WithCause(resultTypeErr).
				WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident)
			return
		}
		if resultType.Kind != StructKind && resultType.Kind != ArrayKind && resultType.Kind != MapKind {
			err = errors.Warning("forg: parse function failed").WithCause(errors.Warning("second param must be one of struct kind, array kind or map kind")).
				WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident)
			return
		}
		f.Result = &FunctionField{
			Paths: make([]string, 0, 1),
			Type:  resultType,
			Name:  results.List[0].Names[0].Name,
		}
		resultPath := resultType.GetPath()
		if resultPath != "" {
			f.Result.Paths = append(f.Result.Paths, resultPath)
		}
		resultParadigmPaths := resultType.GetParadigmPaths()
		if resultParadigmPaths != nil && len(resultParadigmPaths) > 0 {
			f.Result.Paths = append(f.Result.Paths, resultParadigmPaths...)
		}
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
