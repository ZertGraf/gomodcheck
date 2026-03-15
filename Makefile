BINARY = gomodcheck

.PHONY: build build-windows test lint clean

build:
	go build -o $(BINARY) .

build-windows:
	GOOS=windows GOARCH=amd64 go build -o $(BINARY).exe .

test:
	go test -v -count=1 ./...

lint:
	go vet ./...

clean:
	rm -f $(BINARY) $(BINARY).exe