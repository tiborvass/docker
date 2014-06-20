// Package trace provides a simple way to add call stack information to an error.
package trace

import (
	"bytes"
	"errors"
	"fmt"
	"runtime"
)

// Maximum depth of the call stack information
const MAX_DEPTH = 128

type errorPC struct {
	err error
	pc  uintptr
}

// errorPC is very lightweight on purpose.
// runtime.FuncForPC is not called upon creation of the error, but upon calling (stack).Error().
type stack []errorPC

func (st stack) Error() string {
	var buf bytes.Buffer
	for _, x := range st {
		f := runtime.FuncForPC(x.pc).Name()
		if x.err != nil {
			buf.WriteString(fmt.Sprintf("%s <<%v>>\n", f, x.err))
		} else {
			buf.WriteString(f)
			buf.WriteByte('\n')
		}
	}
	return buf.String()
}

// Error adds call stack information to err.
// If err already has that information, then Error simply returns err.
func Error(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(stack); ok {
		return err
	}
	pc := make([]uintptr, MAX_DEPTH)
	n := runtime.Callers(2, pc) // skip=2 because we don't care about runtime.Callers nor Error
	n -= 2                      // because we don't care about runtime.main nor runtime.goexit
	st := make(stack, n)
	for i := n - 1; i > 0; i-- { // i > 0 because we have to set the err field for i == 0 (see after loop)
		st[i] = errorPC{pc: pc[i]}
	}
	st[0] = errorPC{err: err, pc: pc[0]}
	return st
}

// Wrap adds more context to an existing err.
// If err has call stack information, the wrap string is added to it at the level where Wrap was called from.
// Otherwise, Wrap is equivalent to Error by squashing wrap and err.
func Wrap(wrap string, err error) error {
	st, ok := err.(stack)
	if !ok {
		return Error(fmt.Errorf("%s: %v", wrap, err))
	}
	pc := make([]uintptr, MAX_DEPTH)
	n := runtime.Callers(2, pc)
	n -= 2
	st[len(st)-n].err = errors.New(wrap)
	return st
}
