package forg_test

import (
	"context"
	"fmt"
	"github.com/aacfactory/forg"
	"testing"
)

func TestNew(t *testing.T) {
	p, pErr := forg.New(`D:\studio\workspace\go\src\github.com\aacfactory\fns-example\standalone\go.mod`)
	if pErr != nil {
		t.Errorf("%+v", pErr)
		return
	}
	results := p.Start(context.TODO())
	for {
		result, ok := <-results
		if !ok {
			break
		}
		fmt.Println(result.String())
	}
}
