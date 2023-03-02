package module

import (
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/forg/files"
	"golang.org/x/mod/modfile"
	"os"
	"path/filepath"
)

func parseWork(path string) (work *Work, err error) {
	if !filepath.IsAbs(path) {
		absolute, absoluteErr := filepath.Abs(path)
		if absoluteErr != nil {
			err = errors.Warning("forg: parse work failed").
				WithCause(errors.Warning("forg: get absolute representation of work file failed").WithCause(absoluteErr).WithMeta("path", path))
			return
		}
		path = absolute
	}
	if !files.ExistFile(path) {
		err = errors.Warning("forg: parse work failed").
			WithCause(errors.Warning("forg: file was not found").WithMeta("path", path))
		return
	}
	dir := filepath.Dir(path)
	path = filepath.ToSlash(path)
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		err = errors.Warning("forg: parse work failed").WithMeta("path", path).WithCause(readErr)
		return
	}
	file, parseErr := modfile.ParseWork(path, data, nil)
	if parseErr != nil {
		err = errors.Warning("forg: parse work failed").WithMeta("path", path).WithCause(parseErr)
		return
	}
	work = &Work{
		Uses:     make(map[string]string),
		Replaces: make([]*Require, 0, 1),
	}
	if file.Use != nil && len(file.Use) > 0 {
		for _, use := range file.Use {
			usePath := use.Path
			if filepath.IsAbs(usePath) {
				usePath = filepath.ToSlash(usePath)
			} else {
				usePath = filepath.Join(dir, usePath)
			}
			module := use.ModulePath
			if module == "" {
				moduleFile := filepath.Join(usePath, "mod.go")
				modData, readModErr := os.ReadFile(moduleFile)
				if readModErr != nil {
					err = errors.Warning("forg: parse work failed").WithMeta("path", path).
						WithCause(errors.Warning("forg: read mod file failed").WithCause(readModErr).WithMeta("mod", moduleFile))
					return
				}
				mod, parseModErr := modfile.Parse(moduleFile, modData, nil)
				if parseModErr != nil {
					err = errors.Warning("forg: parse work failed").WithMeta("path", path).
						WithCause(errors.Warning("forg: parse mod file failed").WithCause(parseModErr).WithMeta("mod", moduleFile))
					return
				}
				module = mod.Module.Mod.Path
			}
			work.Uses[module] = usePath
		}
	}
	if file.Replace != nil && len(file.Replace) > 0 {
		gopath, hasGOPATH := GOPATH()
		goroot, hasGOROOT := GOROOT()
		if !hasGOPATH && !hasGOROOT {
			err = errors.Warning("forg: parse work failed").
				WithCause(errors.Warning("forg: GOPATH or GOROOT was not found"))
			return
		}
		for _, replace := range file.Replace {
			replaceDir := ""
			if hasGOPATH {
				replaceDir = filepath.Join(gopath, "pkg/mod", replace.New.Path)
			} else if hasGOROOT {
				replaceDir = filepath.Join(goroot, "pkg/mod", replace.New.Path)
			}
			if replace.New.Version != "" {
				replaceDir = fmt.Sprintf("%s@%s", replaceDir, replace.New.Version)
			}
			replaceDir = filepath.ToSlash(replaceDir)
			if !files.ExistFile(replaceDir) {
				err = errors.Warning("forg: parse work file failed").WithMeta("path", replaceDir).WithCause(errors.Warning("forg: replace dir of require is not exist"))
				return
			}
			work.Replaces = append(work.Replaces, &Require{
				Dir:     "",
				Name:    replace.Old.Path,
				Version: replace.Old.Version,
				Replace: &Require{
					Dir:      replaceDir,
					Name:     replace.New.Path,
					Version:  replace.New.Version,
					Replace:  nil,
					Indirect: false,
					Module:   nil,
				},
				Indirect: false,
				Module:   nil,
			})
		}
	}
	return
}

type Work struct {
	Uses     map[string]string
	Replaces []*Require
}
