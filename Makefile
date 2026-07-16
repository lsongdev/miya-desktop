APP_NAME := miya-desktop
BIN_DIR  := bin
APP_VERSION ?= $(shell TZ=Asia/Shanghai date +%y.%m.%d)
MAC_APP ?= $(BIN_DIR)/Miya.app
MAC_PLIST = $(MAC_APP)/Contents/Info.plist
MAC_GOENV ?=
GO_LDFLAGS := -X main.appVersion=$(APP_VERSION)

export VITE_APP_VERSION := $(APP_VERSION)

.PHONY: build build-macos build-macos-arm64 build-windows run dev clean install generate-icons version

build:
	cd frontend && npm run build
	mkdir -p $(BIN_DIR)
	go build -ldflags "$(GO_LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME) .

build-macos:
	cd frontend && npm install && npm run build
	mkdir -p $(MAC_APP)/Contents/MacOS $(MAC_APP)/Contents/Resources
	$(MAC_GOENV) go build -ldflags "$(GO_LDFLAGS)" -o $(MAC_APP)/Contents/MacOS/$(APP_NAME) .
	cp build/darwin/Info.plist $(MAC_PLIST)
	/usr/libexec/PlistBuddy -c "Set :CFBundleVersion $(APP_VERSION)" $(MAC_PLIST)
	/usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString $(APP_VERSION)" $(MAC_PLIST)
	cp build/darwin/icons.icns $(MAC_APP)/Contents/Resources/icons.icns
	codesign --force --deep --sign - $(MAC_APP)

build-macos-arm64:
	$(MAKE) build-macos APP_VERSION=$(APP_VERSION) MAC_APP=$(BIN_DIR)/Miya-arm64.app MAC_GOENV="GOOS=darwin GOARCH=arm64 CGO_ENABLED=1"

build-windows:
	cd frontend && npm install && npm run build
	mkdir -p $(BIN_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags "$(GO_LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME).exe .

install:
	cd frontend && npm install

dev:
	wails3 dev -config ./build/config.yml

run: build
	./$(BIN_DIR)/$(APP_NAME)

generate-icons:
	wails3 generate icons \
		-input build/appicon.png \
		-macfilename build/darwin/icons.icns \
		-windowsfilename build/windows/icon.ico \
		-macassetdir build/darwin

version:
	@printf '%s\n' "$(APP_VERSION)"

clean:
	rm -rf $(BIN_DIR) frontend/dist
