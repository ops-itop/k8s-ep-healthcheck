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

rbac:
	kubectl create clusterrolebinding default-view --clusterrole=view --serviceaccount=default:default

run:
	kubectl run --rm -i demo --image=$(REPO):$(TAG) --image-pull-policy=Never

clean:
	kubectl delete deployment demo
