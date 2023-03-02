package module_test

import (
	"fmt"
	"golang.org/x/mod/modfile"
	"os"
	"testing"
)

func TestWork(t *testing.T) {
	x := `D:\studio\workspace\go\src\tkh.com\go.work`
	data, readErr := os.ReadFile(x)
	if readErr != nil {
		t.Error(readErr)
		return
	}
	file, parseErr := modfile.ParseWork(x, data, nil)
	if parseErr != nil {
		t.Error(parseErr)
		return
	}
	fmt.Println(file.Go.Version)
	fmt.Println(file.Use)
	for _, use := range file.Use {
		fmt.Println("use:", use.Path, use.ModulePath)
	}
	for _, replace := range file.Replace {
		fmt.Println("replace", replace.New.Path, replace.Old.Path)
	}

}
