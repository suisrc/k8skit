package main

import (
	"k8skit/app/s3cdn"
	"os"
	"strings"

	_ "k8skit/cmd"

	"github.com/suisrc/zgg/app/front2"
	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"

	// _ "github.com/suisrc/zgg/z/ze/log/syslog"

	_ "k8skit/app/front3" // 提供多前端服务

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	zc.CFG_ENV, zc.LogTrackFile = "KIT", false
	// zc.C.Syslog, zc.C.LogTty = "udp://klog.default.svc:5141", true

	if _app := os.Getenv(zc.CFG_ENV + "_VNAME"); _app != "" {
		z.AppName = strings.TrimSpace(_app)
	} else if _app, _ := os.ReadFile("vname"); _app != nil {
		z.AppName = strings.TrimSpace(string(_app))
	}
	if _ver := os.Getenv(zc.CFG_ENV + "_VERSION"); _ver != "" {
		z.Version = strings.TrimSpace(_ver)
	} else if _ver, _ := os.ReadFile("version"); _ver != nil {
		z.Version = strings.TrimSpace(string(_ver))
	} // else app.C.Imagex.Version

	front2.Init3(os.DirFS("www"), s3cdn.Front2ServeByS3) // 前端应用, 使用系统文件夹中文件
	z.Execute(z.AppName, z.Version, "(https://github.com/suisrc/k8skit)")
}
