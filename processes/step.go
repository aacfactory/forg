package processes

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"sync"
)

type Unit func(ctx context.Context) (result interface{}, err error)

type Result struct {
	StepNo   int64
	StepNum  int64
	StepName string
	UnitNo   int64
	UnitNum  int64
	Data     interface{}
	Error    error
}

func (result Result) String() string {
	status := "√"
	if result.Error != nil {
		if errors.Map(result.Error).Contains(ErrAborted) {
			status = "aborted"
		} else {
			status = "x"
		}
	}
	return fmt.Sprintf("[%d/%d] %s [%d/%d] %s", result.StepNo, result.StepNum, result.StepName, result.UnitNo, result.UnitNum, status)
}

type Step struct {
	no       int64
	name     string
	num      int64
	units    []Unit
	resultCh chan<- Result
}

func (step *Step) Execute(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	if step.units == nil || len(step.units) == 0 {
		return
	}
	unitNum := int64(len(step.units))
	wg := sync.WaitGroup{}
	for i, unit := range step.units {
		unitNo := int64(i + 1)
		if unit == nil {
			step.resultCh <- Result{
				StepNo:   step.no,
				StepNum:  step.num,
				StepName: step.name,
				UnitNo:   unitNo,
				UnitNum:  unitNum,
				Data:     nil,
				Error:    errors.Warning("processes: unit is nil").WithMeta("step", step.name),
			}
			continue
		}
		wg.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, unitNo int64, unit Unit, step *Step) {
			defer wg.Done()
			if ctx.Err() != nil {
				return
			}
			data, err := unit(ctx)
			defer func() {
				_ = recover()
			}()
			step.resultCh <- Result{
				StepNo:   step.no,
				StepNum:  step.num,
				StepName: step.name,
				UnitNo:   unitNo,
				UnitNum:  unitNum,
				Data:     data,
				Error:    err,
			}
		}(ctx, &wg, unitNo, unit, step)
	}
	wg.Wait()
}
