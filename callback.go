package zigbolt

// This file contains the cgo-exported poll callback. It lives in its own
// file because a file that uses //export may only have declarations (no
// definitions) in its C preamble — the static C helpers live in zigbolt.go.

/*
#include <stdint.h>
*/
import "C"

import (
	"runtime/cgo"
	"unsafe"
)

//export goFragmentHandler
func goFragmentHandler(data *C.uint8_t, length C.uint32_t, msgTypeId C.int32_t, handle C.uintptr_t) {
	if handle == 0 {
		return
	}
	h := cgo.Handle(handle)
	fn, ok := h.Value().(FragmentHandler)
	if !ok || fn == nil {
		return
	}
	var buf []byte
	if data != nil && length > 0 {
		// Copy out of the shared-memory region: the C pointer is only
		// valid for the duration of this callback.
		buf = C.GoBytes(unsafe.Pointer(data), C.int(length))
	}
	fn(buf, int32(msgTypeId))
}
