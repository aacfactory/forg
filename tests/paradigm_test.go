package tests_test

import (
	"fmt"
	"github.com/aacfactory/forg/tests"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParadigm(t *testing.T) {
	path, _ := filepath.Abs(`./paradigm.go`)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.AllErrors)
	if err != nil {
		t.Error(err)
		return
	}
	for _, decl := range f.Decls {
		fmt.Println(decl, reflect.TypeOf(decl))
		genDecl := decl.(*ast.GenDecl)
		for _, spec := range genDecl.Specs {
			fmt.Println(spec, reflect.TypeOf(spec))
			typeSpec := spec.(*ast.TypeSpec)
			fmt.Println(typeSpec.Name, typeSpec.TypeParams)
			for _, field := range typeSpec.TypeParams.List {
				// [E]m any
				fmt.Println(field.Names, field.Type, field.Tag, reflect.TypeOf(field.Type))
				//field.Type.(*ast.BinaryExpr).X
				be := field.Type.(*ast.BinaryExpr)
				fmt.Println(be.X, reflect.TypeOf(be.X), be.Op, be.Y, reflect.TypeOf(be.Y))
				bex := be.X.(*ast.BinaryExpr)
				fmt.Println(bex.X, reflect.TypeOf(bex.X), bex.Op, bex.Y, reflect.TypeOf(bex.Y))
				bexx := bex.X.(*ast.UnaryExpr)
				fmt.Println(bexx.X, reflect.TypeOf(bexx.X), bexx.Op)

			}
		}
	}
}

type S struct {
	V   string
	C1  uint64
	C2  uint64
	C3  uint64
	C4  uint64
	C5  uint64
	C6  uint64
	C7  uint64
	C8  uint64
	C9  uint64
	C10 uint64
	C11 uint64
}

type I interface {
	H()
}

func Hello[E S](e E) (v tests.Paradigm[int], err error) {
	//fmt.Println(e, reflect.ValueOf(e).IsZero())
	return
}

func Hello1(s S) (v S, err error) {
	return
}

func Hello2(s *S) (v *S, err error) {
	return
}

func BenchmarkHello(b *testing.B) {
	// 0.2539 ns/op
	// 0.2540 ns/op
	for i := 0; i < b.N; i++ {
		_, _ = Hello(S{})
	}
}

func BenchmarkHello1(b *testing.B) {
	// 0.2567 ns/op
	//  0.2541 ns/op
	for i := 0; i < b.N; i++ {
		_, _ = Hello1(S{})
	}
}

func BenchmarkHello2(b *testing.B) {
	// 0.2593 ns/op
	// 0.2600 ns/op
	for i := 0; i < b.N; i++ {
		_, _ = Hello2(&S{})
	}
}
