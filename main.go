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
	"k8s.io/klog/v2"
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
	z.Println = klog.Infoln
	z.Printf = klog.Infof
	z.Fatal = klog.Fatal
	z.Fatalf = klog.Fatalf

	flag.StringVar(&app.C.Token, "token", "", "http server api token")
	flag.StringVar(&app.C.InjectAnnotation, "injectAnnotation", "sidecar/configmap", "Injector Annotation")
	flag.StringVar(&app.C.InjectDefaultKey, "injectDefaultKey", "sidecar.yml", "Injector Default Key")

	z.Execute(appname, version, "(https://github.com/suisrc/k8skit) kube-sidecar")
}
