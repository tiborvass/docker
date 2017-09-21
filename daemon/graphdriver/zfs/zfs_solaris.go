// +build solaris,cgo

package zfs

/*
#include <sys/statvfs.h>
#include <stdlib.h>

static inline struct statvfs *getstatfs(char *s) {
        struct statvfs *buf;
        int err;
        buf = (struct statvfs *)malloc(sizeof(struct statvfs));
        err = statvfs(s, buf);
        return buf;
}
*/
import "C"
import (
	"path/filepath"
	"strings"
	"unsafe"

	"github.com/tiborvass/docker/daemon/graphdriver"
	"github.com/sirupsen/logrus"
)

func checkRootdirFs(rootdir string) error {

	cs := C.CString(filepath.Dir(rootdir))
	defer C.free(unsafe.Pointer(cs))
	buf := C.getstatfs(cs)
	defer C.free(unsafe.Pointer(buf))

	// on Solaris buf.f_basetype contains ['z', 'f', 's', 0 ... ]
	if (buf.f_basetype[0] != 122) || (buf.f_basetype[1] != 102) || (buf.f_basetype[2] != 115) ||
		(buf.f_basetype[3] != 0) {
		logrus.Debugf("[zfs] no zfs dataset found for rootdir '%s'", rootdir)
		return graphdriver.ErrPrerequisites
	}

	return nil
}

/* rootfs is introduced to comply with the OCI spec
which states that root filesystem must be mounted at <CID>/rootfs/ instead of <CID>/
*/
func getMountpoint(id string) string {
	maxlen := 12

	// we need to preserve filesystem suffix
	suffix := strings.SplitN(id, "-", 2)

	if len(suffix) > 1 {
		return filepath.Join(id[:maxlen]+"-"+suffix[1], "rootfs", "root")
	}

	return filepath.Join(id[:maxlen], "rootfs", "root")
}
