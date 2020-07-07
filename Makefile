GO			?= go
GOOPTS		?=

pkgs	 	= ./...

BIN_DIR		?= ./bin
BUILD_DIR	?= ./build
DIST_DIR	?= ./dist
APPNAME		?= script_exporter
VERSION		?= $(shell cat ./VERSION 2> /dev/null)
COMMIT		?= $(shell git rev-parse --verify HEAD)
DATE		:= $(shell date +%FT%T%z)

GOPATH		?= $(shell go env GOPATH)
GO_LDFLAGS	+= -X main.name=$(APPNAME)
GO_LDFLAGS	+= -X main.version=$(VERSION)
GO_LDFLAGS	+= -X main.commit=$(COMMIT)
GO_LDFLAGS	+= -X main.date=$(DATE)
# strip debug info from binary
GO_LDFLAGS += -s -w

GO_LDFLAGS := -ldflags="$(GO_LDFLAGS)"

OS		:= $(if $(GOOS),$(GOOS),$(shell go env GOOS))
ARCH	:= $(if $(GOARCH),$(GOARCH),$(shell go env GOARCH))

.PHONY: all
all: test build

.PHONY: build
build:
	$(GO) build $(GO_LDFLAGS) -o $(BUILD_DIR)/$(APPNAME)-$(OS)-$(ARCH) -v ./...

.PHONY: test
test:
	mkdir -p $(BUILD_DIR)
	$(GO) test -v -race -coverprofile=$(BUILD_DIR)/test-coverage.out $(pkgs)

.PHONY: clean
clean:
	$(GO) clean
	rm -rf $(BUILD_DIR) $(DIST_DIR)

.PHONY: compile
compile:
	# FreeBDS
	GOOS=freebsd GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(APPNAME)-freebsd-amd64 -v ./...
	# MacOS
	GOOS=darwin GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(APPNAME)-darwin-amd64 -v ./...
	# Linux
	GOOS=linux GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(APPNAME)-linux-amd64 -v ./...
	# Windows
	GOOS=windows GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(APPNAME)-windows-amd64 -v ./...

.PHONY: dist
dist: clean compile
	mkdir -p $(DIST_DIR)

	tar -czvf $(BUILD_DIR)/$(APPNAME)-freebsd-amd64.tar $(BUILD_DIR)/$(APPNAME)-freebsd-amd64
	gzip $(BUILD_DIR)/$(APPNAME)-freebsd-amd64.tar > $(DIST_DIR)/$(APPNAME)-freebsd-amd64.tar.gz

	tar -czvf $(BUILD_DIR)/$(APPNAME)-darwin-amd64.tar $(BUILD_DIR)/$(APPNAME)-darwin-amd64
	gzip $(BUILD_DIR)/$(APPNAME)-darwin-amd64.tar > $(DIST_DIR)/$(APPNAME)-darwin-amd64.tar.gz

	tar -czvf $(BUILD_DIR)/$(APPNAME)-linux-amd64.tar $(BUILD_DIR)/$(APPNAME)-linux-amd64
	gzip $(BUILD_DIR)/$(APPNAME)-linux-amd64.tar > $(DIST_DIR)/$(APPNAME)-linux-amd64.tar.gz

	tar  -czvf $(BUILD_DIR)/$(APPNAME)-windows-amd64.tar $(BUILD_DIR)/$(APPNAME)-windows-amd64
	gzip $(BUILD_DIR)/$(APPNAME)-windows-amd64.tar > $(DIST_DIR)/$(APPNAME)-windows-amd64.tar.gz
