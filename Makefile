BIN       := bin/devourer-server
LDFLAGS   := -ldflags="-s -w"
BUILD_CMD := CGO_ENABLED=0 go build $(LDFLAGS)

.PHONY: build build-windows build-linux build-linux-arm64 build-macos-arm docker clean tidy

build:
	$(BUILD_CMD) -o $(BIN) ./cmd/server

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BIN).exe ./cmd/server

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BIN)-linux-amd64 ./cmd/server

build-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BIN)-linux-arm64 ./cmd/server

build-macos-arm:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BIN)-macos-arm64 ./cmd/server

docker:
	docker build -t devourer-server .

tidy:
	go mod tidy

clean:
	rm -rf bin/
