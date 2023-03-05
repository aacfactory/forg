package module

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/forg/files"
	"go/ast"
	"golang.org/x/mod/modfile"
	"golang.org/x/sync/singleflight"
	"os"
	"path/filepath"
	"sort"
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
	pkgErr := initPkgDir()
	if pkgErr != nil {
		err = errors.Warning("forg: new module failed").
			WithCause(pkgErr)
		return
	}
	if workPath != "" {
		work, parseWorkErr := parseWork(workPath)
		if parseWorkErr != nil {
			err = errors.Warning("forg: new module failed").
				WithCause(parseWorkErr)
			return
		}
		dir := filepath.Dir(path)
		for _, use := range work.Uses {
			if use.Dir == dir {
				v = use
				break
			}
		}
	} else {
		v = &Module{
			Dir:      filepath.ToSlash(filepath.Dir(path)),
			Path:     "",
			Version:  "",
			Requires: nil,
			Work:     nil,
			Replace:  nil,
			locker:   &sync.Mutex{},
			parsed:   false,
			services: nil,
			types:    nil,
		}
	}
	return
}

// Module parse(extra replaces)，用extra去替换requires的replace，如果是in workspace，则替换时不考虑version
type Module struct {
	Dir      string
	Path     string
	Version  string
	Requires []*Module
	Work     *Work
	Replace  *Module
	locker   sync.Locker
	parsed   bool
	services map[string]*Service
	types    *Types
}

func (mod *Module) parse() (err error) {
	mod.locker.Lock()
	defer mod.locker.Unlock()
	if mod.parsed {
		return
	}
	if mod.Replace != nil {
		err = mod.Replace.parse()
		if err != nil {
			return
		}
		mod.parsed = true
		return
	}

	modFilepath := filepath.ToSlash(filepath.Join(mod.Dir, "mod.go"))
	if !files.ExistFile(modFilepath) {
		err = errors.Warning("forg: parse mod failed").
			WithCause(errors.Warning("forg: mod file was not found").
				WithMeta("file", modFilepath))
		return
	}
	modData, readModErr := os.ReadFile(modFilepath)
	if readModErr != nil {
		err = errors.Warning("forg: parse mod failed").
			WithCause(errors.Warning("forg: read mod file failed").
				WithCause(readModErr).
				WithMeta("file", modFilepath))
		return
	}
	mf, parseModErr := modfile.Parse(modFilepath, modData, nil)
	if parseModErr != nil {
		err = errors.Warning("forg: parse mod failed").
			WithCause(errors.Warning("forg: parse mod file failed").WithCause(parseModErr).WithMeta("file", modFilepath))
		return
	}
	mod.Path = mf.Module.Mod.Path
	mod.Version = mf.Module.Mod.Version
	mod.Requires = make([]*Module, 0, 1)
	if mf.Require != nil && len(mf.Require) > 0 {
		for _, require := range mf.Require {
			if mod.Work != nil {
				use, used := mod.Work.Use(require.Mod.Path)
				if used {
					mod.Requires = append(mod.Requires, use)
					continue
				}
			}
			requireDir := filepath.ToSlash(filepath.Join(PKG(), fmt.Sprintf("%s@%s", require.Mod.Path, require.Mod.Version)))
			if !files.ExistFile(requireDir) {
				err = errors.Warning("forg: parse mod failed").WithMeta("mod", mod.Path).
					WithCause(errors.Warning("forg: require dir was not found").WithMeta("path", require.Mod.Path).WithMeta("version", require.Mod.Version))
				return
			}
			mod.Requires = append(mod.Requires, &Module{
				Dir:      requireDir,
				Path:     require.Mod.Path,
				Version:  require.Mod.Version,
				Requires: nil,
				Work:     nil,
				Replace:  nil,
				locker:   &sync.Mutex{},
				parsed:   false,
				services: nil,
				types:    nil,
			})
		}
	}
	if mf.Replace != nil && len(mf.Replace) > 0 {
		for _, replace := range mf.Replace {
			replaceDir := ""
			if replace.New.Version != "" {
				replaceDir = filepath.Join(PKG(), fmt.Sprintf("%s@%s", replace.New.Path, replace.New.Version))
			} else {
				replaceDir = filepath.Join(PKG(), replace.New.Path)
			}
			replaceDir = filepath.ToSlash(replaceDir)
			if !files.ExistFile(replaceDir) {
				err = errors.Warning("forg: parse mod failed").WithMeta("mod", mod.Path).
					WithCause(errors.Warning("forg: replace dir was not found").WithMeta("replace", replaceDir))
				return
			}
			replaceFile := filepath.ToSlash(filepath.Join(replaceDir, "mod.go"))
			if !files.ExistFile(replaceFile) {
				err = errors.Warning("forg: parse mod failed").WithMeta("mod", mod.Path).
					WithCause(errors.Warning("forg: replace mod file was not found").
						WithMeta("replace", replaceFile))
				return
			}
			replaceData, readReplaceErr := os.ReadFile(replaceFile)
			if readReplaceErr != nil {
				err = errors.Warning("forg: parse mod failed").WithMeta("mod", mod.Path).
					WithCause(errors.Warning("forg: read replace mod file failed").WithCause(readReplaceErr).WithMeta("replace", replaceFile))
				return
			}
			rmf, parseReplaceModErr := modfile.Parse(replaceFile, replaceData, nil)
			if parseReplaceModErr != nil {
				err = errors.Warning("forg: parse mod failed").WithMeta("mod", mod.Path).
					WithCause(errors.Warning("forg: parse replace mod file failed").WithCause(parseReplaceModErr).WithMeta("replace", replaceFile))
				return
			}
			for _, require := range mod.Requires {
				if require.Path == replace.Old.Path && require.Version == replace.Old.Version {
					require.Replace = &Module{
						Dir:      replaceDir,
						Path:     rmf.Module.Mod.Path,
						Version:  rmf.Module.Mod.Version,
						Requires: nil,
						Work:     nil,
						Replace:  nil,
						locker:   &sync.Mutex{},
						parsed:   false,
						services: nil,
						types:    nil,
					}
				}
			}
		}
	}
	if mod.Work != nil && len(mod.Work.Replaces) > 0 && len(mod.Requires) > 0 {
		for i, require := range mod.Requires {
			if require.Work != nil || require.Replace != nil {
				continue
			}
			for _, replace := range mod.Work.Replaces {
				if require.Path == replace.Path {
					mod.Requires[i] = replace
					break
				}
			}
		}
	}

	mod.types = &Types{
		values: sync.Map{},
		group:  singleflight.Group{},
		mod:    mod,
	}
	mod.parsed = true
	return
}

func (mod *Module) Services() (services Services, err error) {
	mod.locker.Lock()
	defer mod.locker.Unlock()
	if mod.services != nil {
		services = make([]*Service, 0, 1)
		for _, service := range mod.services {
			services = append(services, service)
		}
		sort.Sort(services)
		return
	}
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
	services = make([]*Service, 0, 1)
	for _, service := range mod.services {
		services = append(services, service)
	}
	sort.Sort(services)
	return
}

func (mod *Module) findRequire(ctx context.Context, path string) (require *Module, has bool) {
	// todo
	return
}

func (mod *Module) findFile(ctx context.Context, path string, match func(file *ast.File) (ok bool)) (file *ast.File, err error) {
	// todo
	return
}

func (mod *Module) String() (s string) {
	buf := bytes.NewBuffer([]byte{})
	_, _ = buf.WriteString(fmt.Sprintf("path: %s\n", mod.Path))
	_, _ = buf.WriteString(fmt.Sprintf("version: %s\n", mod.Version))
	for _, require := range mod.Requires {
		_, _ = buf.WriteString(fmt.Sprintf("requre: %s@%s", require.Path, require.Version))
		if require.Replace != nil {
			_, _ = buf.WriteString(fmt.Sprintf("=> %s", require.Replace.Path))
			if require.Replace.Version != "" {
				_, _ = buf.WriteString(fmt.Sprintf("@%s", require.Replace.Version))
			}
		}
		_, _ = buf.WriteString("\n")
	}
	services, servicesErr := mod.Services()
	if servicesErr != nil {
		_, _ = buf.WriteString("service: load failed\n")
		_, _ = buf.WriteString(fmt.Sprintf("%+v", servicesErr))

	} else {
		for _, service := range services {
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
	}
	s = buf.String()
	return
}
