// +build !cgo

package stats

import "runtime"

func getNumberOnlineCPUs() (uint32, error) {
	return uint32(runtime.NumCPU()), nil
}
