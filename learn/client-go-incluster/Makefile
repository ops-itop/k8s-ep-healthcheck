REGISTRY ?= 
IMAGE ?= op/client-go-incluster
TAG ?= latest

REPO = $(REGISTRY)/$(IMAGE):$(TAG)

all: build push

build: build-local build-docker

build-local:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/app
build-docker:
	docker build -t $(REPO) .

push:
	docker tag $(REPO) $(REGISTRY)/$(IMAGE):latest
	docker push $(REPO)
	docker push $(REGISTRY)/$(IMAGE):latest
