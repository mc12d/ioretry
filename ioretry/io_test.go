package ioretry_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/mc12d/ioretry/ioretry"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type ioImpl struct {
	f ioretry.IOFunc
}

func (io *ioImpl) Run(ctx context.Context) error {
	return io.f(ctx)
}

func TestMultiIO(t *testing.T) {
	// given
	var (
		io1Err = errors.New("io1")
		io2Err = errors.New("io2")
		io3Err = errors.New("io3")
		io1    = &ioImpl{func(ctx context.Context) error {
			return SleepContextError(ctx, 100*time.Millisecond, io1Err)
		}}
		io2 = &ioImpl{func(ctx context.Context) error {
			return SleepContextError(ctx, 200*time.Millisecond, io2Err)
		}}
		io3 = &ioImpl{func(ctx context.Context) error {
			return SleepContextError(ctx, 300*time.Millisecond, io3Err)
		}}
		io = ioretry.Multi(io1, io2, io3)
	)
	// when
	var (
		ctx, cancel = context.WithTimeout(context.Background(), 250*time.Millisecond)
		err         = io.Run(ctx)
	)
	defer cancel()

	// then
	merr, merrOK := err.(ioretry.MultiResourceError)
	require.True(t, merrOK, fmt.Sprintf("actual err: %T, errMsg: %s", err, err.Error()))
	require.Equal(t, 3, len(merr))

	require.True(t, errors.Is(merr[io1], io1Err))
	require.True(t, errors.Is(merr[io2], io2Err))
	require.True(t, errors.Is(merr[io3], context.DeadlineExceeded))

}

func TestWrapIO(t *testing.T) {
	// given
	var (
		ctx                 = context.Background()
		ctxToCancel, cancel = context.WithCancel(ctx)

		io1Err = errors.New("io1")
		io2Err = errors.New("io2")
		io1    = &ioImpl{func(ctx context.Context) error {
			e := SleepContext(ctx, 120*time.Millisecond)
			if e == nil {
				return io1Err
			}
			return e
		}}
		io2 = &ioImpl{func(ctx context.Context) error {
			e := SleepContext(ctx, 500*time.Millisecond)
			if e == nil {
				return io2Err
			}
			return e
		}}
		// io1 will drain all retries earlier, so io2 would not succeed
		ios = ioretry.MultiEager(
			ioretry.Wrap(io1, ioretry.OptRetry(3, 100*time.Millisecond)),
			ioretry.Wrap(io2, ioretry.OptTimeout(600*time.Millisecond)),
		)
	)
	defer cancel()

	// when
	err := ios.Run(ctxToCancel)

	// then
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
