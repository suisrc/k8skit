package main

import (
	_ "embed"
	"flag"
	_ "k8skit/app/fakessl"
	_ "k8skit/app/sidecar"
	_ "k8skit/cmd"
	"strings"

	"github.com/suisrc/zgg/app/fluent"
	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"
	_ "github.com/suisrc/zgg/ze/rdx"
	"k8s.io/klog/v2"
	// _ "github.com/suisrc/zgg/app/zhe"
	// _ "k8skit/app/zhe"
	// _ "k8skit/cmd"
)

//go:embed vname
var app_ []byte

//go:embed version
var ver_ []byte

// //go:embed www/* www/**/*
// var www_ embed.FS

var (
	app = strings.TrimSpace(string(app_))
	ver = strings.TrimSpace(string(ver_))
)

/**
 * 程序入口
 */
func main() {
	zc.CFG_ENV = "KIT"
	z.Println = klog.Infoln
	z.Printf = klog.Infof
	z.Fatalf = klog.Fatalf
	z.Fatal = klog.Fatal

	flag.StringVar(&app.C.Token, "token", "", "http server api token")
	flag.StringVar(&app.C.InjectAnnotation, "injectAnnotation", "sidecar/configmap", "Injector Annotation")
	flag.StringVar(&app.C.InjectDefaultKey, "injectDefaultKey", "sidecar.yml", "Injector Default Key")

	// front2.Init(www_) // 前端应用，由于需要 wwwFS参数，必须人工初始化
	// kwdog2.Init() // API边车网关， 通过 Sidecar 模式保护主服务
	fluent.Init() // 采集器日志, 为 fluentbit agent 提供 HTTP 收集日志功能

	z.Execute(app, ver, "(https://github.com/suisrc/k8skit) "+app)
}
