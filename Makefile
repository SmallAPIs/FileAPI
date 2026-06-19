.PHONY: build test vet run clean dist

BINARY=fileapi
DIST=dist
LDFLAGS=-s -w
BUILD_FLAGS=-trimpath -ldflags="$(LDFLAGS)"

build:
	CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(BINARY) ./cmd/fileapi

test:
	go test ./...

vet:
	go vet ./...

run:
	go run ./cmd/fileapi serve

clean:
	rm -rf $(DIST) $(BINARY) $(BINARY).exe

dist:
	mkdir -p $(DIST)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o $(DIST)/fileapi-linux-amd64 ./cmd/fileapi
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(BUILD_FLAGS) -o $(DIST)/fileapi-linux-arm64 ./cmd/fileapi
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o $(DIST)/fileapi-windows-amd64.exe ./cmd/fileapi
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o $(DIST)/fileapi-darwin-amd64 ./cmd/fileapi
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(BUILD_FLAGS) -o $(DIST)/fileapi-darwin-arm64 ./cmd/fileapi
