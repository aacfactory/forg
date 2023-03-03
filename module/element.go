package module

import (
	"context"
	"go/ast"
)

const (
	BuiltinKind = ElementKind(iota + 1)
	StructKind
	PointKind
	ArrayKind
	MapKind
)

type ElementKind int

type Element struct {
	Kind   ElementKind
	Indent string
	Struct *Struct
	X      *Element
	Y      *Element
}

func newElement(ctx context.Context, expr ast.Expr, mod *Module, path string, fileImports Imports) (element *Element, err error) {

	return
}

func tryNewBuiltinElement(ctx context.Context, expr ast.Expr, mod *Module, path string, fileImports Imports) (element *Element, ok bool, err error) {

	return
}

func tryNewSameScopeElement(ctx context.Context, expr ast.Expr, mod *Module, path string, fileImports Imports) (element *Element, ok bool, err error) {

	return
}

func tryNewSelectorElement(ctx context.Context, expr ast.Expr, mod *Module, path string, fileImports Imports) (element *Element, ok bool, err error) {

	return
}

func tryNewStarElement(ctx context.Context, expr ast.Expr, mod *Module, path string, fileImports Imports) (element *Element, ok bool, err error) {

	return
}

func tryNewArrayElement(ctx context.Context, expr ast.Expr, mod *Module, path string, fileImports Imports) (element *Element, ok bool, err error) {

	return
}

func tryNewMapElement(ctx context.Context, expr ast.Expr, mod *Module, path string, fileImports Imports) (element *Element, ok bool, err error) {

	return
}
