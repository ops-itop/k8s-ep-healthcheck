# k8s-ep-healthcheck
Healthcheck for custom endpoints. May be useful when use ingress-nginx for custom endpoints (ingress-nginx had remove annotation `upstream-max-fails` and `upstream-max-timeout`, See https://github.com/kubernetes/ingress-nginx/issues/4773).

## Usage

```
make         # build, push image and run
make build   # build image
make run     # deploy
make debug   # debug
```

Custom endpoint should be labeled with `type=external`

### Custom image

```
REGISTRY=your.registry.com PROJECT=myproject make
```

### Notify

Support wechat. edit `deploy/with-rbac.yaml` to set env or use make

```
CORPID=corpid CORPSECRET=corpsecret AGENTID=agentid TOUSER=@all make run
```

### Dashboard

default use ingress. use `make nodeport` to create nodeport service.

## Available config

|ENV | usage| default value|
|--|--|--|
|REGISTRY | custom registry | registry.cn-beijing.aliyuncs.com |
|PROJECT | custom project name |kubebase |
|IMAGE | custom image name |k8s-ep-healthcheck |
|TAG | custom image tag |latest |
|NS | kubernetes namespace for deploy | default |
|CORPID |wechat corpid | corpid |
|CORPSECRET |wechat corpsecret| corpsecret|
|AGENTID | wechat agentid |0|
|TOUSER | wechat touser |@all|
|LOGLEVEL | log level |debug|
|INTERVAL | check interval for endpoints|2|
|TIMEOUT | timeout for tcp check |500|
|RETRY | retry for tcp check |3|
|HOST | hostname for ingress |$(IMAGE).local|
