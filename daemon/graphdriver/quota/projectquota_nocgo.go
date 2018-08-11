// +build !cgo

package quota

import "errors"

var ErrNotSupported = errors.New("quota unsupported")

// Quota limit params - currently we only control blocks hard limit
type Quota struct {
	Size uint64
}

type Control struct {}

func NewControl(_ string) (*Control, error) { return nil, ErrNotSupported }

func (c *Control) SetQuota(_ string, _ Quota) error { return ErrNotSupported }
