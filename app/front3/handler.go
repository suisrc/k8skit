package front3

import (
	"flag"
	"k8skit/app"
	"k8skit/app/s3cdn"
	"net/http"
	"strings"
	"time"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"
	"github.com/suisrc/zgg/z/ze/sqlx"
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

	// 验证方式？简单一点，confa 提供令牌支持， 但是 role 必须是 front3.* 权限
	WebHookPath string `json:"hookpath"`   // 钩子路径, 默认为空，不启动钩子
	MutatePath  string `json:"mutatepath"` // 对于 ingress 的原生补丁， 默认不开启， 需要指定地址
	MutateAddr  string `json:"mutateaddr" default:"0.0.0.0:443"`
	MutateCert  string `json:"mutatecert" default:"mutatecert"` // 钩子证书文件夹
	LogIngress  bool   `json:"logingress" default:"false"`
}

func init() {
	z.Config(&C)

	flag.BoolVar(&C.Front3.Enable, "f3enable", false, "front3 启用")
	flag.BoolVar(&C.Front3.Debug, "f3debug", false, "front3 启用调试")
	flag.StringVar(&C.Front3.AddrPort, "f3addrport", "0.0.0.0:8080", "front3 监听端口")
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
		srv := &F3Serve{
			CdnConfig: s3cdn.C.S3cdn,
			RegConfig: app.C.Imagex,
			AppRepo:   &AppInfoRepo{Database: dsc, TablePrefix: tpx},
			VerRepo:   &VersionRepo{Database: dsc, TablePrefix: tpx},
			AuzRepo:   &AuthzRepo{Database: dsc, TablePrefix: tpx},
			IngRepo:   &IngressRepo{Database: dsc, TablePrefix: tpx},
			Interval:  C.Front3.CacheTimeout * 60, // 缓存清理间隔， 单位秒
			AppCache:  make(map[string]*AppCache),
			// Interval: 30, // 测试用
		}
		// 原生钩子
		if C.Front3.MutatePath != "" {
			// mutate 接口必须在 https 上
			tlc, err := srv.MutateTLS(C.Front3.MutateCert)
			if err != nil { // 有可能证书不存在， 推荐使用 fkc-ksidecar-data 证书
				zgg.ServeStop("[_front3_], init mutate tls config error, " + err.Error())
				return nil
			}
			hdl := http.HandlerFunc(srv.Mutate)
			zgg.Servers["(MUTAT)"] = &http.Server{Addr: C.Front3.MutateAddr, Handler: hdl, TLSConfig: tlc}
			z.Println("[_front3_]: mutate path =", C.Front3.MutateAddr+C.Front3.MutatePath)
		}
		// 外部钩子， 原生钩子和外部钩子分开， 以便于权限控制和配置分流
		if C.Front3.WebHookPath != "" {
			zgg.AddRouter(C.Front3.WebHookPath, srv.WebHook)
			z.Println("[_front3_]: webhook path =", C.Front3.WebHookPath)
		}
		// 本身服务
		zgg.Servers["(F3SRV)"] = &http.Server{Addr: C.Front3.AddrPort, Handler: srv}
		if C.Front3.CacheTicker > 0 {
			srv.CleanerWork(time.Duration(C.Front3.CacheTicker) * time.Minute)
		}

		return srv.Stop
	})

}
