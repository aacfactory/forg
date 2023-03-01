package codes_test

import (
	"fmt"
	"golang.org/x/mod/modfile"
	"os"
	"path/filepath"
	"testing"
)

func TestParseModule(t *testing.T) {
	path := "D:\\studio\\workspace\\go\\src\\github.com\\aacfactory\\forg\\go.mod"
	path = filepath.ToSlash(path)
	fmt.Println(path)
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Error("read:", readErr)
		return
	}
	f, parseErr := modfile.Parse(path, data, nil)
	if parseErr != nil {
		t.Error("parse:", parseErr)
		return
	}
	fmt.Println(f.Module.Mod.Path, f.Module.Mod.Version)
	fmt.Println(f.Go.Version)
	if f.Require != nil {
		for _, require := range f.Require {
			fmt.Println(require.Mod, require.Indirect)
		}
	}
	if f.Replace != nil {
		for _, replace := range f.Replace {
			fmt.Println(replace, replace.New.Path, replace.Old.Path)
		}
	}
	fmt.Println("---")
	fmt.Println(filepath.ToSlash(filepath.Dir(path)))

	fmt.Println(filepath.ToSlash(filepath.Join(filepath.Dir(path), "errors")))

}
