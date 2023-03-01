package tests

type Paradigm[E ~int | ~string | float64] struct {
	Value E
}
