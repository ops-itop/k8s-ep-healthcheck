REGISTRY ?= 
PROJECT ?= op
IMAGE ?= client-go-incluster
TAG ?= latest

REPO = $(REGISTRY)/$(PROJECT)/$(IMAGE)

all: build push

build: build-local build-docker

build-local:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/app
build-docker:
	docker build -t $(REPO):$(TAG) .

push:
	docker tag $(REPO):$(TAG) $(REPO):latest
	docker push $(REPO):$(TAG)
	docker push $(REPO):latest
