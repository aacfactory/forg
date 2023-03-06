package module

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

func newSource(path string, dir string) *Sources {
	return &Sources{
		locker:  &sync.Mutex{},
		dir:     dir,
		path:    path,
		readers: make(map[string]*SourceDirReader),
	}
}

type Sources struct {
	locker  sync.Locker
	dir     string
	path    string
	readers map[string]*SourceDirReader
}

func (sources *Sources) destinationPath(path string) (v string, err error) {
	sub, cut := strings.CutPrefix(path, sources.path)
	if !cut {
		err = errors.Warning("forg: path is not in module").WithMeta("path", path).WithMeta("mod", sources.path)
		return
	}
	v = filepath.ToSlash(filepath.Join(sources.dir, sub))
	return
}

func (sources *Sources) ReadFile(path string, filename string) (file *ast.File, err error) {
	sources.locker.Lock()
	reader, has := sources.readers[path]
	sources.locker.Unlock()
	if has {
		for _, sf := range reader.files {
			_, sfn := filepath.Split(sf.filename)
			if sfn == filename {
				file, err = sf.File()
				return
			}
		}
		err = errors.Warning("forg: read file failed").WithCause(errors.Warning("no file found")).WithMeta("path", path).WithMeta("file", filename).WithMeta("mod", sources.path)
		return
	}
	dir, dirErr := sources.destinationPath(path)
	if dirErr != nil {
		err = errors.Warning("forg: read file failed").WithCause(dirErr).WithMeta("path", path).WithMeta("file", filename).WithMeta("mod", sources.path)
		return
	}
	file, err = parser.ParseFile(token.NewFileSet(), filepath.ToSlash(filepath.Join(dir, filename)), nil, parser.AllErrors|parser.ParseComments)
	if err != nil {
		err = errors.Warning("forg: read file failed").WithCause(err).WithMeta("path", path).WithMeta("file", filename).WithMeta("mod", sources.path)
		return
	}
	return
}

func (sources *Sources) getReader(path string) (reader *SourceDirReader, err error) {
	sources.locker.Lock()
	has := false
	reader, has = sources.readers[path]
	if !has {
		dir, dirErr := sources.destinationPath(path)
		if dirErr != nil {
			err = errors.Warning("forg: get source reader failed").WithCause(dirErr).WithMeta("path", path).WithMeta("mod", sources.path)
			sources.locker.Unlock()
			return
		}
		entries, readErr := os.ReadDir(dir)
		if readErr != nil {
			err = errors.Warning("forg: get source reader failed").WithCause(readErr).WithMeta("path", path).WithMeta("mod", sources.path)
			sources.locker.Unlock()
			return
		}
		if entries == nil || len(entries) == 0 {
			err = errors.Warning("forg: get source reader failed").WithCause(errors.Warning("no entries found")).WithMeta("path", path).WithMeta("mod", sources.path)
			sources.locker.Unlock()
			return
		}
		files := make([]*SourceFile, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() || strings.HasSuffix(entry.Name(), "_test.go") || filepath.Ext(entry.Name()) != ".go" {
				continue
			}
			files = append(files, &SourceFile{
				locker:   &sync.Mutex{},
				parsed:   false,
				filename: filepath.ToSlash(filepath.Join(dir, entry.Name())),
				file:     nil,
				err:      nil,
			})
		}
		reader = &SourceDirReader{
			locker: &sync.Mutex{},
			files:  files,
		}
		sources.readers[path] = reader
	}
	sources.locker.Unlock()
	return
}

func (sources *Sources) ReadDir(path string, fn func(file *ast.File, filename string) (err error)) (err error) {
	reader, readerErr := sources.getReader(path)
	if readerErr != nil {
		err = errors.Warning("forg: read source dir failed").WithCause(readerErr).WithMeta("path", path).WithMeta("mod", sources.path)
		return
	}
	err = reader.Each(fn)
	return
}

func (sources *Sources) FindFileInDir(path string, matcher func(file *ast.File) (ok bool)) (file *ast.File, err error) {
	reader, readerErr := sources.getReader(path)
	if readerErr != nil {
		err = errors.Warning("forg: find file in source dir failed").WithCause(readerErr).WithMeta("path", path).WithMeta("mod", sources.path)
		return
	}
	file, err = reader.Find(matcher)
	return
}

type SourceDirReader struct {
	locker sync.Locker
	files  []*SourceFile
}

func (reader *SourceDirReader) Each(fn func(file *ast.File, filename string) (err error)) (err error) {
	for _, sf := range reader.files {
		file, fileErr := sf.File()
		if fileErr != nil {
			err = fileErr
			return
		}
		err = fn(file, sf.filename)
		if err != nil {
			return
		}
	}
	return
}

func (reader *SourceDirReader) Find(matcher func(file *ast.File) (ok bool)) (file *ast.File, err error) {
	for _, sf := range reader.files {
		file, err = sf.File()
		if err != nil {
			return
		}
		ok := matcher(file)
		if ok {
			return
		}
	}
	err = errors.Warning("forg: source file was not found")
	return
}

type SourceFile struct {
	locker   sync.Locker
	parsed   bool
	filename string
	file     *ast.File
	err      error
}

func (sf *SourceFile) File() (file *ast.File, err error) {
	sf.locker.Lock()
	defer sf.locker.Unlock()
	if !sf.parsed {
		file, err = parser.ParseFile(token.NewFileSet(), sf.filename, nil, parser.AllErrors|parser.ParseComments)
		sf.file = file
		sf.err = errors.Warning("forg: parse source failed").WithCause(err).WithMeta("file", sf.filename)
		sf.parsed = true
		return
	}
	sf.locker.Unlock()
	return
}
