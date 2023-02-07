package forg

import "time"

type Process interface {
	Steps() (steps []Step)
	Start() (stepNoCh chan int)
	Abort(timeout time.Duration) (err error)
}

type Step interface {
	Name() (name string)
	Description() (description string)
}
