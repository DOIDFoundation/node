PACKAGES=$(shell go list ./...)
BUILDDIR?=$(CURDIR)/build

COMMIT_HASH := $(shell git rev-parse --short HEAD)
COMMIT_DATE := $(shell git show -s --format=%cs HEAD)
LD_FLAGS = -X github.com/DOIDFoundation/node/version.Commit=$(COMMIT_HASH) \
		-X github.com/DOIDFoundation/node/version.Date=$(COMMIT_DATE)
BUILD_FLAGS ?= -mod=readonly -ldflags "$(LD_FLAGS)"

# for cross compiling, TARGETOS and TARGETARCH can be set by docker
# GOOS and GOARCH values can be found by command: go tool dist list
TARGETOS ?=
TARGETARCH ?=
GOOS ?= $(TARGETOS)
GOARCH ?= $(TARGETARCH)

ifneq ($(strip $(GOOS)),)
	GO_BUILD_ENV=GOOS=$(GOOS)
	BUILDDIR:=$(BUILDDIR)/$(GOOS)
endif

ifneq ($(strip $(GOARCH)),)
	GO_BUILD_ENV+=GOARCH=$(GOARCH)
	ifneq ($(strip $(GOOS)),)
		BUILDDIR:=$(BUILDDIR)-$(GOARCH)
	else
		BUILDDIR:=$(BUILDDIR)/$(GOARCH)
	endif
endif

all: build test
.PHONY: all

BUILD_TARGETS := doidnode

$(BUILD_TARGETS): go.sum $(BUILDDIR)/

$(BUILDDIR)/:
	mkdir -p $(BUILDDIR)/

build: $(BUILD_TARGETS)
.PHONY: build

PLATFORMS_linux=linux-386 linux-amd64 linux-arm64
PLATFORMS_windows=windows-386 windows-amd64
PLATFORMS_macos=darwin-amd64 darwin-arm64
PLATFORMS=$(PLATFORMS_macos) $(PLATFORMS_linux) $(PLATFORMS_windows)

$(PLATFORMS)::
	GOOS=$(word 1,$(subst -, ,$@)) GOARCH=$(word 2,$(subst -, ,$@)) make doidnode
	zip -j build/doidnode-$@ build/$@/doidnode*
.PHONY: $(PLATFORMS)

$(PLATFORMS_macos)::
	zip -j build/doidnode-$@ build/doidnode.command

build_all: $(PLATFORMS)
.PHONY: build_all

linux: $(PLATFORMS_linux)
.PHONY: linux

macos: $(PLATFORMS_macos)
.PHONY: macos

windows: $(PLATFORMS_windows)
.PHONY: windows

doidnode:
	cd ./cmd/$@ && $(GO_BUILD_ENV) go build $(BUILD_FLAGS) -o $(BUILDDIR)/ ./...
.PHONY: doidnode

test:
	@echo "--> Running go test"
	@go test -p 1 $(PACKAGES) -tags deadlock
.PHONY: test

clean:
	rm -rf $(BUILDDIR)/

go.sum: go.mod
	@echo "Ensure dependencies have not been modified ..." >&2
	@go mod verify
	@go mod tidy
