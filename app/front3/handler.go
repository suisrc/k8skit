package front3

import (
	"crypto/tls"
	"flag"
	"k8skit/app"
	"k8skit/app/registry"
	"k8skit/app/s3cdn"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"
	"github.com/suisrc/zgg/z/ze/sqlx"
	"github.com/suisrc/zgg/z/ze/tlsx"
)

// 多前端代理, 通过镜像动态加载多个站点的前端资源

var (
	C = struct {
		Front3 Config
	}{}
)

type Config struct {
	DB sqlx.DatabaseConfig `json:"database"`

	Enable       bool   `json:"enable"`       // 禁用
	Debug        bool   `json:"debug"`        // 调试模式
	AddrPort     string `json:"addrport"`     // 监听端口，不破坏源服务，独立新服务监控 CDN 索引
	CacheTicker  int64  `json:"cacheticker"`  // 缓存清理间隔， 0 表示不启用, 默认为1天
	CacheTimeout int64  `json:"cachetimeout"` // 缓存存储时间， 0 默认 30 天
	ImageMaps    z.HM   `json:"imagemaps"`    // 镜像映射
	SyncToken    string `json:"synctoken"`    // 同步令牌，用于k8s集群多实例间配置同步
	SyncServe    string `json:"syncserve"`    // 同步服务地址, 一般是 headless service 的地址, 也可以是 http://HOST:PORT 形式

	// 验证方式？简单一点，confa 提供令牌支持， 但是 role 必须是 front3.* 权限
	WebHookPath string `json:"hookpath"` // 钩子路径, 默认为空，不启动钩子
	MutateAddr  string `json:"mutateaddr" default:"0.0.0.0:443"`
	MutateCert  string `json:"mutatecert" default:"mutatecert"` // 钩子证书文件夹

	MutatePath     string   `json:"mutatepath"` // 对于 ingress 的原生补丁， 默认不开启， 需要指定地址
	LogIngress     bool     `json:"logingress" default:"false"`
	RecordPath     string   `json:"recordpath"` // 记录全部模版, 默认为空，不启动记录
	RecordPassMeta []string `json:"recordpassmeta"`
	RecordPassSpec []string `json:"recordpassspec"`
}

func init() {
	z.Config(&C)

	flag.BoolVar(&C.Front3.Enable, "f3enable", false, "front3 启用")
	flag.BoolVar(&C.Front3.Debug, "f3debug", false, "front3 启用调试")
	flag.StringVar(&C.Front3.AddrPort, "f3addrport", "0.0.0.0:80", "front3 监听端口")
	flag.StringVar(&C.Front3.DB.Driver, "f3driver", "mysql", "front3 数据库驱动")
	flag.Int64Var(&C.Front3.CacheTicker, "f3ticker", 86400, "front3 缓存清理间隔, 0 禁用, 默认 1 天")
	flag.Int64Var(&C.Front3.CacheTimeout, "f3cachetimeout", 2592000, "front3 缓存存储时间, 0 默认 30 天")

	flag.StringVar(&C.Front3.MutateAddr, "mutateaddr", "0.0.0.0:443", "钩子地址")
	flag.StringVar(&C.Front3.MutateCert, "mutatecert", "mutatecert", "钩子路径")

	z.Register("88-front3", func(zgg *z.Zgg) z.Closed {
		if !C.Front3.Enable {
			z.Println("[_front3_]: front3 is disable", zc.CFG_ENV+"_FRONT3_ENABLE=false")
			return nil
		}
		dsc, err := sqlx.ConnectDatabase(&C.Front3.DB)
		if err != nil {
			zgg.ServeStop(err.Error())
			return nil
		} else {
			dsn := C.Front3.DB.DataSource
			if idx := strings.Index(dsn, "@"); idx > 0 {
				dsn = dsn[idx+1:]
			}
			z.Println("[_front3_]: connect ok,", dsn)
		}
		// 提供 F3 索引服务
		tpx := C.Front3.DB.TablePrefix
		srv := &Serve{
			// repo config
			AppRepo: &AppInfoRepo{Database: dsc, TablePrefix: tpx},
			VerRepo: &VersionRepo{Database: dsc, TablePrefix: tpx},
			AuzRepo: &AuthzRepo{Database: dsc, TablePrefix: tpx},
			IngRepo: &IngressRepo{Database: dsc, TablePrefix: tpx},
			RecRepo: &RecordRepo{Database: dsc, TablePrefix: tpx},
			// f3s config
			CdnConfig: s3cdn.C.S3cdn,
			RegConfig: app.C.Imagex,
			Interval:  C.Front3.CacheTimeout * 60, // 缓存清理间隔， 单位秒
			// Interval: 30, // 测试用
		}
		// 原生钩子
		if C.Front3.MutatePath != "" || C.Front3.RecordPath != "" {
			// mutate 接口必须在 https 上
			tlc, err := BuildTlsByDir(C.Front3.MutateCert)
			if err != nil { // 有可能证书不存在， 推荐使用 fkc-ksidecar-data 证书
				zgg.ServeStop("[_front3_], init mutate tls config error, " + err.Error())
				return nil
			}
			hdl := http.HandlerFunc(srv.MutateHook)
			zgg.Servers["(MUTAT)"] = &http.Server{Addr: C.Front3.MutateAddr, Handler: hdl, TLSConfig: tlc}

			if C.Front3.MutatePath != "" {
				z.Println("[_front3_]: mutate path =", C.Front3.MutateAddr+C.Front3.MutatePath)
			}
			if C.Front3.RecordPath != "" {
				z.Println("[_front3_]: record path =", C.Front3.MutateAddr+C.Front3.RecordPath)
			}
		}
		// 外部钩子， 原生钩子和外部钩子分开， 以便于权限控制和配置分流
		if C.Front3.WebHookPath != "" {
			z.GET(C.Front3.WebHookPath, srv.WebHook, zgg)
			z.POST(C.Front3.WebHookPath, srv.WebHook, zgg)
			z.Println("[_front3_]: webhook path =", C.Front3.WebHookPath)
		}
		// 本身服务
		if C.Front3.AddrPort != "none" {
			if strings.HasSuffix(C.Front3.AddrPort, ":80") && z.C.Server.Port == 80 {
				z.C.Server.Port = 8080 // 端口存在冲突， 修正主服务端口
			}
			zgg.Servers["(F3SRV)"] = &http.Server{Addr: C.Front3.AddrPort, Handler: srv}
			if C.Front3.CacheTicker > 0 {
				srv.CleanerStart(time.Duration(C.Front3.CacheTicker) * time.Minute)
			}
		}
		return srv.Stop
	})

}

//=============================================================================================================================

// front3 服务， 提供前端缓存， k8s 原始 ingress 配置， k8s 原生 配置记录
type Serve struct {
	// repo config
	AppRepo *AppInfoRepo
	VerRepo *VersionRepo
	AuzRepo *AuthzRepo
	IngRepo *IngressRepo
	RecRepo *RecordRepo
	// f3s config
	CdnConfig s3cdn.Config    // cdn 配置
	RegConfig registry.Config // 镜像仓库配置
	Interval  int64           // 单位秒, 巡检间隔
	CacheApp  sync.Map        // map[string]*AppCache， 由于存在频繁更改情况，使用sync.Map
	_CacheMU  sync.Mutex      // 缓存操作锁
	_CacheTT  *time.Ticker    // 定时清理缓存
	_CacheCC  int64           // 定时清理计数器
}

// 默认 80 端口
func (aa *Serve) ServeHTTP(rw http.ResponseWriter, rr *http.Request) {
	if rr.URL.Path == "/healthz" && rr.Method == http.MethodGet {
		z.Healthz(z.NewCtx(nil, rr, rw, "f3s"))
	} else {
		aa.ServeS3(rw, rr)
	}
}

// 默认 8080 端口， 提供第三方webhook接口，可对系统进行配置和修改， 附加到主服务上，提供外部外部访问
func (aa *Serve) WebHook(zrc *z.Ctx) {
	qry := zrc.Request.URL.Query()
	if src := qry.Get("source"); src != "" && zrc.Request.Method == http.MethodPost {
		// 节点同步, 推送到 AnswerSyncConfig 方法中完成
		aa.AnswerSyncHook(src, qry.Get("method"), zrc) // 异步相应
		return
	}
	// 人工操作， 通过 switch 分类处理, 先判断权限
	switch qry.Get("method") {
	case "update.image":
		aa.UpdateImageVersion(zrc)
	case "delete.cache":
		aa.DeleteLocalCache(zrc)
	default:
		zrc.TEXT("not found", http.StatusOK)
	}
}

// 默认 443 端口， k8s 原生支持， 通过 MutatingWebhookConfiguration or ValidatingWebhookConfiguration 对 k8s 资源进行修正和监控
func (aa *Serve) MutateHook(rw http.ResponseWriter, rr *http.Request) {
	switch rr.URL.Path {
	case "":
		z.Println("[_mutate_]: serve endpoint, mutate path is empty")
		writeErrorAdmissionReview(http.StatusBadRequest, "mutate path is empty", rw)
	case C.Front3.MutatePath:
		aa.Mutate(rw, rr) // 对 ingress 原生修改， 提供 ServeS3 配置的原生支持
	case C.Front3.RecordPath:
		aa.Record(rw, rr) // 可记录 k8s 所有的原生模版信息
	default:
		z.Println("[_mutate_]: serve endpoint, mutate path is invalid:", rr.URL.Path)
		writeErrorAdmissionReview(http.StatusBadRequest, "mutate path is invalid: "+rr.URL.Path, rw)
	}
}

//=============================================================================================================================

func BuildTlsByDir(dir string) (*tls.Config, error) {
	config := tlsx.TLSAutoConfig{}
	if crtBts, err := os.ReadFile(filepath.Join(dir, "ca.crt")); err != nil {
		return nil, err
	} else {
		config.CaCrtBts = crtBts
	}
	if keyBts, err := os.ReadFile(filepath.Join(dir, "ca.key")); err != nil {
		return nil, err
	} else {
		config.CaKeyBts = keyBts
	}
	cfg := &tls.Config{}
	cfg.GetCertificate = (&config).GetCertificate
	return cfg, nil
}
