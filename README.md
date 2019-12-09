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
