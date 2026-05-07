VERSION = v0.0.1

GOOS = linux
ARCH = amd64

REGISTRY = harbor.mkaas.edgecenter.online/docker/ec-csi
IMAGE := $(REGISTRY):$(VERSION)

BINARY := bin/ec-csi-plugin

build:
	GOOS=$(GOOS) GOARCH=$(ARCH) CGO_ENABLED=0 \
	go build -o $(BINARY) ./cmd/main.go

image-build: build
	docker build \
		--platform $(GOOS)/$(ARCH) \
		-t $(IMAGE) .

trivy: image-build
	trivy image \
		--scanners vuln \
		--format table \
		--ignore-unfixed \
		--pkg-types os,library \
		--severity CRITICAL,HIGH,MEDIUM \
		$(IMAGE)