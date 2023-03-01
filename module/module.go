package module

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

func New(ctx context.Context, path string) (v *Module, err error) {
	path = filepath.ToSlash(path)
	if !filepath.IsAbs(path) {
		absolute, absoluteErr := filepath.Abs(path)
		if absoluteErr != nil {
			err = errors.Warning("forg: new module failed").
				WithCause(errors.Warning("forg: get absolute representation of module file path failed").WithCause(absoluteErr).WithMeta("path", path))
			return
		}
		path = absolute
	}
	dir := filepath.ToSlash(filepath.Dir(path))
	v = &Module{
		Dir:       dir,
		Path:      path,
		Name:      "",
		Version:   "",
		GoVersion: "",
		Requires:  make([]*Require, 0, 1),
	}
	parseErr := v.parse(ctx)
	if parseErr != nil {
		err = errors.Warning("forg: new module failed").
			WithCause(parseErr)
		return
	}
	return
}

type Module struct {
	Dir       string
	Path      string
	Name      string
	Version   string
	GoVersion string
	Requires  []*Require
	locker    sync.Mutex
	structs   map[string]*Struct
}

func (mod *Module) parse(ctx context.Context) (err error) {
	gopath, hasGOPATH := GOPATH()
	goroot, hasGOROOT := GOROOT()
	if !hasGOPATH && !hasGOROOT {
		err = errors.Warning("forg: parse module failed").
			WithCause(errors.Warning("forg: GOPATH or GOROOT was not found"))
		return
	}
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
			name := require.Mod.Path
			version := require.Mod.Version
			dir := ""
			if hasGOPATH {
				dir = filepath.Join(gopath, "pkg/mod", fmt.Sprintf("%s@%s", name, version))
			} else if hasGOROOT {
				dir = filepath.Join(goroot, "pkg/mod", fmt.Sprintf("%s@%s", name, version))
			}
			dir = filepath.ToSlash(dir)
			if !files.ExistFile(dir) {
				err = errors.Warning("forg: parse mod file failed").WithMeta("path", dir).WithCause(errors.Warning("forg: require dir is not exist"))
				return
			}
			mod.Requires = append(mod.Requires, &Require{
				Dir:      dir,
				Name:     name,
				Version:  version,
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
					name := replace.New.Path
					version := replace.New.Version
					dir := ""
					if hasGOPATH {
						dir = filepath.Join(gopath, "pkg/mod", name)
					} else if hasGOROOT {
						dir = filepath.Join(goroot, "pkg/mod", name)
					}
					if version != "" {
						dir = fmt.Sprintf("%s@%s", dir, version)
					}
					dir = filepath.ToSlash(dir)
					if !files.ExistFile(dir) {
						err = errors.Warning("forg: parse mod file failed").WithMeta("path", dir).WithCause(errors.Warning("forg: replace dir of require is not exist"))
						return
					}
					require.Replace = &Require{
						Dir:      dir,
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
	return
}

func (mod *Module) getStruct(importer string, name string) (structure *Struct, err error) {

	return
}
