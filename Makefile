# Network Monitor Makefile

COMMIT=$(shell git rev-parse HEAD)
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)

BINARY = netmon
BUILD_DIR = build
STATIC_DIR = ${BUILD_DIR}/static
SOURCES := $(shell find $(SOURCEDIR) -wholename '**/*.go')
VERSION_FILE = "./internal/constants/version.go"
TEMP_VERSION_FILE = "temp_version.go"

build: copy_static
	cp $(VERSION_FILE) $(TEMP_VERSION_FILE)
	sed -i "4s/.\{16\}/&$(shell git rev-parse HEAD)/" $(VERSION_FILE)

	GOARCH=amd64 GOOS=linux go build -o ${BUILD_DIR}/${BINARY} internal/main.go
	sudo setcap cap_net_raw=+ep ${BUILD_DIR}/${BINARY}

	cp $(TEMP_VERSION_FILE) $(VERSION_FILE)
	rm $(TEMP_VERSION_FILE)

copy_static:
	cp -r ./static ./${BUILD_DIR}

run: build
	${BUILD_DIR}/${BINARY} $(STATIC_DIR)

clean:
	go clean
	rm -rf $(BUILD_DIR)/*
