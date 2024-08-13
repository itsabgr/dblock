package cancelable

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
)

type Cancelable struct {
	canceled atomic.Bool
	cancel   context.CancelCauseFunc
	ctx      context.Context
}

func (c *Cancelable) Ctx() context.Context {
	return c.ctx
}

func (c *Cancelable) Cancel(err error) bool {
	if c.canceled.CompareAndSwap(false, true) {
		if err == nil {
			err = errors.New("canceled")
		}
		c.cancel(err)
		return true
	}
	return false
}

func (c *Cancelable) Canceled() bool {
	return c.canceled.Load()
}

func New(pCtx context.Context, fn func(canc *Cancelable)) *Cancelable {
	if err := pCtx.Err(); err != nil {
		panic(err)
	}
	ctx, cancel := context.WithCancelCause(pCtx)
	canc := &Cancelable{
		cancel: cancel,
		ctx:    ctx,
	}
	go fn(canc)
	runtime.Gosched()
	return canc
}
