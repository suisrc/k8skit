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
	go run main.go -c zgg.toml

mapp:
	go run main.go -xrt 2 -local -debug -port 81 -dual \
	--secretName fkc-sidecar-data \
	--injectAnnotation ksidecar/configmap \
	--injectDefaultKey sidecar.yml

tenv:
	KIT_KWLOG2_TOKEN=xxxx123456789 go run main.go -debug -print -port 81

test:
	_out/$(APP) -local -debug -port 81

bflow:
	go mod init ${APP} && go mod tidy && CGO_ENABLED=0 go build -ldflags "-w -s" -o ./_out/$(APP) ./

clean:
	rm -rf _out/$(APP) && rm go.mod go.sum

git:
	@if [ -z "$(tag)" ]; then \
		echo "error: 'tag' not specified! Please specify the 'tag' using 'make tflow tag=(version)-(appname)'";\
		exit 1; \
	fi
	git commit -am "${tag}" && git tag -a $(tag) -m "${tag}" && git push && git push origin $(tag)
