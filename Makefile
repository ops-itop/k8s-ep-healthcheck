REGISTRY ?= registry.cn-beijing.aliyuncs.com
PROJECT ?= kubebase
IMAGE ?= k8s-ep-healthcheck
TAG ?= latest
NS ?= default
VERSION ?= $(shell git describe --always --dirty)
GITHASH ?= $(shell git rev-parse HEAD)
GOVERSION ?= $(shell go version)
BUILDTIME ?= $(shell date)
FLAGS = -X 'main.goVersion=$(GOVERSION)' -X 'main.gitHash=$(GITHASH)' -X 'main.buildTime=$(BUILDTIME)' -X 'main.version=$(VERSION)'

# 配置信息
CORPID ?= corpid
CORPSECRET ?= corpsecret
AGENTID ?= 0
TOUSER ?= @all

REPO = $(REGISTRY)/$(PROJECT)/$(IMAGE)

all: build push run patch

build: build-local build-docker

build-local:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(FLAGS)" -o bin/ep-healthcheck cmd/ep-healthcheck/main.go
build-docker:
	docker build -t $(REPO):$(TAG) .

push:
	docker tag $(REPO):$(TAG) $(REPO):latest
	docker push $(REPO):$(TAG)
	docker push $(REPO):latest

run:
	sed "s/__NAMESPACE__/$(NS)/g" deploy/with-rbac.yaml | \
		sed "s#__IMAGE__#$(REPO):$(TAG)#g" | \
		sed "s/__CORPID__/$(CORPID)/g" | \
		sed "s/__CORPSECRET__/$(CORPSECRET)/g" | \
		sed "s/__AGENTID__/$(AGENTID)/g" | \
		sed "s/__TOUSER__/$(TOUSER)/g" | \
		kubectl -n $(NS) apply -f -

patch:
	kubectl -n $(NS) patch deployment k8s-ep-healthcheck -p '{"spec":{"template":{"spec":{"containers":[{"name":"k8s-ep-healthcheck","env":[{"name":"RESTART_","value":"'$(shell date +%s)'"}]}]}}}}'

ep:
	kubectl -n $(NS) apply -f ep.yaml

debug: ep build patch

clean:
	kubectl delete deployment k8s-ep-healthcheck
