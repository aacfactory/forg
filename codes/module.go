package codes

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/forg/files"
	"golang.org/x/mod/modfile"
	"os"
	"path/filepath"
	"sync"
)

func NewModule(path string) (v *Module, err error) {
	path = filepath.ToSlash(path)
	if !filepath.IsAbs(path) {
		absolute, absoluteErr := filepath.Abs(path)
		if absoluteErr != nil {
			err = errors.Warning("forg: get absolute representation of module file path failed").WithCause(absoluteErr).WithMeta("path", path)
			return
		}
		path = absolute
	}
	dir := filepath.ToSlash(filepath.Dir(path))
	pkg := ""
	gopath, hasGOPATH := GOPATH()
	goroot, hasGOROOT := GOROOT()
	if !hasGOPATH && !hasGOROOT {
		err = errors.Warning("forg: GOPATH or GOROOT was not found")
		return
	}
	if hasGOPATH {
		gopath = filepath.Join(gopath, "pkg/mod")
	}
	if hasGOROOT {
		goroot = filepath.Join(gopath, "pkg/mod")
	}
	path = filepath.ToSlash(pkg)
	v = &Module{
		Dir:                dir,
		Path:               path,
		GOPATH:             gopath,
		GOROOT:             goroot,
		Name:               "",
		Version:            "",
		GoVersion:          "",
		Requires:           make([]*Require, 0, 1),
		enableReadRequires: true,
	}
	return
}

type Module struct {
	Dir                string
	Path               string
	GOPATH             string
	GOROOT             string
	Name               string
	Version            string
	GoVersion          string
	Requires           []*Require
	enableReadRequires bool
	locker             sync.Mutex
	structs            map[string]*Struct
}

func (mod *Module) Read(ctx context.Context) (result interface{}, err error) {
	data, readErr := os.ReadFile(mod.Path)
	if readErr != nil {
		err = errors.Warning("forg: read mod file failed").WithMeta("path", mod.Path).WithCause(readErr)
		return
	}
	f, parseErr := modfile.Parse(mod.Path, data, nil)
	if parseErr != nil {
		err = errors.Warning("forg: parse mod file failed").WithMeta("path", mod.Path).WithCause(parseErr)
		return
	}
	mod.Name = f.Module.Mod.Path
	mod.Version = f.Module.Mod.Version
	mod.GoVersion = f.Go.Version
	if f.Require != nil {
		for _, require := range f.Require {
			requireDir, hasRequireDir := mod.buildRequireDir(require.Mod.String())
			if !hasRequireDir {
				err = errors.Warning("forg: can not get require dir").WithMeta("require", require.Mod.String())
				return
			}
			mod.Requires = append(mod.Requires, &Require{
				Path:     requireDir,
				Name:     require.Mod.Path,
				Version:  require.Mod.Version,
				Replace:  nil,
				Indirect: require.Indirect,
				Module:   nil,
			})
		}
	}
	if f.Replace != nil {
		for _, replace := range f.Replace {
			on := replace.Old.Path
			ov := replace.Old.Version
			for _, require := range mod.Requires {
				if require.Name == on && require.Version == ov {
					requireDir, hasRequireDir := mod.buildRequireDir(replace.New.String())
					if !hasRequireDir {
						err = errors.Warning("forg: can not get replace  dir").WithMeta("require", replace.New.String())
						return
					}
					require.Replace = &Require{
						Path:     requireDir,
						Name:     replace.New.Path,
						Version:  replace.New.Version,
						Replace:  nil,
						Indirect: false,
						Module:   nil,
					}
					break
				}
			}
		}
	}
	if mod.enableReadRequires {
		// todo Dynamic process, get process from ctx, 且 ctx要被传过去，而不是用process头上ctx

	}
	result = fmt.Sprintf("read %s@%s succeed", mod.Name, mod.Version)
	return
}

func (mod *Module) buildRequireDir(path string) (dir string, ok bool) {
	if mod.GOPATH != "" {
		dir = filepath.ToSlash(filepath.Join(mod.GOPATH, path))
		ok = files.ExistFile(dir)
		if ok {
			return
		}
	}
	if mod.GOROOT != "" {
		dir = filepath.ToSlash(filepath.Join(mod.GOROOT, path))
		ok = files.ExistFile(dir)
		if ok {
			return
		}
	}
	return
}

func (mod *Module) disableReadRequires() {
	mod.enableReadRequires = false
}

func (mod *Module) getStruct(importer string, name string) (structure *Struct, err error) {

	return
}

type Require struct {
	Path     string
	Name     string
	Version  string
	Replace  *Require
	Indirect bool
	Module   *Module
}

func (require *Require) Read(ctx context.Context) (result interface{}, err error) {
	path := ""
	if require.Replace == nil {
		path = filepath.Join(require.Path, "mod.go")
	} else {
		path = filepath.Join(require.Replace.Path, "mod.go")
	}
	mod, modErr := NewModule(path)
	if modErr != nil {
		err = errors.Warning("forg: read require module failed").WithCause(modErr).WithMeta("require", fmt.Sprintf("%s@%s", require.Name, require.Version))
		return
	}
	mod.disableReadRequires()
	_, err = mod.Read(ctx)
	if err != nil {
		err = errors.Warning("forg: read require module file failed").WithCause(err).WithMeta("require", fmt.Sprintf("%s@%s", require.Name, require.Version))
		return
	}
	require.Module = mod
	result = fmt.Sprintf("read %s@%s succeed", require.Name, require.Version)
	return
}
