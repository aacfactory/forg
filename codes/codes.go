package codes

import (
	"context"
	"github.com/aacfactory/forg/processes"
)

type CodeFile interface {
	Name() (name string)
	Write(ctx context.Context) (err error)
}

type CodeFileUnit struct {
	cf CodeFile
}

func (unit *CodeFileUnit) Handle(ctx context.Context) (result interface{}, err error) {
	err = unit.cf.Write(ctx)
	if err != nil {
		return
	}
	result = unit.cf.Name()
	return
}

func Unit(file CodeFile) (unit processes.Unit) {
	return &CodeFileUnit{
		cf: file,
	}
}
