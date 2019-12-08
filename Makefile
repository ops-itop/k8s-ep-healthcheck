REGISTRY ?= registry.cn-beijing.aliyuncs.com
PROJECT ?= kubebase
IMAGE ?= k8s-ep-healthcheck
TAG ?= latest
NS ?= default

REPO = $(REGISTRY)/$(PROJECT)/$(IMAGE)

all: build push run

build: build-local build-docker

build-local:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/app
build-docker:
	docker build -t $(REPO):$(TAG) .

push:
	docker tag $(REPO):$(TAG) $(REPO):latest
	docker push $(REPO):$(TAG)
	docker push $(REPO):latest

run:
	sed "s/__NAMESPACE__/$(NS)/g" deploy/with-rbac.yaml |sed "s#__IMAGE__#$(REPO):$(TAG)#g" | kubectl -n $(NS) apply -f -

patch:
	kubectl -n $(NS) patch deployment k8s-ep-healthcheck -p '{"spec":{"template":{"spec":{"containers":[{"name":"k8s-ep-healthcheck","env":[{"name":"RESTART_","value":"'$(shell date +%s)'"}]}]}}}}'

ep:
	kubectl -n $(NS) apply -f ep.yaml

debug: ep build patch

clean:
	kubectl delete deployment k8s-ep-healthcheck
