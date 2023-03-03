package module

import (
	"bytes"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/forg/files"
	"golang.org/x/mod/modfile"
	"golang.org/x/sync/singleflight"
	"os"
	"path/filepath"
	"sync"
)

func New(path string) (v *Module, err error) {
	v, err = NewWithWork(path, "")
	return
}
func NewWithWork(path string, workPath string) (v *Module, err error) {
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
	if !files.ExistFile(path) {
		err = errors.Warning("forg: new module failed").
			WithCause(errors.Warning("forg: file was not found").WithMeta("path", path))
		return
	}
	dir := filepath.ToSlash(filepath.Dir(path))
	v = &Module{
		Dir:       dir,
		Path:      path,
		Name:      "",
		Version:   "",
		GoVersion: "",
		Requires:  make([]*Require, 0, 1),
		types: &Types{
			values:  sync.Map{},
			group:   singleflight.Group{},
			modules: make(map[string]string),
		},
		services: make(map[string]*Service),
	}
	parseErr := v.parse()
	if parseErr != nil {
		err = errors.Warning("forg: new module failed").
			WithCause(parseErr)
		return
	}
	if workPath != "" {
		work, parseWorkErr := parseWork(workPath)
		if parseWorkErr != nil {
			err = errors.Warning("forg: new module failed").
				WithCause(parseWorkErr)
			return
		}
		if len(work.Uses) > 0 {
			for modulePath, moduleDir := range work.Uses {
				replaced := false
				for _, require := range v.Requires {
					if require.Name == modulePath && require.Replace != nil {
						require.Replace = &Require{
							Dir:      moduleDir,
							Name:     modulePath,
							Version:  "",
							Replace:  nil,
							Indirect: false,
							Module:   nil,
						}
						replaced = true
						break
					}
				}
				if !replaced {
					v.Requires = append(v.Requires, &Require{
						Dir:      moduleDir,
						Name:     modulePath,
						Version:  "",
						Replace:  nil,
						Indirect: false,
						Module:   nil,
					})
				}
			}
		}
		if len(work.Replaces) > 0 {
			for _, replace := range work.Replaces {
				replaced := false
				for _, require := range v.Requires {
					if require.Name == replace.Name && require.Replace != nil {
						require.Replace = &Require{
							Dir:      replace.Replace.Dir,
							Name:     replace.Replace.Name,
							Version:  replace.Replace.Version,
							Replace:  nil,
							Indirect: false,
							Module:   nil,
						}
						replaced = true
						break
					}
				}
				if !replaced {
					v.Requires = append(v.Requires, replace)
				}
			}
		}
	}
	v.types.modules[v.Path] = v.Dir
	for _, require := range v.Requires {
		requireDir := require.Dir
		if require.Replace != nil {
			requireDir = require.Replace.Dir
		}
		v.types.modules[require.Name] = requireDir
	}
	loadServiceErr := v.loadServices()
	if loadServiceErr != nil {
		err = errors.Warning("forg: new module failed").
			WithCause(loadServiceErr)
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
	services  map[string]*Service
	types     *Types
}

func (mod *Module) parse() (err error) {
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

func (mod *Module) loadServices() (err error) {
	servicesDir := filepath.ToSlash(filepath.Join(mod.Dir, "modules"))
	entries, readServicesDirErr := os.ReadDir(servicesDir)
	if readServicesDirErr != nil {
		err = errors.Warning("read services dir failed").WithCause(readServicesDirErr).WithMeta("dir", servicesDir)
		return
	}
	if entries == nil || len(entries) == 0 {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		docFilename := filepath.ToSlash(filepath.Join(servicesDir, entry.Name(), "doc.go"))
		if !files.ExistFile(docFilename) {
			continue
		}
		service, loaded, loadErr := tryLoadService(mod, docFilename)
		if loadErr != nil {
			err = errors.Warning("load service failed").WithCause(loadErr).WithMeta("file", docFilename)
			return
		}
		if !loaded {
			continue
		}
		_, exist := mod.services[service.Name]
		if exist {
			err = errors.Warning("load service failed").WithCause(errors.Warning("forg: services was duplicated")).WithMeta("service", service.Name)
			return
		}
		mod.services[service.Name] = service
	}
	return
}

func (mod *Module) String() (s string) {
	buf := bytes.NewBuffer([]byte{})
	_, _ = buf.WriteString(fmt.Sprintf("name: %s\n", mod.Name))
	_, _ = buf.WriteString(fmt.Sprintf("version: %s\n", mod.Version))
	_, _ = buf.WriteString(fmt.Sprintf("goversion: %s\n", mod.GoVersion))
	for _, require := range mod.Requires {
		_, _ = buf.WriteString(fmt.Sprintf("requre: %s@%s", require.Name, require.Version))
		if require.Replace != nil {
			_, _ = buf.WriteString(fmt.Sprintf("=> %s", require.Replace.Name))
			if require.Replace.Version != "" {
				_, _ = buf.WriteString(fmt.Sprintf("@%s", require.Replace.Version))
			}
		}
		_, _ = buf.WriteString("\n")
	}
	for _, service := range mod.services {
		_, _ = buf.WriteString(fmt.Sprintf("service: %s", service.Name))
		if len(service.Components) > 0 {
			_, _ = buf.WriteString("component: ")
			for i, component := range service.Components {
				if i == 0 {
					_, _ = buf.WriteString(fmt.Sprintf("%s", component.Indent))
				} else {
					_, _ = buf.WriteString(fmt.Sprintf(", %s", component.Indent))
				}
			}
		}
		_, _ = buf.WriteString("\n")
		if len(service.Functions) > 0 {
			for _, function := range service.Functions {
				_, _ = buf.WriteString(fmt.Sprintf("fn: %s\n", function.Name()))
			}
		}
	}
	s = buf.String()
	return
}
