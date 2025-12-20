package main

import (
	_ "embed"
	"flag"
	"kube-sidecar/app"
	_ "kube-sidecar/app/fakessl"
	_ "kube-sidecar/app/sidecar"
	_ "kube-sidecar/cmd"
	"strings"

	"github.com/suisrc/zgg/z"
	_ "github.com/suisrc/zgg/ze/rdx"
)

//go:embed vname
var appbyte []byte

//go:embed version
var verbyte []byte

var (
	appname = strings.TrimSpace(string(appbyte))
	version = strings.TrimSpace(string(verbyte))
)

/**
 * 程序入口
 */
func main() {
	flag.StringVar(&app.C.Token, "token", "", "http server api token")
	flag.StringVar(&app.C.InjectAnnotation, "injectAnnotation", "sidecar/configmap", "Injector Annotation")
	flag.StringVar(&app.C.InjectDefaultKey, "injectDefaultKey", "sidecar.yml", "Injector Default Key")

	z.Execute(appname, version, "(https://github.com/suisrc/k8skit) kube-sidecar")
}
