package module

import (
	"fmt"
	"strings"
)

// Imports 一个fn文件一个，所以key不会重复，
type Imports map[string]*Import

func (s Imports) Find(ident string) (v *Import, has bool) {
	v, has = s[ident]
	return
}

func (s Imports) Path(path string) (v *Import, has bool) {
	for _, i := range s {
		if i.Path == path {
			v = i
			has = true
			return
		}
	}
	return
}

func (s Imports) Len() (n int) {
	n = len(s)
	return
}

func (s Imports) Add(i *Import) {
	_, has := s.Find(i.Ident())
	if !has {
		s[i.Ident()] = i
		return
	}
	return
}

type Import struct {
	Path  string
	Alias string
}

func (i *Import) Ident() (ident string) {
	if i.Alias != "" {
		ident = i.Alias
		return
	}
	ident = i.Name()
	return
}

func (i *Import) Name() (name string) {
	idx := strings.LastIndex(i.Path, "/")
	if idx < 0 {
		name = i.Path
	} else {
		name = i.Path[idx+1:]
	}
	return
}

// MergeImports 在service里增加fn的imports用
func MergeImports(ss []Imports) (v Imports) {
	idents := make(map[string]int)
	v = make(map[string]*Import)
	for _, s := range ss {
		for _, i := range s {
			_, has := v.Path(i.Path)
			if has {
				continue
			}
			vv := &Import{
				Path:  i.Path,
				Alias: "",
			}
			_, hasIdent := v.Find(vv.Name())
			if hasIdent {
				times, hasIdents := idents[vv.Ident()]
				if hasIdents {
					times++
				}
				vv.Alias = fmt.Sprintf("%s%d", vv.Name(), times)
				idents[vv.Name()] = times
			}
			v.Add(vv)
		}
	}
	return
}
