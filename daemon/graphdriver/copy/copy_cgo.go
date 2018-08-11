// +build cgo

package copy

/*
#include <linux/fs.h>

#ifndef FICLONE
#define FICLONE		_IOW(0x94, 9, int)
#endif
*/
import "C"

var _FICLONE uintptr = C.FICLONE
