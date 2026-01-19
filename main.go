package main

import (
	_ "embed"
	_ "k8skit/app"
	_ "k8skit/app/fakessl"
	_ "k8skit/app/sidecar"
	_ "k8skit/app/tls"
	_ "k8skit/cmd"
	"strings"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"

	// _ "github.com/suisrc/zgg/z/ze/log/syslog"
	// _ "github.com/suisrc/zgg/z/ze/rdx"
	// _ "k8skit/app/zhe"
	// _ "k8skit/cmd"

	_ "github.com/go-sql-driver/mysql"
)

//go:embed vname
var app_ []byte

//go:embed version
var ver_ []byte

/**
 * 程序入口
 */
func main() {
	_app := strings.TrimSpace(string(app_))
	_ver := strings.TrimSpace(string(ver_))
	zc.CFG_ENV, zc.LogTrackFile = "KIT", false
	z.Execute(_app, _ver, "(https://github.com/suisrc/k8skit)")
}
