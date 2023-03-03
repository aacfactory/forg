package module

import "go/ast"

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

func newElement(expr ast.Expr, mod *Module, path string, fileImports Imports) (element *Element, err error) {

	return
}
