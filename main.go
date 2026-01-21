package main

import (
	"k8skit/app/s3cdn"
	"os"

	_ "k8skit/cmd"

	"github.com/suisrc/zgg/app/front2"
	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"
	// _ "github.com/suisrc/zgg/z/ze/log/syslog"
)

func main() {
	zc.CFG_ENV, zc.LogTrackFile = "KIT", false

	if _app := os.Getenv(zc.CFG_ENV + "_VNAME"); _app != "" {
		z.AppName = _app
	} else if _app, _ := os.ReadFile("vname"); _app != nil {
		z.AppName = string(_app)
	}
	if _ver := os.Getenv(zc.CFG_ENV + "_VERSION"); _ver != "" {
		z.Version = _ver
	} else if _ver, _ := os.ReadFile("version"); _ver != nil {
		z.Version = string(_ver)
	}

	// zc.C.Syslog, zc.C.LogTty = "udp://klog.default.svc:5141", true

	front2.Init3(os.DirFS("www"), s3cdn.Front2ServeByS3) // 前端应用, 使用系统文件夹中文件
	z.Execute(z.AppName, z.Version, "(https://github.com/suisrc/k8skit)")
}
