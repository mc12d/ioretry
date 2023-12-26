package ioretry

import (
	"os"
	"time"
)

type Config struct {
	n int
	d time.Duration

	signalContinueOnError bool
	signals               []os.Signal

	recoverPanic         bool
	panicContinueOnError bool

	repeat bool
}

const (
	Forever  int           = -1
	OutATime time.Duration = 0
)

type Option func(*Config)

var (
    // default config does effectively nothing
	DefaultConfig = func() Config {
		config := Config{}

		OptTimeout(OutATime)(&config)
		OptRecoverPanic(false)(&config)

		return config
	}()
)

func NewConfig(opts ...Option) *Config {
	config := DefaultConfig
	for _, opt := range opts {
		opt(&config)
	}
	return &config
}

// OptRetry
// - use ioretry.Forever for infinite retries
// - use ioretry.OutATime for no time limit
// OptRetry, OptRepeat and OptTimeout are mutually exclusive
func OptRetry(n int, period time.Duration) Option {
	return func(config *Config) {
		config.n = n
		config.d = period
		config.repeat = false
	}
}

// OptRepeat similar to OptRetry, but continues repeating whenever underlying IO fails or succeeds
// OptRetry, OptRepeat and OptTimeout are mutually exclusive
func OptRepeat(n int, period time.Duration) Option {
	return func(config *Config) {
		config.n = n
		config.d = period
		config.repeat = true
	}
}

// OptTimeout just a timeout
// OptRetry, OptRepeat and OptTimeout are mutually exclusive
func OptTimeout(t time.Duration) Option {
	return OptRetry(1, t)
}

// OptRecoverPanic recovers panic and returns ioretry.PanicError
// limitation - does not recover panics in foreign goroutines, like
//
//		f := func(ctx context.Context) error {
//			   ch := make(chan error, 1)
//			   go func() {
//					err := errors.New("I will not be recovered")
//				    defer ch <- err
//			        panic(err)
//			   }
//	           return <-ch
//		}
func OptRecoverPanic(recover bool) Option {
	return func(config *Config) {
		config.recoverPanic = recover
		config.panicContinueOnError = true
	}
}

// OptRecoverPanicAndStopTrying similar to OptRecoverPanic, but discards remaining retries
func OptRecoverPanicAndStopTrying(recover bool) Option {
	return func(config *Config) {
		config.recoverPanic = recover
		config.panicContinueOnError = false
	}
}
