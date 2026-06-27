BINARY=gtx
MODULE=github.com/victorschmidt21/gtx
VERSION?=dev
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/gtx

test:
	go test ./...

install:
	go install $(LDFLAGS) ./cmd/gtx

cross-compile:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/gtx-windows-amd64.exe ./cmd/gtx
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o dist/gtx-linux-amd64     ./cmd/gtx
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o dist/gtx-darwin-arm64    ./cmd/gtx

clean:
	rm -f $(BINARY)
	rm -rf dist/

.PHONY: build test install cross-compile clean
