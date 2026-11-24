//go:build ios && cgo

// Native C ABI for embedding the actual Go runtime in a signed iPhoneOS app.
// This deliberately does NOT start the Agent in P1: its current initialization
// still includes collectors and lifecycle behavior unsuitable for stock iOS.
package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"unsafe"

	"github.com/henrygd/beszel/ios-app/BridgeCore"
)

//export BeszelGoVersion
func BeszelGoVersion() *C.char {
	return C.CString(bridgecore.Version())
}

//export BeszelGoPlatform
func BeszelGoPlatform() *C.char {
	return C.CString(bridgecore.Platform())
}

//export BeszelGoFree
func BeszelGoFree(ptr *C.char) {
	C.free(unsafe.Pointer(ptr))
}

func main() {}
