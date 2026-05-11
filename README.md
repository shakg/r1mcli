# r1mcli

A command-line utility for switching video resolution on R1M IP cameras.

## Overview

`r1mcli` connects to an R1M IP camera over TCP and sends a binary packet that toggles the camera's video output between **720p** and **1080p**. It communicates using a raw, undocumented protocol — no ONVIF or RTSP involved.

## Features

- Switch resolution between 720p and 1080p with a single command
- Save a camera's IP address as the default (persistent config)
- Override host, port, and timeout per invocation
- Dry-run mode to inspect the packet before sending
- Verbose output for debugging
- Zero external dependencies

## Installation

### From source

Requires Go 1.26+.

```bash
go build -o r1mcli .
```

The built binary is placed in the current directory. Optionally move it to a directory in your `PATH`.

### Pre-built binary

A pre-compiled binary is available in the repository (gitignored by default). If you obtained it from another source, make sure it's not stale — rebuild from source if you encounter issues.

## Usage

### Basic flow

```bash
# 1. Save your camera's IP as the default
./r1mcli --set-host 192.168.168.48

# 2. Switch to 1080p
./r1mcli --set-res 1080

# 3. Switch to 720p
./r1mcli --set-res 720
```

### All flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--set-res` | int | (required) | Set resolution: `720` or `1080` |
| `--set-host` | string | | Save a camera host IP as the default |
| `--host` | string | | Override camera host for this command |
| `--port` | int | | Override camera TCP port |
| `--timeout` | int | `5` | Connection/response timeout in seconds |
| `-v` | bool | `false` | Verbose output |
| `--dry-run` | bool | `false` | Print packet without sending |

### Host and port resolution

The tool resolves host and port in this order of priority:

1. **Flag** (`--host` / `--port`) — highest priority
2. **Config file** — saved via `--set-host`
3. **Hardcoded default** — `192.168.168.48:37256`

### Examples

```bash
# Save host and set resolution in one command
./r1mcli --set-host 192.168.168.48 --set-res 1080

# Use an explicit host (no config needed)
./r1mcli --host 192.168.168.50 --set-res 720

# Specify a non-default port
./r1mcli --host 192.168.168.48 --port 9999 --set-res 1080

# Increase timeout for slow networks
./r1mcli --set-res 1080 --timeout 10

# Dry run with verbose output (see the packet, don't send it)
./r1mcli --set-res 1080 --dry-run -v
```

### Verbose dry run output

```
Host: 192.168.168.48
Port: 37256
Timeout: 5s
Resolution: 1080p
Packet (24 bytes): 5566aabb0109000000d70084a31fec99010280073804001e1ec6195e42
```

## Configuration

The config file is stored at:

```
~/.config/r1mcli/config.json
```

It is created automatically the first time you run `--set-host`. If the file does not exist, defaults are used transparently.

```json
{
  "host": "192.168.168.48",
  "port": 37256
}
```

## How it works

1. The tool connects to the camera via TCP on the specified port (default `37256`).
2. It sends a pre-crafted binary packet encoded as a hex string.
3. The camera responds with its own binary packet.
4. The tool prints the response (also as hex).

The packets are hardcoded constants in the source. They were discovered through reverse engineering the camera's proprietary protocol.

### Packet constants

| Command | Hex |
|---------|-----|
| 720p → 1080p | `5566aabb0109000000d70084a31fec99010280073804001e1ec6195e42` |
| 1080p → 720p | `5566aabb0109000000d900848990fa9301020005d002001e1e9df45edd` |

## Project structure

```
.
├── main.go        # Single-file application (~300 lines)
├── go.mod         # Go module definition
├── .gitignore     # Ignores the built binary
└── README.md      # This file
```

## License

This project is provided as-is.
