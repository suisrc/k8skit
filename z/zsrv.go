package z

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"
	// 如果为了性能，可以考虑使用此进行替换
	// "github.com/puzpuzpuz/xsync/v4"
)

// 健康检查
func Healthz(ctx *Ctx) bool {
	return ctx.JSON(&Result{Success: true, Data: time.Now().Format("2006-01-02 15:04:05")})
}

// 初始化注册函数
func init() {
	// register action demo
	Register("10-healthz", func(svc SvcKit, enr Enroll) Closed {
		enr("healthz", Healthz) // all method
		return nil
	})
}

// ----------------------------------------------------------------------------

var _ http.Handler = (*Serve0)(nil)
var _ IServer = (*Serve0)(nil)

// 默认服务实体
type Serve0 struct {
	RefSrv IServer // 自身引用
	// 通用配置属性
	Config IConfig
	// 处理函数列表
	HttpMux *http.ServeMux
	HttpSrv *http.Server
	Closeds []Closed              // 模块关闭函数列表
	Handles map[string]HandleFunc // 接口函数
	// handles *xsync.Map[string, HandleFunc] // https://github.com/puzpuzpuz/xsync
	// 业务模块引用
	SvcKit SvcKit // 服务工具
	TplKit TplKit // 模版工具
	// 标记列表
	FlagStop bool // 终止标记
}

// 解析命令行
func (aa *Serve0) ParseFlags(cfg IConfig, ref IServer) {
	flag.BoolVar(&(cfg.B().Debug), "debug", false, "debug mode")
	flag.BoolVar(&(cfg.B().Local), "local", false, "http server local mode")
	flag.StringVar(&(cfg.B().Addr), "addr", "0.0.0.0", "http server addr")
	flag.IntVar(&(cfg.B().Port), "port", 80, "http server Port")
	flag.StringVar(&(cfg.B().CrtFile), "crtFile", "", "http server cer file")
	flag.StringVar(&(cfg.B().KeyFile), "keyFile", "", "http server key file")
	flag.StringVar(&(cfg.B().ApiPath), "apiRoot", "", "http server api path")
	flag.BoolVar(&(cfg.B().MuxHttp), "mux", false, "http server with mux")
	flag.StringVar(&(cfg.B().ReqXrtd), "xrt", "", "X-Request-Rt default value")
	flag.StringVar(&(cfg.B().TplPath), "tplPath", "", "http server tpl path")
	flag.Parse()
	aa.Config = cfg
	aa.RefSrv = ref
}

func (aa *Serve0) GetConfig() IConfig {
	return aa.Config
}

func (aa *Serve0) GetTplKit() TplKit {
	return aa.TplKit
}

func (aa *Serve0) GetSvcKit() SvcKit {
	return aa.SvcKit
}

// ----------------------------------------------------------------------------
// ----------------------------------------------------------------------------

// 服务初始化
func (aa *Serve0) ServeInit() bool {
	if aa.RefSrv == nil {
		Println("ServeInit: RefSrv is nil, please execute ParseFlags first")
		return false // exit
	}
	if aa.SvcKit == nil {
		aa.SvcKit = NewSvcKit(aa.RefSrv, aa.Config.B().Debug)
	}
	if aa.TplKit == nil {
		aa.TplKit = NewTplKit()
	}
	// aa.handles = xsync.NewMap[string, HandleFunc]()
	aa.Handles = make(map[string]HandleFunc)
	aa.Closeds = make([]Closed, 0)
	if aa.Config.B().MuxHttp {
		aa.HttpMux = http.NewServeMux()
	}
	// -----------------------------------------------
	kit := aa.RefSrv.GetSvcKit()
	enr := aa.RefSrv.AddHandle
	// 注册业务逻辑
	for _, opt := range options {
		if opt.Val == nil {
			continue
		}
		if aa.Config.B().Debug {
			Println("[register]:", opt.Key)
		}
		cls := opt.Val(kit, enr)
		if cls != nil {
			aa.Closeds = append(aa.Closeds, cls)
		}
		if aa.FlagStop {
			Println("[register]: serve already stop! exit...")
			return false // 退出
		}
		slices.Reverse(aa.Closeds) // 倒序, 后进先出
	}
	// -----------------------------------------------
	// 注册业务模版
	if aa.Config.B().TplPath != "" {
		err := aa.TplKit.Preload(aa.Config.B().TplPath, aa.Config.B().Debug)
		if err != nil {
			Printf("tplkit.Preload error: %v\n", err)
		}
	}

	return true
}

// 服务终止，注意，这里只会终止模版，不会终止服务， 终止服务，需要调用 hsv.Shutdown
func (aa *Serve0) ServeStop() {
	if aa.FlagStop {
		return
	}
	aa.FlagStop = true
	if aa.Closeds != nil {
		for _, cls := range aa.Closeds {
			cls() // 模块关闭
		}
	}
}

// 启动 HTTP 服务
func (aa *Serve0) RunAndWait(hdl http.HandlerFunc) {
	// ------------------------------------------------------------------------
	// Printf("http server Started, Linsten: %s:%d\n", srv.Addr, srv.Port)
	// http.ListenAndServe(fmt.Sprintf("%s:%d", addr, port), handler) // 启动HTTP服务
	// ------------------------------------------------------------------------
	// 启动HTTP服务， 并可优雅的终止
	hsv := &http.Server{Addr: fmt.Sprintf("%s:%d", aa.Config.B().Addr, aa.Config.B().Port), Handler: hdl}
	go func() {
		if aa.Config.B().Local {
			Printf("http server started, linsten: %s:%d (LOCAL)\n", "127.0.0.1", aa.Config.B().Port)
			if err := hsv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				Fatalf("Linsten: %s\n", err)
			}
		} else if aa.Config.B().CrtFile == "" || aa.Config.B().KeyFile == "" {
			Printf("http server started, linsten: %s:%d (HTTP)\n", aa.Config.B().Addr, aa.Config.B().Port)
			if err := hsv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				Fatalf("Linsten: %s\n", err)
			}
		} else {
			if aa.Config.B().Port == 80 {
				aa.Config.B().Port = 443 // 默认使用443端口
			}
			Printf("http server started, linsten: %s:%d (HTTPS)\n", aa.Config.B().Addr, aa.Config.B().Port)
			if err := hsv.ListenAndServeTLS(aa.Config.B().CrtFile, aa.Config.B().KeyFile); err != nil && err != http.ErrServerClosed {
				Fatalf("Linsten: %s\n", err)
			}
		}
	}()
	ssc := make(chan os.Signal, 1)
	signal.Notify(ssc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-ssc
	Println("http server stoping...")
	// 等待中断信号以优雅地关闭服务器（设置 5 秒的超时时间）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := hsv.Shutdown(ctx); err != nil {
		Fatal("http server shutdown:", err)
	}
	aa.RefSrv.ServeStop() // 停止业务模块， 先停服务，后停模块
	Println("http server shutdown")
}

// ----------------------------------------------------------------------------
// ----------------------------------------------------------------------------

/**
 * HTTP 处理，Action 获取方法
 * 1. 通过 query.action(query参数)
 * 2. 或者 path=action(path就是action)
 */
func (aa *Serve0) ServeHTTP(rw http.ResponseWriter, rr *http.Request) {
	if aa.Config.B().Debug {
		Printf("[_request]: %s %s\n", rr.Method, rr.URL.String())
	}
	// use net/http mux and router to process http
	if aa.Config.B().MuxHttp && aa.HttpMux != nil {
		aa.HttpMux.ServeHTTP(rw, rr)
		return // 标准服务器处理
	}
	// 查询并执行业务 Action
	ctx := NewCtx(aa.RefSrv.GetSvcKit(), rr, rw)
	defer ctx.Cancel() // 确保取消
	if ctx.Action == "" {
		res := &Result{ErrCode: "action-empty", Message: "未指定操作: empty"}
		ctx.JSON(res) // 空的操作
	} else if handle, exist := aa.RefSrv.GetHandle(rr.Method, ctx.Action); !exist {
		res := &Result{ErrCode: "action-unknow", Message: "未指定操作: " + ctx.Action}
		ctx.JSON(res) // 无效操作
	} else {
		handle(ctx) // 处理函数
	}
}

// ----------------------------------------------------------------------------
// ----------------------------------------------------------------------------

/**
 * 获取处理函数
 * 使用 map 取代 http router, 简化路由和提高性能
 * @param method
 * @param action
 */
func (aa *Serve0) GetHandle(method, action string) (HandleFunc, bool) {
	// if handle, exist := aa.Handles.Load(method + " " + action); exist {
	// 	return handle, true
	// }
	// return aa.Handles.Load(action)
	// ServeInit 初始化后，map 就不会变更了。如果有变更需求，使用 xsync 替换
	if handle, exist := aa.Handles[method+" /"+action]; exist {
		return handle, true
	}
	handle, exist := aa.Handles["/"+action]
	return handle, exist
}

/**
 * 添加处理函数
 * @param key: [method:]action, 如果 method 为空，则默认为 所有请求
 */
func (aa *Serve0) AddHandle(key string, handle HandleFunc) {
	if key == "" {
		return // pass
	}
	// 解析 method 和 action
	method, action, found := key, "", false
	if i := strings.IndexAny(key, " \t"); i >= 0 {
		method, action, found = key[:i], strings.TrimLeft(key[i+1:], " \t"), true
	}
	if !found {
		action = method
		method = ""
	}
	// 去除 action 前的 /
	if len(action) > 0 && action[0] == '/' {
		action = action[1:]
	}
	// 补充 api path
	if aa.Config.B().ApiPath != "" {
		action = aa.Config.B().ApiPath + "/" + action
	}
	// 添加 method 前缀
	if found {
		action = method + " /" + action
	} else {
		action = "/" + action
	}
	if aa.Config.B().Debug {
		Printf("[_handle_]: %36s    %v\n", action, handle)
	}
	// aa.handles.Store(key, handle)
	// Add 操作是通过 ServeInit 出发的，是单线程的，所以不需要加锁
	aa.Handles[action] = handle
	if aa.Config.B().MuxHttp && aa.HttpMux != nil {
		aa.HttpMux.HandleFunc(action, ToHttpFunc(aa.RefSrv.GetSvcKit(), handle))
	}
}
