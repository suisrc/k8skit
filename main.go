package main

import (
	_ "embed"
	"strings"

	"github.com/suisrc/zgg/app/fluent"
	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"
	"k8s.io/klog/v2"
)

//go:embed vname
var appbyte []byte

//go:embed version
var verbyte []byte

// //go:embed www/* www/**/*
// var wwwFS embed.FS

var (
	appname = strings.TrimSpace(string(appbyte))
	version = strings.TrimSpace(string(verbyte))
)

func main() {
	zc.CFG_ENV = "KIT"
	z.Println = klog.Infoln
	z.Printf = klog.Infof
	z.Fatalf = klog.Fatalf
	z.Fatal = klog.Fatal

	fluent.Init()
	z.Execute(appname, version, "(https://github.com/suisrc/k8skit) fluent")
}
