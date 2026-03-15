BINARY = gomodcheck

.PHONY: build build-windows lint clean

build:
	go build -o $(BINARY) .

build-windows:
	GOOS=windows GOARCH=amd64 go build -o $(BINARY).exe .

lint:
	go vet ./...

clean:
	rm -f $(BINARY) $(BINARY).exe