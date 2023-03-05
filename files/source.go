package files

import (
	"github.com/aacfactory/errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func ParseSource(filename string) (file *ast.File, err error) {
	file, err = parser.ParseFile(token.NewFileSet(), filename, nil, parser.AllErrors|parser.ParseComments)
	return
}

func NewSources() *Sources {
	return &Sources{
		locker: &sync.Mutex{},
		files:  make(map[string][]*ast.File),
	}
}

type Sources struct {
	locker sync.Locker
	files  map[string][]*ast.File
}

func (sources *Sources) File(dir string) (set []*ast.File, err error) {
	sources.locker.Lock()
	defer sources.locker.Unlock()
	entries, dirErr := os.ReadDir(dir)
	if dirErr != nil {
		err = errors.Warning("forg: read dir failed").WithMeta("dir", dir).WithCause(dirErr)
		return
	}
	if entries == nil || len(entries) == 0 {
		sources.files[dir] = make([]*ast.File, 0, 1)
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
	}
	// todo files value = 渐进式读
	return
}
