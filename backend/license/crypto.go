package license

/*
#cgo LDFLAGS: -lcrypto
#include "crypto.h"
*/
import "C"
import (
	"unsafe"
)

// 采集机器指纹
func getMachineFingerprint() string {
	cStr := C.get_machine_fingerprint()
	if cStr == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(cStr))
	return C.GoString(cStr)
}

// RSA 验签
func verifySignature(data, sigB64, pubkeyPEM string) bool {
	cData := C.CString(data)
	cSig := C.CString(sigB64)
	cPubkey := C.CString(pubkeyPEM)
	defer func() {
		C.free(unsafe.Pointer(cData))
		C.free(unsafe.Pointer(cSig))
		C.free(unsafe.Pointer(cPubkey))
	}()
	result := C.verify_rsa_signature(cData, cSig, cPubkey)
	return result == 1
}
