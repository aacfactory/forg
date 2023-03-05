package module

import (
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/forg/files"
	"golang.org/x/mod/modfile"
	"os"
	"path/filepath"
	"sync"
)

func parseWork(path string) (work *Work, err error) {
	if !filepath.IsAbs(path) {
		absolute, absoluteErr := filepath.Abs(path)
		if absoluteErr != nil {
			err = errors.Warning("forg: parse work failed").
				WithCause(errors.Warning("forg: get absolute representation of work file failed").WithCause(absoluteErr).WithMeta("work", path))
			return
		}
		path = absolute
	}
	if !files.ExistFile(path) {
		err = errors.Warning("forg: parse work failed").
			WithCause(errors.Warning("forg: file was not found").WithMeta("work", path))
		return
	}
	dir := filepath.Dir(path)
	path = filepath.ToSlash(path)
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		err = errors.Warning("forg: parse work failed").WithMeta("work", path).WithCause(readErr)
		return
	}
	file, parseErr := modfile.ParseWork(path, data, nil)
	if parseErr != nil {
		err = errors.Warning("forg: parse work failed").WithMeta("work", path).WithCause(parseErr)
		return
	}
	work = &Work{
		Uses:     make([]*Module, 0, 1),
		Replaces: make([]*Module, 0, 1),
	}
	if file.Use != nil && len(file.Use) > 0 {
		for _, use := range file.Use {
			usePath := use.Path
			if filepath.IsAbs(usePath) {
				usePath = filepath.ToSlash(usePath)
			} else {
				usePath = filepath.ToSlash(filepath.Join(dir, usePath))
			}
			moduleFile := filepath.ToSlash(filepath.Join(usePath, "mod.go"))
			if !files.ExistFile(moduleFile) {
				err = errors.Warning("forg: parse work failed").WithMeta("work", path).
					WithCause(errors.Warning("forg: mod file was not found").
						WithMeta("mod", moduleFile))
				return
			}
			modData, readModErr := os.ReadFile(moduleFile)
			if readModErr != nil {
				err = errors.Warning("forg: parse work failed").WithMeta("work", path).
					WithCause(errors.Warning("forg: read mod file failed").WithCause(readModErr).WithMeta("mod", moduleFile))
				return
			}
			mf, parseModErr := modfile.Parse(moduleFile, modData, nil)
			if parseModErr != nil {
				err = errors.Warning("forg: parse work failed").WithMeta("work", path).
					WithCause(errors.Warning("forg: parse mod file failed").WithCause(parseModErr).WithMeta("mod", moduleFile))
				return
			}
			work.Uses = append(work.Uses, &Module{
				Dir:      usePath,
				Path:     mf.Module.Mod.Path,
				Version:  "",
				Requires: nil,
				Work:     work,
				Replace:  nil,
				locker:   &sync.Mutex{},
				parsed:   false,
				services: nil,
				types:    nil,
			})
		}
	}
	if file.Replace != nil && len(file.Replace) > 0 {
		for _, replace := range file.Replace {
			replaceDir := ""
			if replace.New.Version != "" {
				replaceDir = filepath.Join(PKG(), fmt.Sprintf("%s@%s", replace.New.Path, replace.New.Version))
			} else {
				replaceDir = filepath.Join(PKG(), replace.New.Path)
			}
			replaceDir = filepath.ToSlash(replaceDir)
			if !files.ExistFile(replaceDir) {
				err = errors.Warning("forg: parse work failed").WithMeta("work", path).
					WithCause(errors.Warning("forg: replace dir was not found").WithMeta("replace", replaceDir))
				return
			}
			moduleFile := filepath.ToSlash(filepath.Join(replaceDir, "mod.go"))
			if !files.ExistFile(moduleFile) {
				err = errors.Warning("forg: parse work failed").WithMeta("work", path).
					WithCause(errors.Warning("forg: replace mod file was not found").
						WithMeta("mod", moduleFile))
				return
			}
			modData, readModErr := os.ReadFile(moduleFile)
			if readModErr != nil {
				err = errors.Warning("forg: parse work failed").WithMeta("work", path).
					WithCause(errors.Warning("forg: read replace mod file failed").WithCause(readModErr).WithMeta("mod", moduleFile))
				return
			}
			mf, parseModErr := modfile.Parse(moduleFile, modData, nil)
			if parseModErr != nil {
				err = errors.Warning("forg: parse work failed").WithMeta("work", path).
					WithCause(errors.Warning("forg: parse replace mod file failed").WithCause(parseModErr).WithMeta("mod", moduleFile))
				return
			}
			work.Replaces = append(work.Replaces, &Module{
				Dir:      "",
				Path:     replace.Old.Path,
				Version:  replace.Old.Version,
				Requires: nil,
				Work:     nil,
				Replace: &Module{
					Dir:      replaceDir,
					Path:     mf.Module.Mod.Path,
					Version:  mf.Module.Mod.Version,
					Requires: nil,
					Work:     nil,
					Replace:  nil,
					locker:   &sync.Mutex{},
					parsed:   false,
					services: nil,
					types:    nil,
				},
				locker:   &sync.Mutex{},
				parsed:   false,
				services: nil,
				types:    nil,
			})
		}
	}
	return
}

type Work struct {
	Uses     []*Module
	Replaces []*Module
}

func (work *Work) Use(path string) (v *Module, used bool) {
	for _, use := range work.Uses {
		if use.Path == path {
			v = use
			used = true
			break
		}
	}
	return
}
