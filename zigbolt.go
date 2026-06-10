// Package zigbolt provides Go bindings for the ZigBolt ultra-low-latency
// messaging library via cgo.
package zigbolt

/*
#cgo LDFLAGS: -L${SRCDIR}/../zigbolt/zig-out/lib -L${SRCDIR}/../../zig-out/lib -lzigbolt
#cgo LDFLAGS: -Wl,-rpath,${SRCDIR}/../zigbolt/zig-out/lib -Wl,-rpath,${SRCDIR}/../../zig-out/lib
#include "zigbolt.h"
#include <stdlib.h>
#include <stdint.h>

// Prototype of the Go callback exported from callback.go. This must match
// the cgo-generated declaration EXACTLY (cgo generates a non-const
// `uint8_t*` for a `*C.uint8_t` parameter — declaring `const uint8_t*`
// here would be a compile error: "conflicting types for 'goFragmentHandler'").
extern void goFragmentHandler(uint8_t* data, uint32_t len, int32_t msg_type_id, uintptr_t handle);

// ZigBolt's fragment callback is a bare function pointer with no user-data
// argument, so the per-call context (a runtime/cgo.Handle) is carried in a
// C thread-local instead of a Go package-global. zigbolt_poll invokes the
// callback synchronously on the calling thread, so:
//   - polls on independent channels from different goroutines never share
//     state (each OS thread has its own slot), and
//   - a nested/reentrant Poll from inside a handler simply saves and
//     restores the slot like a stack frame.
static _Thread_local uintptr_t zb_current_handle;

static void zb_trampoline(const uint8_t* data, uint32_t len, int32_t msg_type_id) {
	goFragmentHandler((uint8_t*)data, len, msg_type_id, zb_current_handle);
}

static uint32_t zb_poll(void* handle, uintptr_t go_handle, uint32_t limit) {
	uintptr_t prev = zb_current_handle;
	zb_current_handle = go_handle;
	uint32_t n = zigbolt_poll(handle, zb_trampoline, limit);
	zb_current_handle = prev;
	return n;
}
*/
import "C"

import (
	"errors"
	"runtime/cgo"
	"unsafe"
)

// BindingVersion is the version of these Go bindings.
const BindingVersion = "0.2.1"

// DefaultTermLength is the default ring-buffer term length (1 MiB),
// unified across all ZigBolt bindings.
const DefaultTermLength uint32 = 1 << 20

// FragmentHandler is called for each message fragment received during Poll.
type FragmentHandler func(data []byte, msgTypeId int32)

// Transport manages the shared-memory transport configuration.
type Transport struct {
	handle unsafe.Pointer
}

// NewTransport creates a new transport with the given term buffer length.
// useHugepages and preFault control OS-level memory optimizations.
func NewTransport(termLength uint32, useHugepages, preFault bool) (*Transport, error) {
	var hp, pf C.uint8_t
	if useHugepages {
		hp = 1
	}
	if preFault {
		pf = 1
	}
	h := C.zigbolt_transport_create(C.uint32_t(termLength), hp, pf)
	if h == nil {
		return nil, errors.New("zigbolt: failed to create transport")
	}
	return &Transport{handle: h}, nil
}

// Close releases the transport resources.
func (t *Transport) Close() {
	if t.handle != nil {
		C.zigbolt_transport_destroy(t.handle)
		t.handle = nil
	}
}

// IpcChannel represents a shared-memory IPC channel for publishing or subscribing.
type IpcChannel struct {
	handle unsafe.Pointer
}

// CreateChannel creates a new IPC channel with the given name and term buffer length.
// The creator owns the shared memory segment.
func CreateChannel(name string, termLength uint32) (*IpcChannel, error) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	h := C.zigbolt_ipc_create(cName, C.uint32_t(termLength))
	if h == nil {
		return nil, errors.New("zigbolt: failed to create IPC channel")
	}
	return &IpcChannel{handle: h}, nil
}

// OpenChannel opens an existing IPC channel by name.
func OpenChannel(name string, termLength uint32) (*IpcChannel, error) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	h := C.zigbolt_ipc_open(cName, C.uint32_t(termLength))
	if h == nil {
		return nil, errors.New("zigbolt: failed to open IPC channel")
	}
	return &IpcChannel{handle: h}, nil
}

// Close releases the IPC channel resources.
func (ch *IpcChannel) Close() {
	if ch.handle != nil {
		C.zigbolt_ipc_destroy(ch.handle)
		ch.handle = nil
	}
}

// Publish sends a message on this IPC channel.
// Returns nil on success or an error if the publish failed.
func (ch *IpcChannel) Publish(data []byte, msgTypeId int32) error {
	if ch.handle == nil {
		return errors.New("zigbolt: channel is closed")
	}
	var dataPtr *C.uint8_t
	if len(data) > 0 {
		dataPtr = (*C.uint8_t)(unsafe.Pointer(&data[0]))
	}
	rc := C.zigbolt_publish(ch.handle, dataPtr, C.uint32_t(len(data)), C.int32_t(msgTypeId))
	if rc < 0 {
		return errors.New("zigbolt: publish failed")
	}
	return nil
}

// Poll reads up to `limit` message fragments from the channel and invokes
// handler for each one. Returns the number of fragments read.
//
// The handler is passed per call via a runtime/cgo.Handle carried in a C
// thread-local, so concurrent Polls on independent channels do not block
// each other and a Poll from inside a handler (reentrant poll) is safe.
func (ch *IpcChannel) Poll(handler FragmentHandler, limit uint32) uint32 {
	if ch.handle == nil || handler == nil {
		return 0
	}
	h := cgo.NewHandle(handler)
	defer h.Delete()
	n := C.zb_poll(ch.handle, C.uintptr_t(h), C.uint32_t(limit))
	return uint32(n)
}

// Version returns the ZigBolt library version as (major, minor, patch).
func Version() (major, minor, patch uint32) {
	major = uint32(C.zigbolt_version_major())
	minor = uint32(C.zigbolt_version_minor())
	patch = uint32(C.zigbolt_version_patch())
	return
}
