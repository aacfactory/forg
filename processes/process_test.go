package processes_test

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/forg/processes"
	"testing"
	"time"
)

type WorkUnit struct {
	name  string
	no    int
	value int
}

func (unit *WorkUnit) Execute(ctx context.Context) (message interface{}, err error) {
	if ctx.Err() != nil {
		err = ctx.Err()
		return
	}
	if unit.no%2 == 0 {
		err = errors.ServiceError("failed")
	} else {
		unit.value = unit.no
		message = unit.name
	}
	time.Sleep(1 * time.Second)
	return
}

func TestNew(t *testing.T) {
	units := make([]*WorkUnit, 0, 1)
	process := processes.New()
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("s%d", i)
		unit := &WorkUnit{
			name: name,
			no:   i,
		}
		units = append(units, unit)
		process.Add(name, unit.Execute)
	}

	results := process.Start(context.TODO())
	go func(process *processes.Process) {
		time.Sleep(3100 * time.Millisecond)
		fmt.Println("abort:", process.Abort(2*time.Second))
	}(process)
	for {
		result, ok := <-results
		if !ok {
			break
		}
		fmt.Println("result:", result.No, result.Name, result.Succeed, result.Result, result.Error)
	}
	for _, unit := range units {
		fmt.Println("unit:", unit.no, unit.value)
	}
}

func TestParallelUnits(t *testing.T) {
	process := processes.New()
	for i := 0; i < 2; i++ {
		subs := make([]processes.Unit, 0, 1)
		for j := 0; j < 10; j++ {
			unit := &WorkUnit{
				name: fmt.Sprintf("s:%d:%d", i, j),
				no:   i,
			}
			subs = append(subs, unit.Execute)
		}
		process.Add(fmt.Sprintf("s:%d", i), processes.ParallelUnits(subs...))
	}
	results := process.Start(context.TODO())
	go func(process *processes.Process) {
		time.Sleep(2000 * time.Millisecond)
		fmt.Println("abort:", process.Abort(2*time.Second))
	}(process)
	for {
		result, ok := <-results
		if !ok {
			break
		}
		fmt.Println("result:", result.No, result.Name, result.Succeed, result.Result, result.Error)
	}
}
