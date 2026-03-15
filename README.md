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
go version: 1.25.0

updatable dependencies (10):

  github.com/creack/pty                              v1.1.9 -> v1.1.24
  github.com/golang/protobuf                         v1.5.0 -> v1.5.4
  github.com/google/gofuzz                           v1.0.0 -> v1.2.0
  github.com/jordanlewis/gcassert                    v0.0.0-20250430164644-389ef753e22e -> v0.0.0-20260313214104-ad3fae17affe
  github.com/klauspost/compress                      v1.17.6 -> v1.18.4
  github.com/rogpeppe/go-internal                    v1.10.0 -> v1.14.1
  github.com/stretchr/objx                           v0.5.2 -> v0.5.3
  github.com/yuin/goldmark                           v1.4.13 -> v1.7.16
  golang.org/x/mod                                   v0.33.0 -> v0.34.0
  golang.org/x/tools                                 v0.42.0 -> v0.43.0
```

## How it works

1. Clones the repository into a temp directory (`git clone --depth=1`)
2. Parses `go.mod` for module name and Go version
3. Runs `go list -m -u all` to detect available updates
4. Prints the results