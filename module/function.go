package module

import (
	"context"
	"go/ast"
	"strings"
	"time"
)

type FunctionField struct {
	Import  *Import
	Element *Element
	Name    string
}

type Function struct {
	mod             *Module
	hostFileImports Imports
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
	if f.Param != nil && f.Param.Import != nil {
		v.Add(f.Param.Import)
	}
	if f.Result != nil && f.Result.Import != nil {
		v.Add(f.Result.Import)
	}
	return
}

func (f *Function) Parse(ctx context.Context) (err error) {

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
