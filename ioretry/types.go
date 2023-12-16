package ioretry

import "context"

type IOFunc func(ctx context.Context) error

type IO interface {
	Run(ctx context.Context) error
}
