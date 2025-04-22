.PHONY: test build container push clean

REGISTRY_NAME=cr.yandex/crp9ftr22d26age3hulg
IMAGE_NAME=csi-s3
IMAGE_NAME2=yandex-cloud/csi-s3/csi-s3-driver
VERSION ?= 0.42.1
IMAGE_TAG=$(REGISTRY_NAME)/$(IMAGE_NAME):$(VERSION)
TEST_IMAGE_TAG=$(IMAGE_NAME):test

build:
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o _output/s3driver ./cmd/s3driver
test:
	docker build -t $(TEST_IMAGE_TAG) -f test/Dockerfile .
	docker run --rm --privileged -v $(PWD):/build --device /dev/fuse $(TEST_IMAGE_TAG)
container:
	docker build -t $(IMAGE_TAG) .
push: container
	docker tag $(IMAGE_TAG) $(REGISTRY_NAME)/$(IMAGE_NAME):latest
	docker tag $(IMAGE_TAG) $(REGISTRY_NAME)/$(IMAGE_NAME2):$(VERSION)
	docker tag $(IMAGE_TAG) $(REGISTRY_NAME)/$(IMAGE_NAME2):latest
	docker push $(IMAGE_TAG)
	docker push $(REGISTRY_NAME)/$(IMAGE_NAME)
clean:
	go clean -r -x
	-rm -rf _output