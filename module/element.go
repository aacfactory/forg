package module

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
