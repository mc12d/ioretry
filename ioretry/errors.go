package ioretry

import (
	"fmt"
	"strings"
)

type PanicError struct {
	Err error
}

func (pe PanicError) Error() string {
	return fmt.Sprintf("encountered panic: %s", pe.Err)
}

type MultiFuncError []error

func (me MultiFuncError) Error() string {
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

type MultiIOError map[IO]error

func (me MultiIOError) Error() string {
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
