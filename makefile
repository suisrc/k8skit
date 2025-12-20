.PHONY: start build

NOW = $(shell date -u '+%Y%m%d%I%M%S')

APP = $(shell cat vname)

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
	go run main.go -local -debug -f2show=/site/list123

test:
	_out/$(APP) -local -debug

bflow:
	go mod init ${APP} && go mod tidy && CGO_ENABLED=0 go build -ldflags "-w -s" -o ./_out/$(APP) ./

clean:
	rm -rf _out/$(APP) && rm go.mod go.sum
