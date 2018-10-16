.PHONY: build docker-build docker push clean

VERSION ?= $(shell git describe --tags --always --dirty)
LDFLAGS ?= -X main.version=$(VERSION)

# Default to build the Linux binary
build:
	GOOS=linux CGO_ENABLED=0 go build -o ./bin/tke-bridge-agent -ldflags "$(LDFLAGS)" ./cmd/

docker-build:
	docker run --rm -v $(shell pwd):/go/src/git.code.oa.com/tke/cni-bridge-agent \
		--workdir=/go/src/git.code.oa.com/tke/cni-bridge-agent \
		golang:1.10 make build

docker:
	@docker build -f scripts/Dockerfile.agent -t "ccr.ccs.tencentyun.com/tke-cni/cni-bridge-agent:$(VERSION)" .
	@echo "Built Docker image \"ccr.ccs.tencentyun.com/tke-cni/cni-bridge-agent:$(VERSION)\""

push:
	docker push "ccr.ccs.tencentyun.com/tke-cni/cni-bridge-agent:$(VERSION)"

clean:
	rm -rf bin
