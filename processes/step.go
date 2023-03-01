package processes

import (
	"context"
)

type Unit func(ctx context.Context) (result interface{}, err error)

func ParallelUnits(units ...Unit) (u Unit) {
	pu := &PUnits{
		units: units,
	}
	u = pu.Execute
	return
}

type PUnits struct {
	units []Unit
}

func (pu *PUnits) Execute(ctx context.Context) (result interface{}, err error) {
	if pu.units == nil || len(pu.units) == 0 {
		return
	}
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	rc := make(chan Result, len(pu.units))
	n := 0
	for _, unit := range pu.units {
		if unit == nil {
			continue
		}
		n++
		go func(ctx context.Context, unit Unit, rc chan Result) {
			if ctx.Err() != nil {
				rc <- Result{
					Succeed: false,
					Error:   ErrAborted.WithCause(ctx.Err()),
				}
				return
			}
			data, failed := unit(ctx)
			defer func() {
				_ = recover()
			}()
			rc <- Result{
				Succeed: failed == nil,
				Result:  data,
				Error:   failed,
			}
		}(ctx, unit, rc)
	}
	rr := make([]interface{}, 0, 1)
	for {
		stop := false
		select {
		case <-ctx.Done():
			err = ErrAborted.WithCause(ctx.Err())
			stop = true
			break
		case r, ok := <-rc:
			if !ok {
				stop = true
				break
			}
			if r.Succeed {
				rr = append(rr, r.Result)
				break
			}
			err = r.Error
			stop = true
			break
		}
		if stop {
			break
		}
		if len(rr) == n {
			break
		}
	}
	cancel()
	close(rc)
	if err == nil {
		result = rr
	}
	return
}

type Step struct {
	No   int
	Name string
	Unit Unit
}
