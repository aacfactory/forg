package codes

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"golang.org/x/sync/singleflight"
	"sync"
)

type Structs struct {
	mod    *Module
	values sync.Map
	group  singleflight.Group
}

func (sts *Structs) get(ctx context.Context, importer string, name string) (v *Struct, err error) {
	key := fmt.Sprintf("%s.%s", importer, name)
	vv, has := sts.values.Load(key)
	if has {
		v = vv.(*Struct)
		return
	}
	vv = ctx.Value(key)
	if vv != nil {
		v = vv.(*Struct)
		return
	}
	st, readErr, _ := sts.group.Do(key, func() (v interface{}, err error) {
		st := &Struct{
			Importer: importer,
			Name:     name,
		}
		ctx = context.WithValue(ctx, key, st)
		// todo process

		return
	})
	if readErr != nil {
		err = errors.Warning("forg: get struct failed").WithMeta("name", key).WithCause(readErr)
		return
	}
	v = st.(*Struct)
	return
}

type Struct struct {
	Importer string
	Name     string
}

func (st *Struct) Read(ctx context.Context) (result interface{}, err error) {
	// todo get mod from ctx

	// todo get struct file
	return
}
