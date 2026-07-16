APP_NAME := miya-desktop
BIN_DIR  := bin
APP_VERSION ?= $(shell TZ=Asia/Shanghai date '+%y.%m.%d' | awk -F. '{printf "%d.%d.%d", $$1, $$2, $$3}')
MAC_APP ?= $(BIN_DIR)/Miya.app
MAC_PLIST = $(MAC_APP)/Contents/Info.plist
MAC_GOENV ?=
WAILS3 ?= go tool wails3
GO_LDFLAGS := -X main.appVersion=$(APP_VERSION)
WINDOWS_LDFLAGS := $(GO_LDFLAGS) -H windowsgui
WINDOWS_SYSO := wails_windows_amd64.syso
WINDOWS_INFO := build/windows/info.generated.json
WINDOWS_MANIFEST := build/windows/wails.generated.exe.manifest

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
	node scripts/windows-info.mjs $(WINDOWS_INFO) $(APP_VERSION)
	node scripts/windows-manifest.mjs $(WINDOWS_MANIFEST) $(APP_VERSION)
	$(WAILS3) generate syso -arch amd64 -icon build/windows/icon.ico -manifest $(WINDOWS_MANIFEST) -info $(WINDOWS_INFO) -out $(WINDOWS_SYSO)
	GOOS=windows GOARCH=amd64 go build -ldflags "$(WINDOWS_LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME).exe .
	rm -f $(WINDOWS_SYSO) $(WINDOWS_INFO) $(WINDOWS_MANIFEST)

install:
	cd frontend && npm install

dev:
	$(WAILS3) dev -config ./build/config.yml

run: build
	./$(BIN_DIR)/$(APP_NAME)

generate-icons:
	$(WAILS3) generate icons \
		-input build/appicon.png \
		-macfilename build/darwin/icons.icns \
		-windowsfilename build/windows/icon.ico \
		-macassetdir build/darwin

version:
	@printf '%s\n' "$(APP_VERSION)"

clean:
	rm -rf $(BIN_DIR) frontend/dist
