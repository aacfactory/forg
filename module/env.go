package module

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/forg/files"
	"os"
	"path/filepath"
	"strings"
)

func GOPATH() (gopath string, has bool) {
	gopath, has = os.LookupEnv("GOPATH")
	if has {
		gopath = strings.TrimSpace(gopath)
		has = gopath != ""
	}
	return
}

func GOROOT() (goroot string, has bool) {
	goroot, has = os.LookupEnv("GOROOT")
	if has {
		goroot = strings.TrimSpace(goroot)
		has = goroot != ""
	}
	return
}

var pkgDir = ""

func initPkgDir() (err error) {
	gopath, hasGOPATH := GOPATH()
	if hasGOPATH {
		pkgDir = filepath.ToSlash(filepath.Join(gopath, "pkg/mod"))
		if !files.ExistFile(pkgDir) {
			pkgDir = ""
			err = errors.Warning("forg: GOPATH was found but no 'pkg/mod' dir")
			return
		}
		return
	}
	goroot, hasGOROOT := GOROOT()
	if hasGOROOT {
		pkgDir = filepath.ToSlash(filepath.Join(goroot, "pkg/mod"))
		if !files.ExistFile(pkgDir) {
			pkgDir = ""
			err = errors.Warning("forg: GOROOT was found but no 'pkg/mod' dir")
			return
		}
		return
	}
	if !hasGOPATH && !hasGOROOT {
		err = errors.Warning("forg: GOPATH and GOROOT were not found")
		return
	}
	return
}

func PKG() (pkg string) {
	pkg = pkgDir
	return
}
