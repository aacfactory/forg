package forg_test

import (
	"context"
	"fmt"
	"github.com/aacfactory/forg"
	"testing"
)

func TestNew(t *testing.T) {
	p, pErr := forg.Load(`D:\studio\workspace\go\src\github.com\aacfactory\fns-example\standalone`)
	if pErr != nil {
		t.Errorf("%+v", pErr)
		return
	}
	process, codingErr := p.Coding(context.TODO())
	if codingErr != nil {
		t.Errorf("%+v", codingErr)
		return
	}
	results := process.Start(context.TODO())
	for {
		result, ok := <-results
		if !ok {
			break
		}
		fmt.Println(result.String())
	}
}
