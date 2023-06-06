PACKAGES=$(shell go list ./...)
BUILDDIR?=$(CURDIR)/build

COMMIT_HASH := $(shell git rev-parse --short HEAD)
COMMIT_DATE := $(shell git show -s --format=%cs HEAD)
LD_FLAGS = -X github.com/DOIDFoundation/node/version.Commit=$(COMMIT_HASH) \
		-X github.com/DOIDFoundation/node/version.Date=$(COMMIT_DATE)
BUILD_FLAGS = -mod=readonly -ldflags "$(LD_FLAGS)"

all: build test
.PHONY: all

BUILD_TARGETS := doidnode

$(BUILD_TARGETS): go.sum $(BUILDDIR)/

$(BUILDDIR)/:
	mkdir -p $(BUILDDIR)/

build: $(BUILD_TARGETS)
.PHONY: build

doidnode:
	cd ./cmd/$@ && go build -mod=readonly $(BUILD_FLAGS) -o $(BUILDDIR)/ ./...
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
