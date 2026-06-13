.PHONY: build test vet run clean dist

BINARY=fileapi
DIST=dist

build:
	go build -o $(BINARY) ./cmd/fileapi

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
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(DIST)/fileapi-linux-amd64 ./cmd/fileapi
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(DIST)/fileapi-linux-arm64 ./cmd/fileapi
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(DIST)/fileapi-windows-amd64.exe ./cmd/fileapi
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(DIST)/fileapi-darwin-amd64 ./cmd/fileapi
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(DIST)/fileapi-darwin-arm64 ./cmd/fileapi
