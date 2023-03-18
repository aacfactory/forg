package module_test

import (
	"fmt"
	"github.com/aacfactory/forg/module"
	"testing"
)

func TestLatestVersion(t *testing.T) {
	v, err := module.LatestVersion("github.com/aacfactory/forg")
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	fmt.Println(v)
}

func TestLatestVersionFromProxy(t *testing.T) {
	v, err := module.LatestVersionFromProxy("github.com/aacfactory/forg")
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	fmt.Println(v)
}
