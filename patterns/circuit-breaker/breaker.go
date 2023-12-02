package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"time"
)

type Circuit[T any] interface {
	Run(context.Context) (T, error)
}

type CircuitFunc[T any] func(context.Context) (T, error)

func (c CircuitFunc[T]) Run(ctx context.Context) (T, error) {
	return c(ctx)
}

func Breaker[T any](circuit Circuit[T], failureThreshold uint) Circuit[T] {
	consecutiveFailures := 0
	lastAttempt := time.Now()
	var m sync.RWMutex

	return CircuitFunc[T](func(ctx context.Context) (T, error) {
		m.RLock()
		d := consecutiveFailures - int(failureThreshold)
		if d >= 0 {
			shouldRetryAt := lastAttempt.Add(2 << d)
			if !time.Now().After(shouldRetryAt) {
				m.RUnlock()
				var empty T
				return empty, errors.New("service unreachable")
			}
		}

		m.RUnlock()

		res, err := circuit.Run(ctx)

		m.Lock()
		defer m.Unlock()

		lastAttempt = time.Now()

		if err != nil {
			consecutiveFailures++
			return res, err
		}

		consecutiveFailures = 0

		return res, nil
	})
}
