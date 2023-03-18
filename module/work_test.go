package module_test

import (
	"github.com/aacfactory/forg/module"
	"testing"
)

func TestWork(t *testing.T) {
	work := &module.Work{
		Filename: `D:\studio\workspace\go\src\tkh.com\go.work`,
		Uses:     nil,
		Replaces: nil,
	}
	parseErr := work.Parse()
	if parseErr != nil {
		t.Errorf("%+v", parseErr)
		return
	}
}
