package processes

import (
	"context"
	"github.com/aacfactory/errors"
	"time"
)

var (
	ErrAborted = errors.Warning("forg: abort")
)

func New() *Process {
	return &Process{
		units:    0,
		steps:    make([]*Step, 0, 1),
		resultCh: make(chan Result, 512),
		closedCh: make(chan struct{}, 1),
		cancel:   nil,
	}
}

type ProcessController interface {
	Steps() (n int64)
	Units() (n int64)
	Start(ctx context.Context) (results <-chan Result)
	Abort(timeout time.Duration) (err error)
}

type Process struct {
	units    int64
	steps    []*Step
	resultCh chan Result
	closedCh chan struct{}
	cancel   context.CancelFunc
}

func (p *Process) Add(name string, units ...Unit) {
	no := int64(len(p.steps) + 1)
	p.steps = append(p.steps, &Step{
		no:       no,
		name:     name,
		num:      0,
		units:    units,
		resultCh: p.resultCh,
	})
	for _, step := range p.steps {
		step.num = no
	}
	if units != nil {
		p.units = p.units + int64(len(units))
	}
}

func (p *Process) Steps() (n int64) {
	n = int64(len(p.steps))
	return
}

func (p *Process) Units() (n int64) {
	n = p.units
	return
}

func (p *Process) Start(ctx context.Context) (results <-chan Result) {
	ctx, p.cancel = context.WithCancel(ctx)
	go func(ctx context.Context, p *Process, result chan Result) {
		for _, step := range p.steps {
			stop := false
			select {
			case <-ctx.Done():
				result <- Result{
					StepNo:   0,
					StepNum:  0,
					StepName: "",
					UnitNo:   0,
					UnitNum:  0,
					Data:     nil,
					Error:    ErrAborted.WithCause(ctx.Err()),
				}
				stop = true
				p.closedCh <- struct{}{}
				break
			default:
				err := step.Execute(ctx)
				if err != nil {
					stop = true
					p.closedCh <- struct{}{}
				}
				break
			}
			if stop {
				break
			}
		}
		close(result)
		close(p.closedCh)
	}(ctx, p, p.resultCh)
	results = p.resultCh
	return
}

func (p *Process) Abort(timeout time.Duration) (err error) {
	if p.cancel == nil {
		return
	}
	p.cancel()
	select {
	case <-time.After(timeout):
		err = errors.Timeout("forg: abort timeout")
		break
	case <-p.closedCh:
		break
	}
	return
}

func IsAbortErr(err error) (ok bool) {
	ok = errors.Map(err).Contains(ErrAborted) || errors.Map(err).Contains(context.Canceled)
	return
}
