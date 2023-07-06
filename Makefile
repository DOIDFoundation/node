PACKAGES=$(shell go list ./...)
BUILDDIR?=$(CURDIR)/build

COMMIT_HASH := $(shell git rev-parse --short HEAD)
COMMIT_DATE := $(shell git show -s --format=%cs HEAD)
LD_FLAGS = -X github.com/DOIDFoundation/node/version.Commit=$(COMMIT_HASH) \
		-X github.com/DOIDFoundation/node/version.Date=$(COMMIT_DATE)
BUILD_FLAGS = -mod=readonly -ldflags "$(LD_FLAGS)"

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


PLATFORMS=darwin-amd64 darwin-arm64 linux-386 linux-amd64 linux-arm64 windows-386 windows-amd64
build_all:
	$(foreach PLATFORM, $(PLATFORMS), GOOS=$(word 1,$(subst -, ,${PLATFORM})) GOARCH=$(word 2,$(subst -, ,${PLATFORM})) make doidnode;)
	$(foreach PLATFORM, $(PLATFORMS), zip -j build/doidnode-$(PLATFORM) build/$(PLATFORM)/doidnode*;)
	$(foreach PLATFORM, darwin-amd64 darwin-arm64, zip -j build/doidnode-$(PLATFORM) build/doidnode.command;)

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
