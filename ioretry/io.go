package ioretry

import (
	"context"
)

func Wrap(r IO, opts ...Option) IO {
	return &io{
		underlying: r,
		options:    opts,
	}
}

type io struct {
	underlying IO

	options []Option
}

func (m *io) Run(ctx context.Context) error {
	return WrapFunc(m.underlying.Run, m.options...)(ctx)
}
