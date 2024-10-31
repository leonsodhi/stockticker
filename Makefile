ARCH=amd64
EXECUTABLE=stockticker

GO_DOCKER_IMAGE=golang:1.23.2
GO ?= go
GOOS ?=

DOCKER_REPO ?= leonsodhi
DOCKER_IMAGE_NAME ?= stockticker
DOCKER_IMAGE_TAG ?=

.PHONY: build
build:
	CGO_ENABLED=0 GOOS=$(GOOS) $(GO) build -o bin/$(EXECUTABLE) cmd/$(EXECUTABLE)/*.go

.PHONY: build-in-docker
build-in-docker: clean
	docker run -e GOPATH=/gopath/src/$(EXECUTABLE)/docker-cache -v `pwd`:/gopath/src/$(EXECUTABLE) $(GO_DOCKER_IMAGE) bash -c 'cd /gopath/src/$(EXECUTABLE) && go mod download && GOOS=$(GOOS) make build'

.PHONY: clean
clean:
	rm -rf bin/*

.PHONY: test
test:
	CGO_ENABLED=0 GOOS=$(GOOS) $(GO) test -v $$(go list -e ./... | grep '^$(EXECUTABLE)')

.PHONY: test-in-docker
test-in-docker:
	docker run -e GOPATH=/gopath/src/$(EXECUTABLE)/docker-cache -v `pwd`:/gopath/src/$(EXECUTABLE) $(GO_DOCKER_IMAGE) bash -c 'cd /gopath/src/$(EXECUTABLE) && go mod download && make test'

.PHONY: docker-image
docker-image:
	docker build -t "$(DOCKER_REPO)/$(DOCKER_IMAGE_NAME):latest" .
ifneq ($(strip $(DOCKER_IMAGE_TAG)),)
	docker tag "$(DOCKER_REPO)/$(DOCKER_IMAGE_NAME):latest" "$(DOCKER_REPO)/$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)"
endif

.PHONY: docker-push
docker-push: docker-image
	docker push "$(DOCKER_REPO)/$(DOCKER_IMAGE_NAME):latest"
ifneq ($(strip $(DOCKER_IMAGE_TAG)),)
	docker push "$(DOCKER_REPO)/$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)"
endif
