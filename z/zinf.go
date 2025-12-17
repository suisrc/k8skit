package z

import (
	"slices"
	"sync"
)

// ----------------------------------------------------------------------------
// ----------------------------------------------------------------------------

// 定义处理函数
type HandleFunc func(rc *Ctx) bool

// map any
type HA map[string]any

// map str
type HS map[string]string

// ----------------------------------------------------------------------------
// ----------------------------------------------------------------------------

// close function
type Closed func()

// Enroll 注册处理函数
type Enroll func(key string, handle HandleFunc)

// 定义配置函数
type OptionFunc func(svc SvcKit, enr Enroll) Closed

var (
	// 应用配置列表，依据 key 排序，初始化顺序
	options = []Ref[string, OptionFunc]{}

	optlock = sync.Mutex{} // 注册方法，全局锁即可
	// optlock = sync.RWMutex{}
	// optlock = xsync.NewRBMutex()
)

// ----------------------------------------------------------------------------
// ----------------------------------------------------------------------------

// 配置接口
type IConfig interface {
	B() *Config
	V(key string) any
}

var _ IConfig = (*Config)(nil)

// 默认配置， Server配置需要内嵌该结构体
type Config struct {
	Debug   bool
	Local   bool
	Addr    string
	Port    int
	CrtFile string
	KeyFile string
	ApiPath string // root api path
	MuxHttp bool   // 使用 http.mux 处理，标准处理函数
	ReqXrtd string // 响应内容类型
	TplPath string // 模版路径
}

func (aa *Config) B() *Config {
	return aa
}

func (aa *Config) V(key string) any {
	return nil
}

// ----------------------------------------------------------------------------
// ----------------------------------------------------------------------------

// 服务工具接口
type SvcKit interface {
	Get(key string) any             // 获取服务
	Set(key string, val any) SvcKit // 增加服务 val = nil 是卸载服务
	Map() map[string]any            // 服务列表, 注意，是副本
	Inj(obj any) SvcKit             // 注册服务 injec 使用 `svckit:"xxx"` 初始化服务
	Srv() IServer                   // 获取模块管理器 *Server 接口
}

// 模块接口, Server
type IServer interface {
	GetConfig() IConfig
	// 获取模版工具
	GetTplKit() TplKit
	// 获取服务工具
	GetSvcKit() SvcKit
	// 获取处理函数
	GetHandle(method, action string) (HandleFunc, bool)
	// 增加处理函数，?该函数有点危险，改用方法单独传递
	AddHandle(key string, handle HandleFunc)
	// 关闭服务, 危险，非关闭的服务，请勿调用
	ServeStop()
	// // 服务初始化
	// ServeInit() bool
	// // 服务响应方法
	// ServeHTTP(rw http.ResponseWriter, rr *http.Request)
}

// ----------------------------------------------------------------------------
// ----------------------------------------------------------------------------

// 在 init 注册配置函数
func Register(key string, opt OptionFunc) {
	optlock.Lock() // 注册方法，全局锁即可
	defer optlock.Unlock()
	// options = append(options, Ref[string, OptionFunc]{Key: key, Val: opt})
	idx := slices.IndexFunc(options, func(opt Ref[string, OptionFunc]) bool {
		return opt.Key > key
	})
	ref := Ref[string, OptionFunc]{Key: key, Val: opt}
	if idx < 0 {
		options = append(options, ref)
	} else {
		options = slices.Insert(options, idx, ref)
	}

	// debug print console
	// Print("options: |")
	// for _, opt := range options[:] {
	// 	Printf(" %v |", opt.Key)
	// }
	// Println()
}
