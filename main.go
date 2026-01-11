package main

import (
	"k8skit/app/s3cdn"
	"os"

	_ "k8skit/cmd"

	"github.com/suisrc/zgg/app/front2"
	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"
	_ "github.com/suisrc/zgg/ze/log_syslog"
)

func main() {
	if _app, _ := os.ReadFile("vname"); _app != nil {
		z.AppName = string(_app)
	}
	if _ver, _ := os.ReadFile("version"); _ver != nil {
		z.Version = string(_ver)
	}

	zc.CFG_ENV, zc.LogTrackFile = "KIT", false
	// zc.C.Syslog, zc.C.LogTty = "udp://klog.default.svc:5141", true

	// front2.Init3(www_, "/www", s3cdn.Front2ServeByS3) // 前端应用，由于需要 wwwFS参数，必须人工初始化
	front2.Init3(os.DirFS("www"), "/", s3cdn.Front2ServeByS3) // 前端应用, 使用系统文件夹中文件

	z.Execute(z.AppName, z.Version, "(https://github.com/suisrc/k8skit)")
}
