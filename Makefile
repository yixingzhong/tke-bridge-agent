.PHONY: build docker-build docker push clean

VERSION ?= $(shell git describe --tags --always --dirty)
LDFLAGS ?= -X main.version=$(VERSION)

PKG := github.com/qyzhaoxun/tke-bridge-agent

BINARY ?= tke-bridge-agent

CONTAINER_BUILD_PATH ?= /go/src/$(PKG)
BIN_PATH ?= ./bin/$(BINARY)

REGISTRY ?= ccr.ccs.tencentyun.com/tkeimages
IMAGE ?= $(REGISTRY)/$(BINARY)

# Default to build the Linux binary
build:
	GOOS=linux CGO_ENABLED=0 go build -o $(BIN_PATH) -ldflags "$(LDFLAGS)" ./cmd/

docker-build:
	docker run --rm -v $(shell pwd):$(CONTAINER_BUILD_PATH) \
		--workdir=$(CONTAINER_BUILD_PATH) \
		golang:1.10 make build

docker: docker-build
	@docker build -f scripts/Dockerfile.agent -t "$(IMAGE):$(VERSION)" .
	@echo "Built Docker image \"$(IMAGE):$(VERSION)\""

push: docker
	docker push "$(IMAGE):$(VERSION)"

clean:
	rm -rf bin
