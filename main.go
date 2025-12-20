package main

import (
	"embed"
	"strings"

	"github.com/suisrc/zgg/app/front2"
	"github.com/suisrc/zgg/z"
	// "k8s.io/klog/v2"
)

func main() {
	front2.Init(wwwFS)
	z.Execute(appname, version, "(https://github.com/suisrc/k8skit) front2")
}

//go:embed vname
var appbyte []byte

//go:embed version
var verbyte []byte

//go:embed www/* www/**/*
var wwwFS embed.FS

var (
	appname = strings.TrimSpace(string(appbyte))
	version = strings.TrimSpace(string(verbyte))
)
