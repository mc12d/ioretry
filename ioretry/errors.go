package ioretry

import (
	"fmt"
	"os"
	"strings"
)

type SignalError struct {
	Sig os.Signal
}

func (e SignalError) Error() string {
	sig := "unknown"
	if e.Sig != nil {
		sig = e.Sig.String()
	}
	return fmt.Sprintf("encountered signal: %s", sig)
}

type PanicError struct {
	Err error
}

func (pe PanicError) Error() string {
	return fmt.Sprintf("encountered panic: %s", pe.Err)
}

type MultiError []error

func (me MultiError) Error() string {
	sb := strings.Builder{}
	sb.WriteString("multiple errors: ")

	for i, e := range me {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(e.Error())
	}
	return sb.String()
}

type MultiResourceError map[IO]error

func (me MultiResourceError) Error() string {
	sb := strings.Builder{}
	sb.WriteString("multiple errors: ")

	firstLine := true
	for res, err := range me {
		if firstLine {
			sb.WriteString(", ")
			firstLine = false
		}
		sb.WriteString(fmt.Sprintf("[resource %T] %s", res, err.Error()))
	}
	return sb.String()
}
