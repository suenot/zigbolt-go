// Package zigbolt provides Go bindings for the ZigBolt ultra-low-latency
// messaging library via cgo.
package zigbolt

/*
#cgo LDFLAGS: -lzigbolt
#include "zigbolt.h"
#include <stdlib.h>

// Gateway C callback that forwards to Go.
extern void goFragmentHandler(const uint8_t* data, uint32_t len, int32_t msg_type_id);
*/
import "C"

import (
	"errors"
	"sync"
	"unsafe"
)

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

// pollMu protects the global handler during a Poll call.
// ZigBolt's C callback is a plain function pointer with no user-data argument,
// so we route through a global variable guarded by this mutex.
var (
	pollMu      sync.Mutex
	pollHandler FragmentHandler
)

//export goFragmentHandler
func goFragmentHandler(data *C.uint8_t, length C.uint32_t, msgTypeId C.int32_t) {
	if pollHandler == nil {
		return
	}
	// Create a Go slice backed by C memory — valid only for the duration of this callback.
	buf := C.GoBytes(unsafe.Pointer(data), C.int(length))
	pollHandler(buf, int32(msgTypeId))
}

// Poll reads up to `limit` message fragments from the channel and invokes
// handler for each one. Returns the number of fragments read.
func (ch *IpcChannel) Poll(handler FragmentHandler, limit uint32) uint32 {
	if ch.handle == nil {
		return 0
	}
	pollMu.Lock()
	pollHandler = handler
	n := C.zigbolt_poll(ch.handle, C.zigbolt_fragment_handler_t(C.goFragmentHandler), C.uint32_t(limit))
	pollHandler = nil
	pollMu.Unlock()
	return uint32(n)
}

// Version returns the ZigBolt library version as (major, minor, patch).
func Version() (major, minor, patch uint32) {
	major = uint32(C.zigbolt_version_major())
	minor = uint32(C.zigbolt_version_minor())
	patch = uint32(C.zigbolt_version_patch())
	return
}
