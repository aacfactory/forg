package codes

import (
	"context"
	"github.com/aacfactory/gcg"
)

type Coder interface {
	Write(ctx context.Context) (code gcg.Code, err error)
}
