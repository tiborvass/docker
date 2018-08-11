// +build cgo

package stats

/*
#include <unistd.h>
*/
import "C"

func getNumberOnlineCPUs() (uint32, error) {
	i, err := C.sysconf(C._SC_NPROCESSORS_ONLN)
	// According to POSIX - errno is undefined after successful
	// sysconf, and can be non-zero in several cases, so look for
	// error in returned value not in errno.
	// (https://sourceware.org/bugzilla/show_bug.cgi?id=21536)
	if i == -1 {
		return 0, err
	}
	return uint32(i), nil
}
