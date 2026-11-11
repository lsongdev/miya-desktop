APP_VERSION ?=
BIN_DIR ?= bin
MYGO ?= go tool mygo

.PHONY: configure build build-macos build-macos-arm64 build-windows run dev clean install test version

configure:
	go run ./scripts/configure -version "$(APP_VERSION)"

build: configure
	$(MYGO) build -o $(BIN_DIR)

build-macos: configure
	$(MYGO) build -platform darwin/universal -o $(BIN_DIR)

build-macos-arm64: configure
	$(MYGO) build -platform darwin/arm64 -o $(BIN_DIR)

build-windows: configure
	$(MYGO) build -platform windows/amd64 -o $(BIN_DIR)

install:
	go mod download

dev:
	$(MYGO) dev

run: dev

test:
	go test ./...
	go vet ./...

version:
	@awk -F'"' '/"version"/ { print $$4; exit }' mygo.json

clean:
	rm -rf $(BIN_DIR) .mygo
