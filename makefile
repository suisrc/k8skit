.PHONY: start build

NOW = $(shell date -u '+%Y%m%d%I%M%S')

APP = kube-sidecar

dev: main

# 初始化mod
init:
	go mod init ${APP}

tidy:
	go mod tidy

build:
	CGO_ENABLED=0 go build -ldflags "-w -s" -o ./_out/$(APP) ./

# go env -w GOPROXY=https://proxy.golang.com.cn,direct
proxy:
	go env -w GO111MODULE=on
	go env -w GOPROXY=http://mvn.res.local/repository/go,direct
	go env -w GOSUMDB=sum.golang.google.cn

helm:
	helm -n default template deploy/chart > deploy/bundle.yml

main:
	go run main.go -local -debug

mapp:
	go run main.go -local -debug -xrt 2 \
	--token MQ8wDQYDVQQHEwZEYWxpYW4x \
	--injectAnnotation ksidecar/configmap \
	--injectDefaultKey sidecar.yml

test:
	_out/$(APP) version

hello:
	go run main.go hello

# go run main.go cert -path _out/cert -domain localhost -cname=localhost
cert:
	go run main.go cert -domain localhost

# https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-v1.19.2-linux-amd64.tar.gz
test-kube:
	TEST_ASSET_ETCD=_out/kubebuilder/bin/etcd \
	TEST_ASSET_KUBE_APISERVER=_out/kubebuilder/bin/kube-apiserver \
	TEST_ASSET_KUBECTL=_out/kubebuilder/bin/kubectl \
	go test -v -run TestCustom testdata/custom_test.go

test-custom:
	go test -v cmd/custom_test.go

