package main

import (
	_ "embed"
	"strings"

	_ "k8skit/app/zhe"
	_ "k8skit/cmd"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"
	_ "github.com/suisrc/zgg/z/ze/log/syslog"
	_ "github.com/suisrc/zgg/z/ze/rdx"
)

//go:embed vname
var app_ []byte

//go:embed version
var ver_ []byte

// //go:embed www/* www/**/*
// var www_ embed.FS

func main() {
	_app := strings.TrimSpace(string(app_))
	_ver := strings.TrimSpace(string(ver_))
	zc.CFG_ENV, zc.C.LogTff = "KIT", false
	// zc.C.Syslog, zc.C.LogTty = "udp://klog.default.svc:5141", true
	// z.HttpServeDef = false // 标记是否启动默认 HTTP 服务， z.RegisterDefaultHttpServe

	z.Execute(_app, _ver, "(https://github.com/suisrc/k8skit)")
}
