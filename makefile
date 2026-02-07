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
	KIT_KWLOG2_TOKEN=xxxx123456789 go run main.go -local -debug -port 81 -c __zgg.toml

deploy:
	go run main.go  deploy -print -c __zgg.toml -s3rewrite

imagex:
	go run main.go imagex -c __zgg.toml

tenv:
	KIT_KWLOG2_TOKEN=xxxx123456789 go run main.go -debug -print -port 81

tgzc:
	go run main.go tgzc _out/image/www/sso/v1.0.152 _out/image/www/sso/v1.0.152.tgz

tgzx:
	go run main.go tgzx _out/image/www/sso/v1.0.152.tgz _out/image/www/sso/v1.0.152-copy

test:
	_out/$(APP) -local -debug -port 81

bflow:
	go mod init ${APP} && go mod tidy && CGO_ENABLED=0 go build -ldflags "-w -s" -o ./_out/$(APP) ./

clean:
	rm -rf _out/$(APP) && rm go.mod go.sum

git:
	@if [ -z "$(tag)" ]; then \
		echo "error: 'tag' not specified! Please specify the 'tag' using 'make tflow tag=(version)'";\
		exit 1; \
	fi
	git commit -am "${tag}" && git tag -a $(tag)-front3 -m "${tag}" && git push && git push origin $(tag)-front3
