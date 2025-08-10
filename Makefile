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
	@cp $(VERSION_FILE) $(TEMP_VERSION_FILE)
	@sed -i "4s/.\{16\}/&$(shell git rev-parse HEAD)/" $(VERSION_FILE)

	@GOARCH=amd64 GOOS=linux go build -o ${BUILD_DIR}/${BINARY} internal/main.go

	@cp $(TEMP_VERSION_FILE) $(VERSION_FILE)
	@rm $(TEMP_VERSION_FILE)

	@setcap cap_net_raw=+ep ${BUILD_DIR}/${BINARY}

copy_static:
	@cp -r ./static ./${BUILD_DIR}

run: build
	${BUILD_DIR}/${BINARY} $(STATIC_DIR)

install: build
ifneq ($(shell uname), Linux)
	@echo "install only available on Linux platforms"
	exit 1
endif
	cp -f ./config/netmon.service /etc/systemd/system/netmon.service
	cp ./${BUILD_DIR}/${BINARY} /usr/local/bin
# Copy static records except for database.
	rsync -a --exclude="*.db" ./${STATIC_DIR} /srv/netmon/
	systemctl daemon-reload
	systemctl enable netmon
	systemctl start netmon
	@echo Installed netmon as service

uninstall:
ifneq ($(shell uname), Linux)
	@echo "uninstall only available on Linux platforms"
	exit 1
endif
	systemctl stop netmon
	rm /etc/systemd/system/netmon.service
	rm /usr/local/bin/${BINARY}
	rm -rf /srv/netmon
	systemctl daemon-reload
	@echo Uninstalled netmon service

clean:
	go clean
	rm -rf $(BUILD_DIR)/*
