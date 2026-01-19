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
		echo "error: 'tag' not specified! Please specify the 'tag' using 'make git tag=(version)-(appname)'";\
		exit 1; \
	fi
	git commit -am "${tag}" && git tag -a $(tag) -m "${tag}" && git push origin $(tag) && git reset --hard HEAD~1

front2:
	@if [ -z "$(tag)" ]; then \
		echo "error: 'tag' not specified! Please specify the 'tag' using 'make front2 tag=(version)";\
		exit 1; \
	fi
	sed -i -e 's|// front2.Init3(os.|front2.Init3(os.|g' -e '7i"os"' -e '7i"github.com/suisrc/zgg/app/front2"' main.go
	git commit -am "${tag}" && git tag -a $(tag)-front2 -m "${tag}" && git push origin $(tag)-front2 && git reset --hard HEAD~1

kwlog2:
	@if [ -z "$(tag)" ]; then \
		echo "error: 'tag' not specified! Please specify the 'tag' using 'make kwlog2 tag=(version)";\
		exit 1; \
	fi
	sed -i -e 's|// kwlog2.|kwlog2.|g' -e '7i"github.com/suisrc/zgg/app/kwlog2"' main.go
	git commit -am "${tag}" && git tag -a $(tag)-kwlog2 -m "${tag}" && git push origin $(tag)-kwlog2 && git reset --hard HEAD~1

kwdog2:
	@if [ -z "$(tag)" ]; then \
		echo "error: 'tag' not specified! Please specify the 'tag' using 'make kwdog2 tag=(version)";\
		exit 1; \
	fi
	sed -i -e 's|// z.HttpServeDef|z.HttpServeDef|g' -e '7i"github.com/suisrc/zgg/z/ze/gte"' \
	-e 's|// proxy2.|proxy2.|g' -e '7i"github.com/suisrc/zgg/app/proxy2"' \
	-e 's|// kwdog2.|kwdog2.|g' -e '7i"github.com/suisrc/zgg/app/kwdog2"' main.go
	git commit -am "${tag}" && git tag -a $(tag)-kwdog2 -m "${tag}" && git push origin $(tag)-kwdog2 && git reset --hard HEAD~1

wgetar:
	@if [ -z "$(tag)" ]; then \
		echo "error: 'tag' not specified! Please specify the 'tag' using 'make wgetar tag=(version)";\
		exit 1; \
	fi
	cp wget_tar Dockerfile
	git commit -am "${tag}" && git tag -a $(tag)-wgetar -m "${tag}" && git push origin $(tag)-wgetar && git reset --hard HEAD~1