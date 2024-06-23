package task

import (
	"context"
	"go.uber.org/zap"
	"sync"
)

type Worker interface {
	DoWork(ctx context.Context)
}

type WorkerFunc func(ctx context.Context)

func (w WorkerFunc) DoWork(ctx context.Context) {
	w(ctx)
}

type Pool struct {
	work   chan Worker
	wg     sync.WaitGroup
	cancel context.CancelFunc
	logger *zap.Logger
}

func NewPool(poolSize int, logger *zap.Logger) *Pool {
	t := &Pool{
		work:   make(chan Worker),
		logger: logger,
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel

	t.wg.Add(poolSize)
	for i := 0; i < poolSize; i++ {
		go func() {
			defer t.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case w, ok := <-t.work:
					if !ok {
						continue
					}
					w.DoWork(ctx)
				}
			}
		}()
	}

	return t
}

func (p *Pool) Add(w Worker) {
	p.work <- w
}

func (p *Pool) AddFunc(f func(ctx context.Context)) {
	p.Add(WorkerFunc(f))
}

func (p *Pool) Shutdown() {
	p.cancel()
	close(p.work)
	p.wg.Wait()
}
