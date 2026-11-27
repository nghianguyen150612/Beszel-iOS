//go:build ios && cgo

// Native C ABI for embedding the actual Go runtime in a signed iPhoneOS app.
// Embedded runtime uses explicit configuration and never starts SSH.
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

//export BeszelAgentStart
func BeszelAgentStart(config *C.char) *C.char {
	if config == nil {
		return C.CString("missing configuration")
	}
	if err := bridgecore.Start(C.GoString(config)); err != nil {
		return C.CString(err.Error())
	}
	return nil
}

//export BeszelAgentStop
func BeszelAgentStop() { bridgecore.Stop() }

//export BeszelAgentStatusJSON
func BeszelAgentStatusJSON() *C.char { return C.CString(bridgecore.StatusJSON()) }

//export BeszelAgentUpdateMetrics
func BeszelAgentUpdateMetrics(payload *C.char) *C.char {
	if payload == nil {
		return C.CString("missing metrics")
	}
	if err := bridgecore.UpdateMetrics(C.GoString(payload)); err != nil {
		return C.CString(err.Error())
	}
	return nil
}

func main() {}
