package ioretry_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mc12d/ioretry/ioretry"
	"github.com/stretchr/testify/require"
)

func SleepContext(ctx context.Context, d time.Duration) error {
	return SleepContextError(ctx, d, nil)
}

func SleepContextError(ctx context.Context, d time.Duration, err error) error {
	c := make(chan error)
	go func() {
		time.Sleep(d)
		c <- err
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-c:
		return err
	}
}

func TestMultiFunc(t *testing.T) {
	// given
	var (
		f1 = func(ctx context.Context) error {
			return SleepContext(ctx, 100*time.Millisecond)
		}
		f2 = func(ctx context.Context) error {
			return SleepContext(ctx, 200*time.Millisecond)
		}
		f3 = func(ctx context.Context) error {
			return SleepContext(ctx, 300*time.Millisecond)
		}
		f = ioretry.MultiFunc(f1, f2, f3)
	)
	// when
	var (
		ctx, cancel = context.WithTimeout(context.Background(), 250*time.Millisecond)
		err         = f(ctx)
	)
	defer cancel()

	// then
	merr, merrOK := err.(ioretry.MultiFuncError)
	require.True(t, merrOK, fmt.Sprintf("actual err: %T, errMsg: %s", err, err.Error()))
	require.Equal(t, 1, len(merr))
	require.True(t, errors.Is(merr[0], context.DeadlineExceeded))

}

func TestMultiFuncEager(t *testing.T) {
	// given
	var (
		f1Err = errors.New("f1 err")
		f2Err = errors.New("f2 err")
		f3Err = errors.New("f3 err")

		f1 = func(ctx context.Context) error {
			return SleepContextError(ctx, 100*time.Millisecond, f1Err)
		}
		f2 = func(ctx context.Context) error {
			return SleepContextError(ctx, 200*time.Millisecond, f2Err)
		}
		f3 = func(ctx context.Context) error {
			return SleepContextError(ctx, 300*time.Millisecond, f3Err)
		}
		f = ioretry.MultiFuncFailFast(f1, f2, f3)
	)
	// when
	var (
		ctx, cancel = context.WithTimeout(context.Background(), 150*time.Millisecond)
		err         = f(ctx)
	)
	defer cancel()

	// then
	require.True(t, errors.Is(err, f1Err), fmt.Sprintf("actual err: %T, errMsg: %s", err, err.Error()))

}

func TestWrapFunc(t *testing.T) {
	// given
	var (
		ctx                      = context.Background()
		ctxToCancel, cancel      = context.WithCancel(ctx)
		retrySucceededAt3Counter = int32(0)

		timeoutFailedF     = func(ctx context.Context) error { return SleepContext(ctx, 200*time.Millisecond) }
		timeoutSucceededF  = func(ctx context.Context) error { return SleepContext(ctx, 10*time.Millisecond) }
		retryFailedF       = timeoutFailedF
		retrySucceededF    = timeoutSucceededF
		retrySucceededAt3F = func(ctx context.Context) error {
			if atomic.LoadInt32(&retrySucceededAt3Counter) == 2 {
				return SleepContext(ctx, 10*time.Millisecond)
			}
			atomic.AddInt32(&retrySucceededAt3Counter, 1)
			return SleepContext(ctx, 200*time.Millisecond)

		}
		retryCancelledF = timeoutFailedF
		panicF          = func(ctx context.Context) error { panic(errors.New("panicking ✋")) }
	)
	var (
		timeoutFailedFF     = ioretry.WrapFunc(timeoutFailedF, ioretry.OptTimeout(100*time.Millisecond))
		timeoutSucceededFF  = ioretry.WrapFunc(timeoutSucceededF, ioretry.OptTimeout(100*time.Millisecond))
		retryFailedFF       = ioretry.WrapFunc(retryFailedF, ioretry.OptRetry(2, 100*time.Millisecond))
		retrySucceededFF    = ioretry.WrapFunc(retrySucceededF, ioretry.OptRetry(2, 100*time.Millisecond))
		retrySucceededAt3FF = ioretry.WrapFunc(retrySucceededAt3F, ioretry.OptRetry(4, 100*time.Millisecond))
		retryCancelledFF    = ioretry.WrapFunc(retryCancelledF, ioretry.OptRetry(4, 100*time.Millisecond))
		panicFF             = ioretry.WrapFunc(panicF, ioretry.OptRecoverPanic(true))
	)

	// when
	var (
		errTimeoutFailed     = make(chan error, 1)
		errTimeoutSucceeded  = make(chan error, 1)
		errRetryFailed       = make(chan error, 1)
		errRetrySucceeded    = make(chan error, 1)
		errRetrySucceededAt3 = make(chan error, 1)
		errRetryCancelled    = make(chan error, 1)
		errPanic             = make(chan error, 1)
	)
	goCh(ctx, timeoutFailedFF, errTimeoutFailed)
	goCh(ctx, timeoutSucceededFF, errTimeoutSucceeded)
	goCh(ctx, retryFailedFF, errRetryFailed)
	goCh(ctx, retrySucceededFF, errRetrySucceeded)
	goCh(ctx, retrySucceededAt3FF, errRetrySucceededAt3)
	goCh(ctxToCancel, retryCancelledFF, errRetryCancelled)
	goCh(ctx, panicFF, errPanic)
	go time.AfterFunc(250*time.Millisecond, func() {
		cancel()
	})

	// then
	require.ErrorIs(t, <-errTimeoutFailed, context.DeadlineExceeded)
	require.ErrorIs(t, <-errRetryFailed, context.DeadlineExceeded)
	require.Nil(t, <-errRetrySucceededAt3)
	require.Equal(t, int32(2), retrySucceededAt3Counter)
	require.Nil(t, <-errTimeoutSucceeded)
	require.Nil(t, <-errRetrySucceeded)
	require.ErrorIs(t, <-errRetryCancelled, context.Canceled)
}

func TestMultiWrapFunc(t *testing.T) {
	// given
	var (
		ctx                 = context.Background()
		ctxToCancel, cancel = context.WithCancel(ctx)

		shortFuncErr = errors.New("short func")
		longgFuncErr = errors.New("longg func")
		shortFunc    = func(ctx context.Context) error {
			e := SleepContext(ctx, 200*time.Millisecond)
			if e == nil {
				return shortFuncErr
			}
			return e
		}
		longgFunc = func(ctx context.Context) error {
			e := SleepContext(ctx, 200*time.Millisecond)
			if e == nil {
				return longgFuncErr
			}
			return e
		}

		iofuncs = ioretry.MultiFunc(
			ioretry.WrapFunc(shortFunc, ioretry.OptRetry(10, 180*time.Millisecond)),
			ioretry.WrapFunc(longgFunc, ioretry.OptRetry(5, 300*time.Millisecond)),
		)
	)
	defer cancel()

	// when
	err := iofuncs(ctxToCancel)

	// then
	require.ErrorAs(t, err, &longgFuncErr)
	require.ErrorAs(t, err, &context.DeadlineExceeded)
}

func goCh(ctx context.Context, f ioretry.IOFunc, ch chan error) {
	go func() {
		ch <- f(ctx)
	}()
}
