package files

import (
	"go/ast"
	"go/parser"
	"go/token"
)

func ParseSource(filename string) (file *ast.File, err error) {
	file, err = parser.ParseFile(token.NewFileSet(), filename, nil, parser.AllErrors|parser.ParseComments)
	return
}
