package handler

import (
	"context"
	"sync"
)

// runtimeLock serializes runtime changes while allowing queued requests to cancel.
// Its zero value is ready to use.
type runtimeLock struct {
	once sync.Once
	held chan struct{}
}

func (m *runtimeLock) LockContext(ctx context.Context) error {
	m.once.Do(func() { m.held = make(chan struct{}, 1) })
	select {
	case <-ctx.Done():
		return ctx.Err()
	case m.held <- struct{}{}:
		if err := ctx.Err(); err != nil {
			m.Unlock()
			return err
		}
		return nil
	}
}

func (m *runtimeLock) Lock()   { _ = m.LockContext(context.Background()) }
func (m *runtimeLock) Unlock() { <-m.held }
