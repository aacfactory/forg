package codes

import (
	"os"
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
