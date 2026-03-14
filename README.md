# zigbolt-go

Go bindings for [ZigBolt](https://github.com/suenot/zigbolt) — an ultra-low-latency messaging library for high-frequency trading, built in Zig.

This package wraps ZigBolt's C-ABI shared library via cgo.

## Prerequisites

- Go 1.21+
- ZigBolt shared library (`libzigbolt.so` / `libzigbolt.dylib`) installed and visible to the linker
- The ZigBolt header is bundled in this package (`zigbolt.h`)

### Building ZigBolt from source

```bash
cd /path/to/zigbolt
zig build -Doptimize=ReleaseFast
# Install the shared library where the linker can find it, e.g.:
sudo cp zig-out/lib/libzigbolt.so /usr/local/lib/   # Linux
sudo cp zig-out/lib/libzigbolt.dylib /usr/local/lib/ # macOS
sudo ldconfig  # Linux only
```

## Installation

```bash
go get github.com/suenot/zigbolt-go
```

## Usage

### Publish

```go
package main

import (
    "log"
    zigbolt "github.com/suenot/zigbolt-go"
)

func main() {
    ch, err := zigbolt.CreateChannel("/my-channel", 1024*1024)
    if err != nil {
        log.Fatal(err)
    }
    defer ch.Close()

    ch.Publish([]byte("hello"), 1)
}
```

### Subscribe

```go
package main

import (
    "fmt"
    zigbolt "github.com/suenot/zigbolt-go"
)

func main() {
    ch, err := zigbolt.OpenChannel("/my-channel", 1024*1024)
    if err != nil {
        log.Fatal(err)
    }
    defer ch.Close()

    ch.Poll(func(data []byte, msgTypeId int32) {
        fmt.Printf("Received [type=%d]: %s\n", msgTypeId, data)
    }, 10)
}
```

### Version

```go
major, minor, patch := zigbolt.Version()
fmt.Printf("ZigBolt %d.%d.%d\n", major, minor, patch)
```

## API Reference

| Function | Description |
|---|---|
| `NewTransport(termLength, useHugepages, preFault)` | Create a transport with memory configuration |
| `CreateChannel(name, termLength)` | Create a new IPC channel (publisher) |
| `OpenChannel(name, termLength)` | Open an existing IPC channel (subscriber) |
| `(*IpcChannel).Publish(data, msgTypeId)` | Send a message |
| `(*IpcChannel).Poll(handler, limit)` | Receive up to `limit` messages |
| `(*IpcChannel).Close()` | Release channel resources |
| `(*Transport).Close()` | Release transport resources |
| `Version()` | Get library version (major, minor, patch) |

## Examples

```bash
# Terminal 1 — publisher
go run ./examples/publisher

# Terminal 2 — subscriber
go run ./examples/subscriber
```

## License

Same license as ZigBolt. See the [ZigBolt repository](https://github.com/suenot/zigbolt) for details.
