package ioretry

import (
	"os"
	"time"
)

type Config struct {
	n int
	d time.Duration

	signals         []os.Signal
	continueOnError bool
	continueOnPanic bool
}

const RetryInfinitely = -1

type Option func(*Config)

var (
	DefaultConfig = func() Config {
		config := Config{}

		OptTimeout(time.Second)(&config)
		OptHandleSignals()(&config)
		OptContinueOnPanic(false)(&config)

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

// OptRetry specify -1 for n = +inf
// OptRetry, OptRepeat and OptTimeout are mutually exclusive
func OptRetry(n int, period time.Duration) Option {
	return func(config *Config) {
		config.n = n
		config.d = period
		config.continueOnError = false
	}
}

// OptRepeat similar to OptRetry, but continues repeating if underlying IO returns no error
// OptRetry, OptRepeat and OptTimeout are mutually exclusive
func OptRepeat(n int, period time.Duration) Option {
	return func(config *Config) {
		config.n = n
		config.d = period
		config.continueOnError = true
	}
}

// OptTimeout just a timeout
// OptRetry, OptRepeat and OptTimeout are mutually exclusive
func OptTimeout(t time.Duration) Option {
	return OptRetry(1, t)
}

// OptHandleSignals will return ioretry.SignalError immediately if encountered any of specified signals
// unlike signal.Notify, empty argument corresponds to no signals being handled
func OptHandleSignals(signals ...os.Signal) Option {
	return func(config *Config) {
		config.signals = signals
	}
}

func OptContinueOnPanic(recover bool) Option {
	return func(config *Config) {
		config.continueOnPanic = recover
	}
}
