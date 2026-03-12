package cmd

import (
	"flag"
	"fmt"
	"k8skit/app/k8sc"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"
	"github.com/suisrc/zgg/z/ze/sqlx"
	"k8s.io/client-go/kubernetes"
)

// go run main.go world

var (
	C = struct {
		CmdSync   SyncConfig
		CmdFixSvc FixSvcConfig
	}{}
)

func init() {
	z.Config(&C)
	z.CMD["hello"] = hello
}

func hello() {
	fmt.Println("hello world!")
}

func InitConfig(kcli bool) (*sqlx.Dsx, *kubernetes.Clientset) {
	z.Initializ()
	// parse command line arguments
	var cfs string
	flag.StringVar(&cfs, "c", "", "config file path")
	flag.Parse()
	// parse config file
	zc.LoadConfig(cfs)
	// ----------------------------------------------------------
	dsc, err := sqlx.ConnectDB(&C.CmdSync.Database, z.Println)
	if err != nil {
		fmt.Println("sync, connect db error: ", err.Error())
		return nil, nil
	}
	dsx := &sqlx.Dsx{Ex: dsc}
	if !kcli {
		return dsx, nil
	}
	cli, err := k8sc.CreateClient(z.C.Server.Local)
	if err != nil {
		fmt.Println("create k8s client error: ", err.Error()) // 初始化失败，直接退出
		return nil, nil
	}
	return dsx, cli
}
