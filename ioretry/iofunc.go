package ioretry

import (
	"context"
	"errors"
	"fmt"
	"time"
)

func MultiFunc(ff ...IOFunc) IOFunc {
	return func(ctx context.Context) error {
		errs := make(chan error, len(ff))
		for _, f := range ff {
			go func(f IOFunc) {
				errs <- f(ctx)
			}(f)
		}

		me := make([]error, 0)
		for i := 0; i < len(ff); i++ {
			select {
			case err := <-errs:
				if err != nil {
					me = append(me, err)
				}
			case <-ctx.Done():
				return MultiFuncError([]error{ctx.Err()})
			}
		}

		if len(me) == 0 {
			return nil
		}
		return MultiFuncError(me)
	}
}

func MultiFuncFailFast(ff ...IOFunc) IOFunc {
	return func(ctx context.Context) error {
		errs := make(chan error, len(ff))
		newCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		for _, f := range ff {
			go func(f IOFunc) {
				errs <- f(newCtx)
			}(f)
		}

		for i := 0; i < len(ff); i++ {
			select {
			case err := <-errs:
				if err != nil {
					return err
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	}
}

func WrapFunc(f IOFunc, opts ...Option) IOFunc {
	config := NewConfig(opts...)
	return func(ctx context.Context) error {
		var (
			continueLoop bool
			err          error
		)
		for i := 0; i < config.n || config.n == Forever; i++ {
			t := time.Now()
			err, continueLoop = handleIteration(ctx, f, config)
			if !continueLoop {
				return err
			}
			time.Sleep(config.d - time.Since(t))
		}
		if err != nil {
			return fmt.Errorf("max retry count reached, underlying error: %w", err)
		}
		return nil
	}
}

// handleIteration performs single IOFunc call and controls execution process as specified by options
func handleIteration(parentCtx context.Context, f IOFunc, config *Config) (err error, continueLoop bool) {
	var (
		recoverCh        chan error
		errCh            = make(chan error, 1)
		childCtx, cancel = func() (context.Context, context.CancelFunc) {
			if config.d == OutATime {
				return context.WithCancel(parentCtx)
			}
			return context.WithTimeout(parentCtx, config.d)
		}()
	)
	defer cancel()

	if config.recoverPanic {
		recoverCh = make(chan error, 1)
	}

	goFuncAndRecover(childCtx, f, errCh, recoverCh)
	select {
	case <-childCtx.Done():
		if parentCtx.Err() != nil {
			return fmt.Errorf("parent context is done prematurely: %w", parentCtx.Err()), false
		}
		if errors.Is(childCtx.Err(), context.DeadlineExceeded) {
			return childCtx.Err(), true
		}
		return
	case err = <-errCh:
		return err, err != nil || config.repeat
	case err = <-recoverCh:
		return PanicError{Err: err}, true
	}
}

func goFuncAndRecover(ctx context.Context, f IOFunc, errCh chan error, recoverCh chan error) {
	go func() {
		if recoverCh != nil {
			defer func() {
				if r := recover(); r != nil {
					recoverCh <- wrapPanicValue(r)
				}
			}()
		}
		errCh <- f(ctx)
	}()
}

func wrapPanicValue(r any) error {
	switch t := r.(type) {
	case error:
		return fmt.Errorf("encountered panic, error: %w", t)
	case string:
		return fmt.Errorf("encoutered panic, message: %s", t)
	default:
		return fmt.Errorf("encoutered panic, related type: %T", t)
	}
}
