package module

import "context"

type File struct {
	filename string
}

func (f *File) Scan(ctx context.Context) (result interface{}, err error) {

	return
}
