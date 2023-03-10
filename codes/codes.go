package codes

import (
	"context"
	"github.com/aacfactory/forg/processes"
)

type CodeFile interface {
	Name() (name string)
	Write(ctx context.Context) (err error)
}

func Unit(file CodeFile) (unit processes.Unit) {
	return func(ctx context.Context) (result interface{}, err error) {
		err = file.Write(ctx)
		if err != nil {
			return
		}
		result = file.Name()
		return
	}
}
