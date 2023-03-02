package module_test

import (
	"fmt"
	"github.com/aacfactory/forg/module"
	"testing"
)

func TestMergeImports(t *testing.T) {
	i1 := module.Imports{}
	i1.Add(&module.Import{
		Path:  "encoding/json",
		Alias: "",
	})
	i1.Add(&module.Import{
		Path:  "a/a",
		Alias: "",
	})
	i1.Add(&module.Import{
		Path:  "a/b",
		Alias: "",
	})
	i2 := module.Imports{}
	i2.Add(&module.Import{
		Path:  "encoding/json",
		Alias: "stdjson",
	})
	i2.Add(&module.Import{
		Path:  "b/a",
		Alias: "",
	})
	i2.Add(&module.Import{
		Path:  "b/b",
		Alias: "",
	})
	i2.Add(&module.Import{
		Path:  "c/a",
		Alias: "xx",
	})
	v := module.MergeImports([]module.Imports{i1, i2})
	for s, i := range v {
		fmt.Println("m:", s, i.Path, i.Name(), i.Alias)
	}
}
