# gomodcheck

CLI tool that analyzes a Go repository and reports outdated dependencies.

## Requirements

- Go
- Git

## Build
```bash
make build
```

Cross-compile for Windows:
```bash
make build-windows
```

## Usage
```bash
gomodcheck <repo-url>
```
```bash
$ gomodcheck https://github.com/gin-gonic/gin

module:     github.com/gin-gonic/gin
go version: 1.23

updatable dependencies (3):

  golang.org/x/net                                   v0.25.0 -> v0.28.0
  golang.org/x/text                                  v0.15.0 -> v0.17.0
  google.golang.org/protobuf                         v1.34.1 -> v1.34.2
```

## How it works

1. Clones the repository into a temp directory (`git clone --depth=1`)
2. Parses `go.mod` for module name and Go version
3. Runs `go list -m -u all` to detect available updates
4. Prints the results