package ioretry

import (
	"context"
)

func Multi(ios ...IO) IO {
	return &multiIO{
		underlying: ios,
		eager:      false,
	}
}

func MultiFailFast(ios ...IO) IO {
	return &multiIO{
		underlying: ios,
		eager:      true,
	}
}

type multiIO struct {
	underlying []IO
	eager      bool
}

type multiIOResult struct {
	err error
	io  IO
}

func (m *multiIO) Run(ctx context.Context) error {
	errs := make(chan *multiIOResult, len(m.underlying))
	for _, io := range m.underlying {
		go func(io IO) {
			errs <- &multiIOResult{io: io, err: io.Run(ctx)}
		}(io)
	}
	me := make(map[IO]error, 0)
	for i := 0; i < len(m.underlying); i++ {
		select {
		case err := <-errs:
			if err.err != nil && m.eager {
				return err.err
			}
			me[err.io] = err.err
		}
	}
	if len(me) == 0 {
		return nil
	}
	return MultiIOError(me)
}

func (m *multiIO) removeNilValues(errs map[IO]error) {
	for _, k := range m.underlying {
		if v, ok := errs[k]; ok && v == nil {
			delete(errs, k)
		}
	}
}
