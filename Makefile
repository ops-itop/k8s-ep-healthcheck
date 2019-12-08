REGISTRY ?= registry.cn-beijing.aliyuncs.com
PROJECT ?= op
IMAGE ?= k8s-ep-healthcheck
TAG ?= latest
NS ?= default

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
	kubectl -n $(NS) apply -f rbac.yaml
	kubectl create clusterrolebinding ep-healthcheck-rw --clusterrole=ep-healthcheck --serviceaccount=$(NS):ep-healthcheck

run:
	kubectl -n $(NS) run --rm -i demo --image=$(REPO):$(TAG) --image-pull-policy=Never --serviceaccount=ep-healthcheck

patch:
	kubectl -n $(NS) patch deployment demo -p '{"spec":{"template":{"spec":{"containers":[{"name":"demo","env":[{"name":"RESTART_","value":"'$(shell date +%s)'"}]}]}}}}'

ep:
	kubectl -n $(NS) apply -f ep.yaml

debug: ep build patch

clean:
	kubectl delete deployment demo
