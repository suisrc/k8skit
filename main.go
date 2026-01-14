package main

import (
	_ "embed"
	"strings"

"github.com/suisrc/zgg/ze/gte"
"github.com/suisrc/zgg/app/proxy2"
"github.com/suisrc/zgg/app/kwdog2"
	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"
	_ "github.com/suisrc/zgg/ze/log_syslog"
	_ "github.com/suisrc/zgg/ze/rdx"
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

func main() {
	_app := strings.TrimSpace(string(app_))
	_ver := strings.TrimSpace(string(ver_))
	zc.CFG_ENV, zc.LogTrackFile = "KIT", false
	// zc.C.Syslog, zc.C.LogTty = "udp://klog.default.svc:5141", true
	z.HttpServeDef = false // 标记是否启动默认 HTTP 服务， z.RegisterDefaultHttpServe
	kwdog2.RecordFunc = gte.ToRecord1

	// front2.Init3(www_, "/www", nil) // 前端应用，由于需要 wwwFS参数，必须人工初始化
	// front2.Init3(os.DirFS("www"), "/", nil) // 前端应用, 使用系统文件夹中文件
	// kwlog2.Init3(nil) // 采集器日志, 为 fluentbit agent 提供 HTTP 收集日志功能
	kwdog2.Init3(nil) // API反向网关， 通过 Sidecar 模式保护内部服务
	proxy2.Init3(nil) // API正向网关， 通过 Sidecar 模式记录外部访问

	z.Execute(_app, _ver, "(https://github.com/suisrc/k8skit)")
}
