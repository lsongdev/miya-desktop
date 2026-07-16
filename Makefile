APP_NAME := miya-desktop
BIN_DIR  := bin

.PHONY: build build-macos build-windows run dev clean install generate-icons

build:
	cd frontend && npm run build
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_NAME) .

build-macos:
	cd frontend && npm install && npm run build
	mkdir -p $(BIN_DIR)/Miya.app/Contents/{MacOS,Resources}
	go build -o $(BIN_DIR)/Miya.app/Contents/MacOS/$(APP_NAME) .
	cp build/darwin/icons.icns $(BIN_DIR)/Miya.app/Contents/Resources/icons.icns

build-windows:
	cd frontend && npm install && npm run build
	mkdir -p $(BIN_DIR)
	GOOS=windows GOARCH=amd64 go build -o $(BIN_DIR)/$(APP_NAME).exe .

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

clean:
	rm -rf $(BIN_DIR) frontend/dist
