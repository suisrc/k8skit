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
	go run main.go -debug -port 81 -dual \
	--dsn "cfg:i3SbJ6snkQeZXt@tcp(mysql.base.svc:3306)/cfg?charset=utf8&parseTime=True&loc=Asia%2FShanghai" \
	--injectServerHost http://vscode.default.svc:81
	
# 	go run main.go -c zgg.toml
# 	--secretName fkc-ksidecar-data \
# 	--injectAnnotation ksidecar/configmap \
# 	--injectDefaultKey value.yml

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
		echo "error: 'tag' not specified! Please specify the 'tag' using 'make tflow tag=(version)'";\
		exit 1; \
	fi
	git commit -am "${tag}" && git tag -a $(tag)-sidecar -m "${tag}" && git push && git push origin $(tag)-sidecar
