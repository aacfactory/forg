package module_test

import (
	"fmt"
	"github.com/aacfactory/forg/module"
	"testing"
)

func TestParseModule(t *testing.T) {
	path := "D:\\studio\\workspace\\go\\src\\tkh.com\\tkh\\go.mod"
	mod, createErr := module.New(path)
	if createErr != nil {
		t.Errorf("%+v", createErr)
		return
	}
	fmt.Println(mod)

}
