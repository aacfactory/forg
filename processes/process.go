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
		steps:    make([]Step, 0, 1),
		closedCh: make(chan struct{}, 1),
		cancel:   nil,
	}
}

type Process struct {
	steps    []Step
	closedCh chan struct{}
	cancel   context.CancelFunc
}

func (p *Process) Add(name string, unit Unit) {
	no := len(p.steps)
	p.steps = append(p.steps, Step{
		No:   no,
		Name: name,
		Unit: unit,
	})
}

func (p *Process) Len() (n int) {
	n = len(p.steps)
	return
}

func (p *Process) Start(ctx context.Context) (result <-chan Result) {
	ctx, p.cancel = context.WithCancel(ctx)
	results := make(chan Result, 512)
	go func(ctx context.Context, p *Process, result chan Result) {
		for _, step := range p.steps {
			stop := false
			select {
			case <-ctx.Done():
				stop = true
				result <- Result{
					No:      step.No,
					Name:    step.Name,
					Succeed: false,
					Result:  nil,
					Error:   ErrAborted.WithCause(ctx.Err()),
				}
				p.closedCh <- struct{}{}
				break
			default:
				data, err := step.Unit(ctx)
				result <- Result{
					No:      step.No,
					Name:    step.Name,
					Succeed: err == nil,
					Result:  data,
					Error:   err,
				}
			}
			if stop {
				break
			}
		}
		close(result)
		close(p.closedCh)
	}(ctx, p, results)
	result = results
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
