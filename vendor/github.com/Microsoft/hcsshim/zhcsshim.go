// MACHINE GENERATED BY 'go generate' COMMAND; DO NOT EDIT

package hcsshim

import (
	"syscall"
	"unsafe"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
)

var _ unsafe.Pointer

// Do the interface allocations only once for common
// Errno values.
const (
	errnoERROR_IO_PENDING = 997
)

var (
	errERROR_IO_PENDING error = syscall.Errno(errnoERROR_IO_PENDING)
)

// errnoErr returns common boxed Errno values, to prevent
// allocations at runtime.
func errnoErr(e syscall.Errno) error {
	switch e {
	case 0:
		return nil
	case errnoERROR_IO_PENDING:
		return errERROR_IO_PENDING
	}
	// TODO: add more here, after collecting data on the common
	// error values see on Windows. (perhaps when running
	// all.bat?)
	return e
}

var (
	modole32     = windows.NewLazySystemDLL("ole32.dll")
	modiphlpapi  = windows.NewLazySystemDLL("iphlpapi.dll")
	modvmcompute = windows.NewLazySystemDLL("vmcompute.dll")
	modntdll     = windows.NewLazySystemDLL("ntdll.dll")
	modkernel32  = windows.NewLazySystemDLL("kernel32.dll")

	procCoTaskMemFree                      = modole32.NewProc("CoTaskMemFree")
	procSetCurrentThreadCompartmentId      = modiphlpapi.NewProc("SetCurrentThreadCompartmentId")
	procActivateLayer                      = modvmcompute.NewProc("ActivateLayer")
	procCopyLayer                          = modvmcompute.NewProc("CopyLayer")
	procCreateLayer                        = modvmcompute.NewProc("CreateLayer")
	procCreateSandboxLayer                 = modvmcompute.NewProc("CreateSandboxLayer")
	procExpandSandboxSize                  = modvmcompute.NewProc("ExpandSandboxSize")
	procDeactivateLayer                    = modvmcompute.NewProc("DeactivateLayer")
	procDestroyLayer                       = modvmcompute.NewProc("DestroyLayer")
	procExportLayer                        = modvmcompute.NewProc("ExportLayer")
	procGetLayerMountPath                  = modvmcompute.NewProc("GetLayerMountPath")
	procGetBaseImages                      = modvmcompute.NewProc("GetBaseImages")
	procImportLayer                        = modvmcompute.NewProc("ImportLayer")
	procLayerExists                        = modvmcompute.NewProc("LayerExists")
	procNameToGuid                         = modvmcompute.NewProc("NameToGuid")
	procPrepareLayer                       = modvmcompute.NewProc("PrepareLayer")
	procUnprepareLayer                     = modvmcompute.NewProc("UnprepareLayer")
	procProcessBaseImage                   = modvmcompute.NewProc("ProcessBaseImage")
	procProcessUtilityImage                = modvmcompute.NewProc("ProcessUtilityImage")
	procImportLayerBegin                   = modvmcompute.NewProc("ImportLayerBegin")
	procImportLayerNext                    = modvmcompute.NewProc("ImportLayerNext")
	procImportLayerWrite                   = modvmcompute.NewProc("ImportLayerWrite")
	procImportLayerEnd                     = modvmcompute.NewProc("ImportLayerEnd")
	procExportLayerBegin                   = modvmcompute.NewProc("ExportLayerBegin")
	procExportLayerNext                    = modvmcompute.NewProc("ExportLayerNext")
	procExportLayerRead                    = modvmcompute.NewProc("ExportLayerRead")
	procExportLayerEnd                     = modvmcompute.NewProc("ExportLayerEnd")
	procHcsEnumerateComputeSystems         = modvmcompute.NewProc("HcsEnumerateComputeSystems")
	procHcsCreateComputeSystem             = modvmcompute.NewProc("HcsCreateComputeSystem")
	procHcsOpenComputeSystem               = modvmcompute.NewProc("HcsOpenComputeSystem")
	procHcsCloseComputeSystem              = modvmcompute.NewProc("HcsCloseComputeSystem")
	procHcsStartComputeSystem              = modvmcompute.NewProc("HcsStartComputeSystem")
	procHcsShutdownComputeSystem           = modvmcompute.NewProc("HcsShutdownComputeSystem")
	procHcsTerminateComputeSystem          = modvmcompute.NewProc("HcsTerminateComputeSystem")
	procHcsPauseComputeSystem              = modvmcompute.NewProc("HcsPauseComputeSystem")
	procHcsResumeComputeSystem             = modvmcompute.NewProc("HcsResumeComputeSystem")
	procHcsGetComputeSystemProperties      = modvmcompute.NewProc("HcsGetComputeSystemProperties")
	procHcsModifyComputeSystem             = modvmcompute.NewProc("HcsModifyComputeSystem")
	procHcsRegisterComputeSystemCallback   = modvmcompute.NewProc("HcsRegisterComputeSystemCallback")
	procHcsUnregisterComputeSystemCallback = modvmcompute.NewProc("HcsUnregisterComputeSystemCallback")
	procHcsCreateProcess                   = modvmcompute.NewProc("HcsCreateProcess")
	procHcsOpenProcess                     = modvmcompute.NewProc("HcsOpenProcess")
	procHcsCloseProcess                    = modvmcompute.NewProc("HcsCloseProcess")
	procHcsTerminateProcess                = modvmcompute.NewProc("HcsTerminateProcess")
	procHcsGetProcessInfo                  = modvmcompute.NewProc("HcsGetProcessInfo")
	procHcsGetProcessProperties            = modvmcompute.NewProc("HcsGetProcessProperties")
	procHcsModifyProcess                   = modvmcompute.NewProc("HcsModifyProcess")
	procHcsGetServiceProperties            = modvmcompute.NewProc("HcsGetServiceProperties")
	procHcsRegisterProcessCallback         = modvmcompute.NewProc("HcsRegisterProcessCallback")
	procHcsUnregisterProcessCallback       = modvmcompute.NewProc("HcsUnregisterProcessCallback")
	procHcsModifyServiceSettings           = modvmcompute.NewProc("HcsModifyServiceSettings")
	procHNSCall                            = modvmcompute.NewProc("HNSCall")
	procNtCreateFile                       = modntdll.NewProc("NtCreateFile")
	procNtSetInformationFile               = modntdll.NewProc("NtSetInformationFile")
	procRtlNtStatusToDosErrorNoTeb         = modntdll.NewProc("RtlNtStatusToDosErrorNoTeb")
	procLocalAlloc                         = modkernel32.NewProc("LocalAlloc")
	procLocalFree                          = modkernel32.NewProc("LocalFree")
)

func coTaskMemFree(buffer unsafe.Pointer) {
	syscall.Syscall(procCoTaskMemFree.Addr(), 1, uintptr(buffer), 0, 0)
	return
}

func SetCurrentThreadCompartmentId(compartmentId uint32) (hr error) {
	r0, _, _ := syscall.Syscall(procSetCurrentThreadCompartmentId.Addr(), 1, uintptr(compartmentId), 0, 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func activateLayer(info *driverInfo, id string) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(id)
	if hr != nil {
		return
	}
	return _activateLayer(info, _p0)
}

func _activateLayer(info *driverInfo, id *uint16) (hr error) {
	if hr = procActivateLayer.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procActivateLayer.Addr(), 2, uintptr(unsafe.Pointer(info)), uintptr(unsafe.Pointer(id)), 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func copyLayer(info *driverInfo, srcId string, dstId string, descriptors []WC_LAYER_DESCRIPTOR) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(srcId)
	if hr != nil {
		return
	}
	var _p1 *uint16
	_p1, hr = syscall.UTF16PtrFromString(dstId)
	if hr != nil {
		return
	}
	return _copyLayer(info, _p0, _p1, descriptors)
}

func _copyLayer(info *driverInfo, srcId *uint16, dstId *uint16, descriptors []WC_LAYER_DESCRIPTOR) (hr error) {
	var _p2 *WC_LAYER_DESCRIPTOR
	if len(descriptors) > 0 {
		_p2 = &descriptors[0]
	}
	if hr = procCopyLayer.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall6(procCopyLayer.Addr(), 5, uintptr(unsafe.Pointer(info)), uintptr(unsafe.Pointer(srcId)), uintptr(unsafe.Pointer(dstId)), uintptr(unsafe.Pointer(_p2)), uintptr(len(descriptors)), 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func createLayer(info *driverInfo, id string, parent string) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(id)
	if hr != nil {
		return
	}
	var _p1 *uint16
	_p1, hr = syscall.UTF16PtrFromString(parent)
	if hr != nil {
		return
	}
	return _createLayer(info, _p0, _p1)
}

func _createLayer(info *driverInfo, id *uint16, parent *uint16) (hr error) {
	if hr = procCreateLayer.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procCreateLayer.Addr(), 3, uintptr(unsafe.Pointer(info)), uintptr(unsafe.Pointer(id)), uintptr(unsafe.Pointer(parent)))
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func createSandboxLayer(info *driverInfo, id string, parent string, descriptors []WC_LAYER_DESCRIPTOR) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(id)
	if hr != nil {
		return
	}
	var _p1 *uint16
	_p1, hr = syscall.UTF16PtrFromString(parent)
	if hr != nil {
		return
	}
	return _createSandboxLayer(info, _p0, _p1, descriptors)
}

func _createSandboxLayer(info *driverInfo, id *uint16, parent *uint16, descriptors []WC_LAYER_DESCRIPTOR) (hr error) {
	var _p2 *WC_LAYER_DESCRIPTOR
	if len(descriptors) > 0 {
		_p2 = &descriptors[0]
	}
	if hr = procCreateSandboxLayer.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall6(procCreateSandboxLayer.Addr(), 5, uintptr(unsafe.Pointer(info)), uintptr(unsafe.Pointer(id)), uintptr(unsafe.Pointer(parent)), uintptr(unsafe.Pointer(_p2)), uintptr(len(descriptors)), 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func expandSandboxSize(info *driverInfo, id string, size uint64) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(id)
	if hr != nil {
		return
	}
	return _expandSandboxSize(info, _p0, size)
}

func _expandSandboxSize(info *driverInfo, id *uint16, size uint64) (hr error) {
	if hr = procExpandSandboxSize.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procExpandSandboxSize.Addr(), 3, uintptr(unsafe.Pointer(info)), uintptr(unsafe.Pointer(id)), uintptr(size))
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func deactivateLayer(info *driverInfo, id string) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(id)
	if hr != nil {
		return
	}
	return _deactivateLayer(info, _p0)
}

func _deactivateLayer(info *driverInfo, id *uint16) (hr error) {
	if hr = procDeactivateLayer.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procDeactivateLayer.Addr(), 2, uintptr(unsafe.Pointer(info)), uintptr(unsafe.Pointer(id)), 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func destroyLayer(info *driverInfo, id string) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(id)
	if hr != nil {
		return
	}
	return _destroyLayer(info, _p0)
}

func _destroyLayer(info *driverInfo, id *uint16) (hr error) {
	if hr = procDestroyLayer.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procDestroyLayer.Addr(), 2, uintptr(unsafe.Pointer(info)), uintptr(unsafe.Pointer(id)), 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func exportLayer(info *driverInfo, id string, path string, descriptors []WC_LAYER_DESCRIPTOR) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(id)
	if hr != nil {
		return
	}
	var _p1 *uint16
	_p1, hr = syscall.UTF16PtrFromString(path)
	if hr != nil {
		return
	}
	return _exportLayer(info, _p0, _p1, descriptors)
}

func _exportLayer(info *driverInfo, id *uint16, path *uint16, descriptors []WC_LAYER_DESCRIPTOR) (hr error) {
	var _p2 *WC_LAYER_DESCRIPTOR
	if len(descriptors) > 0 {
		_p2 = &descriptors[0]
	}
	if hr = procExportLayer.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall6(procExportLayer.Addr(), 5, uintptr(unsafe.Pointer(info)), uintptr(unsafe.Pointer(id)), uintptr(unsafe.Pointer(path)), uintptr(unsafe.Pointer(_p2)), uintptr(len(descriptors)), 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func getLayerMountPath(info *driverInfo, id string, length *uintptr, buffer *uint16) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(id)
	if hr != nil {
		return
	}
	return _getLayerMountPath(info, _p0, length, buffer)
}

func _getLayerMountPath(info *driverInfo, id *uint16, length *uintptr, buffer *uint16) (hr error) {
	if hr = procGetLayerMountPath.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall6(procGetLayerMountPath.Addr(), 4, uintptr(unsafe.Pointer(info)), uintptr(unsafe.Pointer(id)), uintptr(unsafe.Pointer(length)), uintptr(unsafe.Pointer(buffer)), 0, 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func getBaseImages(buffer **uint16) (hr error) {
	if hr = procGetBaseImages.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procGetBaseImages.Addr(), 1, uintptr(unsafe.Pointer(buffer)), 0, 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func importLayer(info *driverInfo, id string, path string, descriptors []WC_LAYER_DESCRIPTOR) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(id)
	if hr != nil {
		return
	}
	var _p1 *uint16
	_p1, hr = syscall.UTF16PtrFromString(path)
	if hr != nil {
		return
	}
	return _importLayer(info, _p0, _p1, descriptors)
}

func _importLayer(info *driverInfo, id *uint16, path *uint16, descriptors []WC_LAYER_DESCRIPTOR) (hr error) {
	var _p2 *WC_LAYER_DESCRIPTOR
	if len(descriptors) > 0 {
		_p2 = &descriptors[0]
	}
	if hr = procImportLayer.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall6(procImportLayer.Addr(), 5, uintptr(unsafe.Pointer(info)), uintptr(unsafe.Pointer(id)), uintptr(unsafe.Pointer(path)), uintptr(unsafe.Pointer(_p2)), uintptr(len(descriptors)), 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func layerExists(info *driverInfo, id string, exists *uint32) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(id)
	if hr != nil {
		return
	}
	return _layerExists(info, _p0, exists)
}

func _layerExists(info *driverInfo, id *uint16, exists *uint32) (hr error) {
	if hr = procLayerExists.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procLayerExists.Addr(), 3, uintptr(unsafe.Pointer(info)), uintptr(unsafe.Pointer(id)), uintptr(unsafe.Pointer(exists)))
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func nameToGuid(name string, guid *GUID) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(name)
	if hr != nil {
		return
	}
	return _nameToGuid(_p0, guid)
}

func _nameToGuid(name *uint16, guid *GUID) (hr error) {
	if hr = procNameToGuid.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procNameToGuid.Addr(), 2, uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(guid)), 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func prepareLayer(info *driverInfo, id string, descriptors []WC_LAYER_DESCRIPTOR) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(id)
	if hr != nil {
		return
	}
	return _prepareLayer(info, _p0, descriptors)
}

func _prepareLayer(info *driverInfo, id *uint16, descriptors []WC_LAYER_DESCRIPTOR) (hr error) {
	var _p1 *WC_LAYER_DESCRIPTOR
	if len(descriptors) > 0 {
		_p1 = &descriptors[0]
	}
	if hr = procPrepareLayer.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall6(procPrepareLayer.Addr(), 4, uintptr(unsafe.Pointer(info)), uintptr(unsafe.Pointer(id)), uintptr(unsafe.Pointer(_p1)), uintptr(len(descriptors)), 0, 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func unprepareLayer(info *driverInfo, id string) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(id)
	if hr != nil {
		return
	}
	return _unprepareLayer(info, _p0)
}

func _unprepareLayer(info *driverInfo, id *uint16) (hr error) {
	if hr = procUnprepareLayer.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procUnprepareLayer.Addr(), 2, uintptr(unsafe.Pointer(info)), uintptr(unsafe.Pointer(id)), 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func processBaseImage(path string) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(path)
	if hr != nil {
		return
	}
	return _processBaseImage(_p0)
}

func _processBaseImage(path *uint16) (hr error) {
	if hr = procProcessBaseImage.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procProcessBaseImage.Addr(), 1, uintptr(unsafe.Pointer(path)), 0, 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func processUtilityImage(path string) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(path)
	if hr != nil {
		return
	}
	return _processUtilityImage(_p0)
}

func _processUtilityImage(path *uint16) (hr error) {
	if hr = procProcessUtilityImage.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procProcessUtilityImage.Addr(), 1, uintptr(unsafe.Pointer(path)), 0, 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func importLayerBegin(info *driverInfo, id string, descriptors []WC_LAYER_DESCRIPTOR, context *uintptr) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(id)
	if hr != nil {
		return
	}
	return _importLayerBegin(info, _p0, descriptors, context)
}

func _importLayerBegin(info *driverInfo, id *uint16, descriptors []WC_LAYER_DESCRIPTOR, context *uintptr) (hr error) {
	var _p1 *WC_LAYER_DESCRIPTOR
	if len(descriptors) > 0 {
		_p1 = &descriptors[0]
	}
	if hr = procImportLayerBegin.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall6(procImportLayerBegin.Addr(), 5, uintptr(unsafe.Pointer(info)), uintptr(unsafe.Pointer(id)), uintptr(unsafe.Pointer(_p1)), uintptr(len(descriptors)), uintptr(unsafe.Pointer(context)), 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func importLayerNext(context uintptr, fileName string, fileInfo *winio.FileBasicInfo) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(fileName)
	if hr != nil {
		return
	}
	return _importLayerNext(context, _p0, fileInfo)
}

func _importLayerNext(context uintptr, fileName *uint16, fileInfo *winio.FileBasicInfo) (hr error) {
	if hr = procImportLayerNext.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procImportLayerNext.Addr(), 3, uintptr(context), uintptr(unsafe.Pointer(fileName)), uintptr(unsafe.Pointer(fileInfo)))
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func importLayerWrite(context uintptr, buffer []byte) (hr error) {
	var _p0 *byte
	if len(buffer) > 0 {
		_p0 = &buffer[0]
	}
	if hr = procImportLayerWrite.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procImportLayerWrite.Addr(), 3, uintptr(context), uintptr(unsafe.Pointer(_p0)), uintptr(len(buffer)))
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func importLayerEnd(context uintptr) (hr error) {
	if hr = procImportLayerEnd.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procImportLayerEnd.Addr(), 1, uintptr(context), 0, 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func exportLayerBegin(info *driverInfo, id string, descriptors []WC_LAYER_DESCRIPTOR, context *uintptr) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(id)
	if hr != nil {
		return
	}
	return _exportLayerBegin(info, _p0, descriptors, context)
}

func _exportLayerBegin(info *driverInfo, id *uint16, descriptors []WC_LAYER_DESCRIPTOR, context *uintptr) (hr error) {
	var _p1 *WC_LAYER_DESCRIPTOR
	if len(descriptors) > 0 {
		_p1 = &descriptors[0]
	}
	if hr = procExportLayerBegin.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall6(procExportLayerBegin.Addr(), 5, uintptr(unsafe.Pointer(info)), uintptr(unsafe.Pointer(id)), uintptr(unsafe.Pointer(_p1)), uintptr(len(descriptors)), uintptr(unsafe.Pointer(context)), 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func exportLayerNext(context uintptr, fileName **uint16, fileInfo *winio.FileBasicInfo, fileSize *int64, deleted *uint32) (hr error) {
	if hr = procExportLayerNext.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall6(procExportLayerNext.Addr(), 5, uintptr(context), uintptr(unsafe.Pointer(fileName)), uintptr(unsafe.Pointer(fileInfo)), uintptr(unsafe.Pointer(fileSize)), uintptr(unsafe.Pointer(deleted)), 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func exportLayerRead(context uintptr, buffer []byte, bytesRead *uint32) (hr error) {
	var _p0 *byte
	if len(buffer) > 0 {
		_p0 = &buffer[0]
	}
	if hr = procExportLayerRead.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall6(procExportLayerRead.Addr(), 4, uintptr(context), uintptr(unsafe.Pointer(_p0)), uintptr(len(buffer)), uintptr(unsafe.Pointer(bytesRead)), 0, 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func exportLayerEnd(context uintptr) (hr error) {
	if hr = procExportLayerEnd.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procExportLayerEnd.Addr(), 1, uintptr(context), 0, 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func hcsEnumerateComputeSystems(query string, computeSystems **uint16, result **uint16) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(query)
	if hr != nil {
		return
	}
	return _hcsEnumerateComputeSystems(_p0, computeSystems, result)
}

func _hcsEnumerateComputeSystems(query *uint16, computeSystems **uint16, result **uint16) (hr error) {
	if hr = procHcsEnumerateComputeSystems.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procHcsEnumerateComputeSystems.Addr(), 3, uintptr(unsafe.Pointer(query)), uintptr(unsafe.Pointer(computeSystems)), uintptr(unsafe.Pointer(result)))
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func hcsCreateComputeSystem(id string, configuration string, identity syscall.Handle, computeSystem *hcsSystem, result **uint16) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(id)
	if hr != nil {
		return
	}
	var _p1 *uint16
	_p1, hr = syscall.UTF16PtrFromString(configuration)
	if hr != nil {
		return
	}
	return _hcsCreateComputeSystem(_p0, _p1, identity, computeSystem, result)
}

func _hcsCreateComputeSystem(id *uint16, configuration *uint16, identity syscall.Handle, computeSystem *hcsSystem, result **uint16) (hr error) {
	if hr = procHcsCreateComputeSystem.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall6(procHcsCreateComputeSystem.Addr(), 5, uintptr(unsafe.Pointer(id)), uintptr(unsafe.Pointer(configuration)), uintptr(identity), uintptr(unsafe.Pointer(computeSystem)), uintptr(unsafe.Pointer(result)), 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func hcsOpenComputeSystem(id string, computeSystem *hcsSystem, result **uint16) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(id)
	if hr != nil {
		return
	}
	return _hcsOpenComputeSystem(_p0, computeSystem, result)
}

func _hcsOpenComputeSystem(id *uint16, computeSystem *hcsSystem, result **uint16) (hr error) {
	if hr = procHcsOpenComputeSystem.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procHcsOpenComputeSystem.Addr(), 3, uintptr(unsafe.Pointer(id)), uintptr(unsafe.Pointer(computeSystem)), uintptr(unsafe.Pointer(result)))
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func hcsCloseComputeSystem(computeSystem hcsSystem) (hr error) {
	if hr = procHcsCloseComputeSystem.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procHcsCloseComputeSystem.Addr(), 1, uintptr(computeSystem), 0, 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func hcsStartComputeSystem(computeSystem hcsSystem, options string, result **uint16) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(options)
	if hr != nil {
		return
	}
	return _hcsStartComputeSystem(computeSystem, _p0, result)
}

func _hcsStartComputeSystem(computeSystem hcsSystem, options *uint16, result **uint16) (hr error) {
	if hr = procHcsStartComputeSystem.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procHcsStartComputeSystem.Addr(), 3, uintptr(computeSystem), uintptr(unsafe.Pointer(options)), uintptr(unsafe.Pointer(result)))
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func hcsShutdownComputeSystem(computeSystem hcsSystem, options string, result **uint16) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(options)
	if hr != nil {
		return
	}
	return _hcsShutdownComputeSystem(computeSystem, _p0, result)
}

func _hcsShutdownComputeSystem(computeSystem hcsSystem, options *uint16, result **uint16) (hr error) {
	if hr = procHcsShutdownComputeSystem.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procHcsShutdownComputeSystem.Addr(), 3, uintptr(computeSystem), uintptr(unsafe.Pointer(options)), uintptr(unsafe.Pointer(result)))
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func hcsTerminateComputeSystem(computeSystem hcsSystem, options string, result **uint16) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(options)
	if hr != nil {
		return
	}
	return _hcsTerminateComputeSystem(computeSystem, _p0, result)
}

func _hcsTerminateComputeSystem(computeSystem hcsSystem, options *uint16, result **uint16) (hr error) {
	if hr = procHcsTerminateComputeSystem.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procHcsTerminateComputeSystem.Addr(), 3, uintptr(computeSystem), uintptr(unsafe.Pointer(options)), uintptr(unsafe.Pointer(result)))
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func hcsPauseComputeSystem(computeSystem hcsSystem, options string, result **uint16) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(options)
	if hr != nil {
		return
	}
	return _hcsPauseComputeSystem(computeSystem, _p0, result)
}

func _hcsPauseComputeSystem(computeSystem hcsSystem, options *uint16, result **uint16) (hr error) {
	if hr = procHcsPauseComputeSystem.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procHcsPauseComputeSystem.Addr(), 3, uintptr(computeSystem), uintptr(unsafe.Pointer(options)), uintptr(unsafe.Pointer(result)))
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func hcsResumeComputeSystem(computeSystem hcsSystem, options string, result **uint16) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(options)
	if hr != nil {
		return
	}
	return _hcsResumeComputeSystem(computeSystem, _p0, result)
}

func _hcsResumeComputeSystem(computeSystem hcsSystem, options *uint16, result **uint16) (hr error) {
	if hr = procHcsResumeComputeSystem.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procHcsResumeComputeSystem.Addr(), 3, uintptr(computeSystem), uintptr(unsafe.Pointer(options)), uintptr(unsafe.Pointer(result)))
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func hcsGetComputeSystemProperties(computeSystem hcsSystem, propertyQuery string, properties **uint16, result **uint16) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(propertyQuery)
	if hr != nil {
		return
	}
	return _hcsGetComputeSystemProperties(computeSystem, _p0, properties, result)
}

func _hcsGetComputeSystemProperties(computeSystem hcsSystem, propertyQuery *uint16, properties **uint16, result **uint16) (hr error) {
	if hr = procHcsGetComputeSystemProperties.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall6(procHcsGetComputeSystemProperties.Addr(), 4, uintptr(computeSystem), uintptr(unsafe.Pointer(propertyQuery)), uintptr(unsafe.Pointer(properties)), uintptr(unsafe.Pointer(result)), 0, 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func hcsModifyComputeSystem(computeSystem hcsSystem, configuration string, result **uint16) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(configuration)
	if hr != nil {
		return
	}
	return _hcsModifyComputeSystem(computeSystem, _p0, result)
}

func _hcsModifyComputeSystem(computeSystem hcsSystem, configuration *uint16, result **uint16) (hr error) {
	if hr = procHcsModifyComputeSystem.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procHcsModifyComputeSystem.Addr(), 3, uintptr(computeSystem), uintptr(unsafe.Pointer(configuration)), uintptr(unsafe.Pointer(result)))
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func hcsRegisterComputeSystemCallback(computeSystem hcsSystem, callback uintptr, context uintptr, callbackHandle *hcsCallback) (hr error) {
	if hr = procHcsRegisterComputeSystemCallback.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall6(procHcsRegisterComputeSystemCallback.Addr(), 4, uintptr(computeSystem), uintptr(callback), uintptr(context), uintptr(unsafe.Pointer(callbackHandle)), 0, 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func hcsUnregisterComputeSystemCallback(callbackHandle hcsCallback) (hr error) {
	if hr = procHcsUnregisterComputeSystemCallback.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procHcsUnregisterComputeSystemCallback.Addr(), 1, uintptr(callbackHandle), 0, 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func hcsCreateProcess(computeSystem hcsSystem, processParameters string, processInformation *hcsProcessInformation, process *hcsProcess, result **uint16) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(processParameters)
	if hr != nil {
		return
	}
	return _hcsCreateProcess(computeSystem, _p0, processInformation, process, result)
}

func _hcsCreateProcess(computeSystem hcsSystem, processParameters *uint16, processInformation *hcsProcessInformation, process *hcsProcess, result **uint16) (hr error) {
	if hr = procHcsCreateProcess.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall6(procHcsCreateProcess.Addr(), 5, uintptr(computeSystem), uintptr(unsafe.Pointer(processParameters)), uintptr(unsafe.Pointer(processInformation)), uintptr(unsafe.Pointer(process)), uintptr(unsafe.Pointer(result)), 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func hcsOpenProcess(computeSystem hcsSystem, pid uint32, process *hcsProcess, result **uint16) (hr error) {
	if hr = procHcsOpenProcess.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall6(procHcsOpenProcess.Addr(), 4, uintptr(computeSystem), uintptr(pid), uintptr(unsafe.Pointer(process)), uintptr(unsafe.Pointer(result)), 0, 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func hcsCloseProcess(process hcsProcess) (hr error) {
	if hr = procHcsCloseProcess.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procHcsCloseProcess.Addr(), 1, uintptr(process), 0, 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func hcsTerminateProcess(process hcsProcess, result **uint16) (hr error) {
	if hr = procHcsTerminateProcess.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procHcsTerminateProcess.Addr(), 2, uintptr(process), uintptr(unsafe.Pointer(result)), 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func hcsGetProcessInfo(process hcsProcess, processInformation *hcsProcessInformation, result **uint16) (hr error) {
	if hr = procHcsGetProcessInfo.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procHcsGetProcessInfo.Addr(), 3, uintptr(process), uintptr(unsafe.Pointer(processInformation)), uintptr(unsafe.Pointer(result)))
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func hcsGetProcessProperties(process hcsProcess, processProperties **uint16, result **uint16) (hr error) {
	if hr = procHcsGetProcessProperties.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procHcsGetProcessProperties.Addr(), 3, uintptr(process), uintptr(unsafe.Pointer(processProperties)), uintptr(unsafe.Pointer(result)))
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func hcsModifyProcess(process hcsProcess, settings string, result **uint16) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(settings)
	if hr != nil {
		return
	}
	return _hcsModifyProcess(process, _p0, result)
}

func _hcsModifyProcess(process hcsProcess, settings *uint16, result **uint16) (hr error) {
	if hr = procHcsModifyProcess.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procHcsModifyProcess.Addr(), 3, uintptr(process), uintptr(unsafe.Pointer(settings)), uintptr(unsafe.Pointer(result)))
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func hcsGetServiceProperties(propertyQuery string, properties **uint16, result **uint16) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(propertyQuery)
	if hr != nil {
		return
	}
	return _hcsGetServiceProperties(_p0, properties, result)
}

func _hcsGetServiceProperties(propertyQuery *uint16, properties **uint16, result **uint16) (hr error) {
	if hr = procHcsGetServiceProperties.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procHcsGetServiceProperties.Addr(), 3, uintptr(unsafe.Pointer(propertyQuery)), uintptr(unsafe.Pointer(properties)), uintptr(unsafe.Pointer(result)))
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func hcsRegisterProcessCallback(process hcsProcess, callback uintptr, context uintptr, callbackHandle *hcsCallback) (hr error) {
	if hr = procHcsRegisterProcessCallback.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall6(procHcsRegisterProcessCallback.Addr(), 4, uintptr(process), uintptr(callback), uintptr(context), uintptr(unsafe.Pointer(callbackHandle)), 0, 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func hcsUnregisterProcessCallback(callbackHandle hcsCallback) (hr error) {
	if hr = procHcsUnregisterProcessCallback.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procHcsUnregisterProcessCallback.Addr(), 1, uintptr(callbackHandle), 0, 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func hcsModifyServiceSettings(settings string, result **uint16) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(settings)
	if hr != nil {
		return
	}
	return _hcsModifyServiceSettings(_p0, result)
}

func _hcsModifyServiceSettings(settings *uint16, result **uint16) (hr error) {
	if hr = procHcsModifyServiceSettings.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall(procHcsModifyServiceSettings.Addr(), 2, uintptr(unsafe.Pointer(settings)), uintptr(unsafe.Pointer(result)), 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func _hnsCall(method string, path string, object string, response **uint16) (hr error) {
	var _p0 *uint16
	_p0, hr = syscall.UTF16PtrFromString(method)
	if hr != nil {
		return
	}
	var _p1 *uint16
	_p1, hr = syscall.UTF16PtrFromString(path)
	if hr != nil {
		return
	}
	var _p2 *uint16
	_p2, hr = syscall.UTF16PtrFromString(object)
	if hr != nil {
		return
	}
	return __hnsCall(_p0, _p1, _p2, response)
}

func __hnsCall(method *uint16, path *uint16, object *uint16, response **uint16) (hr error) {
	if hr = procHNSCall.Find(); hr != nil {
		return
	}
	r0, _, _ := syscall.Syscall6(procHNSCall.Addr(), 4, uintptr(unsafe.Pointer(method)), uintptr(unsafe.Pointer(path)), uintptr(unsafe.Pointer(object)), uintptr(unsafe.Pointer(response)), 0, 0)
	if int32(r0) < 0 {
		hr = syscall.Errno(win32FromHresult(r0))
	}
	return
}

func ntCreateFile(handle *uintptr, accessMask uint32, oa *objectAttributes, iosb *ioStatusBlock, allocationSize *uint64, fileAttributes uint32, shareAccess uint32, createDisposition uint32, createOptions uint32, eaBuffer *byte, eaLength uint32) (status uint32) {
	r0, _, _ := syscall.Syscall12(procNtCreateFile.Addr(), 11, uintptr(unsafe.Pointer(handle)), uintptr(accessMask), uintptr(unsafe.Pointer(oa)), uintptr(unsafe.Pointer(iosb)), uintptr(unsafe.Pointer(allocationSize)), uintptr(fileAttributes), uintptr(shareAccess), uintptr(createDisposition), uintptr(createOptions), uintptr(unsafe.Pointer(eaBuffer)), uintptr(eaLength), 0)
	status = uint32(r0)
	return
}

func ntSetInformationFile(handle uintptr, iosb *ioStatusBlock, information uintptr, length uint32, class uint32) (status uint32) {
	r0, _, _ := syscall.Syscall6(procNtSetInformationFile.Addr(), 5, uintptr(handle), uintptr(unsafe.Pointer(iosb)), uintptr(information), uintptr(length), uintptr(class), 0)
	status = uint32(r0)
	return
}

func rtlNtStatusToDosError(status uint32) (winerr error) {
	r0, _, _ := syscall.Syscall(procRtlNtStatusToDosErrorNoTeb.Addr(), 1, uintptr(status), 0, 0)
	if r0 != 0 {
		winerr = syscall.Errno(r0)
	}
	return
}

func localAlloc(flags uint32, size int) (ptr uintptr) {
	r0, _, _ := syscall.Syscall(procLocalAlloc.Addr(), 2, uintptr(flags), uintptr(size), 0)
	ptr = uintptr(r0)
	return
}

func localFree(ptr uintptr) {
	syscall.Syscall(procLocalFree.Addr(), 1, uintptr(ptr), 0, 0)
	return
}
