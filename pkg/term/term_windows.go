package term

import "errors"

type State struct{}

type Winsize struct {
	Height uint16
	Width  uint16
	x      uint16
	y      uint16
}

var ErrNotSupported = errors.New("not supported")

func GetWinsize(fd uintptr) (*Winsize, error) {
	return nil, ErrNotSupported
}

func SetWinsize(fd uintptr, ws *Winsize) error {
	return ErrNotSupported
}

func IsTerminal(fd uintptr) bool {
	return true
}

func RestoreTerminal(fd uintptr, state *State) error {
	return ErrNotSupported
}

func SaveState(fd uintptr) (*State, error) {
	return nil, ErrNotSupported
}

func DisableEcho(fd uintptr, state *State) error {
	return ErrNotSupported
}

func SetRawTerminal(fd uintptr) (*State, error) {
	return nil, ErrNotSupported
}
