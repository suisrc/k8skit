.PHONY: start build

NOW = $(shell date -u '+%Y%m%d%I%M%S')
FTV = fluent4.2.2

APP = alisls-fluentd

# 初始化mod
init:
	go mod init github.com/suisrc/${APP}

# 修正依赖
tidy:
	go mod tidy

build:
	go build -buildmode=c-shared -o ali_sls.so .

git:
	@if [ -z "$(tag)" ]; then \
		echo "error: 'tag' not specified! Please specify the 'tag' using 'make tflow tag=(version)'";\
		exit 1; \
	fi
	git commit -am "${tag}" && git tag -a $(tag)-$(FTV) -m "${tag}" && git push && git push origin $(tag)-$(FTV)
