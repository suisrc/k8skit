package z

/*
 这是一极精简的 web 服务框架， 默认没有 routing， 使用 map 进行 action 检索 ，为效率而生。
 通过 query.action(query参数) 或者 path=action(path就是action) 两种方式选择 handle 函数。
 但是，可以通过 '-mux' 参数切换使用 net/http.mux 标准库的 http.router 完成标准 web 服务切换。

 这是一个标准实现，如果需要自定义实现，可以以该代码为基础，重写该模块
 这是一个标准实现，如果需要自定义实现，可以以该代码为基础，重写该模块
 这是一个标准实现，如果需要自定义实现，可以以该代码为基础，重写该模块

 [z]包为 zgg(z? google golang) 服务抽象层，尽量不要修改，以免出现兼容性问题
*/

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

var _ IConfig = (*EConfig)(nil)

/**
 * 扩展配置
 */
type EConfig struct {
	Config
	Token            string
	InjectAnnotation string
	InjectDefaultKey string
}

func (aa *EConfig) B() *Config {
	return &aa.Config
}

func (aa *EConfig) V(key string) any {
	switch key {
	case "Token":
		return aa.Token
	case "InjectAnnotation":
		return aa.InjectAnnotation
	case "InjectDefaultKey":
		return aa.InjectDefaultKey
	default:
		return aa.Config.V(key)
	}
}

// ----------------------------------------------------------------------------

/**
 * 服务对象，理论上需要存储运行时的所有数据
 */
type Server struct {
	Serve0
}

/**
 * 解析命令行参数
 */
func (aa *Server) ParseFlags() {
	conf := &EConfig{}
	flag.StringVar(&conf.Token, "token", "", "http server api token")
	flag.StringVar(&conf.InjectAnnotation, "injectAnnotation", "sidecar/configmap", "Injector Annotation")
	flag.StringVar(&conf.InjectDefaultKey, "injectDefaultKey", "sidecar.yml", "Injector Default Key")
	aa.Serve0.ParseFlags(conf, aa) // 解析命令行参数， 只有最终的实现 Server 才能调用
}

/**
 * 启动服务
 */
func (aa *Server) RunDefault() {
	aa.ParseFlags()
	if !aa.Serve0.ServeInit() {
		return // init error, exit
	}
	aa.Serve0.RunAndWait(aa.ServeHTTP)
}

// /**
//  * 可覆盖重写HTTP服务函数
//  */
// func (aa *Server) ServeHTTP(rw http.ResponseWriter, rr *http.Request) {
// 	aa.Serve0.ServeHTTP(rw, rr)
// }

// ----------------------------------------------------------------------------

func RunHttpServe() {
	(&Server{}).RunDefault()
}

func PrintVersion() {
	fmt.Printf("kube-sidecar %s (https://github.com/suisrc/kube-sidecar)\n", Version)
}

var (
	// Command Registry
	CmdR = map[string]func(){
		"web":     RunHttpServe,
		"version": PrintVersion,
	}
)

// ----------------------------------------------------------------------------

/**
 * 程序入口
 */
func Execute() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("================ exit: panic,", err)
			os.Exit(1) // exit with panic
		}
	}()
	if len(os.Args) < 2 || strings.HasPrefix(os.Args[1], "-") {
		RunHttpServe() // run  def http server
		return         // wait for server stop
	}
	cmd := os.Args[1]
	if command, ok := CmdR[cmd]; ok {
		// 修改命令参数
		os.Args = append(os.Args[:1], os.Args[2:]...)
		command() // run command
		// flag.Parse() > flag.CommandLine.Parse(os.Args[2:])
	} else {
		fmt.Println("unknown command:", cmd)
	}
}

// func Exit(err error, code int) {
// 	fmt.Println("exit with error:", err.Error())
// 	os.Exit(code)
// }
