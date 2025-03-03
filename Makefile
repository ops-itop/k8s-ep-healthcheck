REGISTRY ?= registry.cn-beijing.aliyuncs.com
PROJECT ?= kubebase
APP ?= k8s-ep-healthcheck
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
LOGLEVEL ?= debug
INTERVAL ?= 2
TIMEOUT ?= 500
RETRY ?= 3
HOST ?= $(APP).local
WATCHTIMEOUT ?= 300

REPO = $(REGISTRY)/$(PROJECT)/$(APP)

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
		sed "s/__LOGLEVEL__/$(LOGLEVEL)/g" | \
		sed "s/__INTERVAL__/$(INTERVAL)/g" | \
		sed "s/__TIMEOUT__/$(TIMEOUT)/g" | \
		sed "s/__RETRY__/$(RETRY)/g" | \
		sed "s/__APP__/$(APP)/g" | \
		sed "s/__HOST__/$(HOST)/g" | \
		sed "s/__WATCHTIMEOUT__/$(WATCHTIMEOUT)/g" | \
		kubectl -n $(NS) apply -f -

patch:
	kubectl -n $(NS) patch deployment $(APP) -p '{"spec":{"template":{"spec":{"containers":[{"name":"$(APP)","env":[{"name":"RESTART_","value":"'$(shell date +%s)'"}]}]}}}}'

ep:
	kubectl -n $(NS) apply -f deploy/ep.yaml

debug: ep build patch

proxy:
	echo "/api/v1/namespaces/$(NS)/services/$(APP):8080/proxy/"
	kubectl proxy --address=0.0.0.0 --port=8080 --accept-hosts '.*'

nodeport:
	sed "s/__APP__/$(APP)/g" deploy/nodeport.yaml | \
		kubectl -n $(NS) apply -f -
clean:
	kubectl delete deployment $(APP)
