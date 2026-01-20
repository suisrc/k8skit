package app

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/ze/sqlx"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	Token string

	// C = struct {
	// }{}
)

func init() {
	z.Register("11-app.init", func(zgg *z.Zgg) z.Closed {
		// 创建 k8sclient
		cli, err := CreateClient(z.C.Server.Local)
		if err != nil {
			zgg.ServeStop("create k8s client error: ", err.Error()) // 初始化失败，直接退出
			return nil
		}
		z.Println("[_k8scli_]: create k8s client success: local=", z.C.Server.Local)
		zgg.SvcKit.Set("k8sclient", cli) // 注册 k8sclient
		return nil
	})
}

// -----------------------------------------------------------------------

// CreateClient Create the server
func CreateClient(local bool) (*kubernetes.Clientset, error) {
	cfg, err := BuildConfig(local)
	if err != nil {
		return nil, errors.Wrapf(err, "error setting up cluster config")
	}
	return kubernetes.NewForConfig(cfg)
}

// BuildConfig Build the config
func BuildConfig(local bool) (*rest.Config, error) {
	if local {
		z.Println("[_k8scli_]: using local kubeconfig.")
		kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	z.Println("[_k8scli_]: using in cluster kubeconfig.")
	return rest.InClusterConfig()
}

// -----------------------------------------------------------------------

func PostJson(req *http.Request) error {
	if req.Method != http.MethodPost {
		return fmt.Errorf("wrong http verb. got %s", req.Method)
	}
	if req.Body == nil {
		return errors.New("empty body")
	}
	contentType := req.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		return fmt.Errorf("wrong content type. expected 'application/json', got: '%s'", contentType)
	}
	return nil
}

// -----------------------------------------------------------------------

type DatabaseConfig struct {
	DN           string `json:"driver"` // mysql
	DS           string `json:"dsn"`    // user:pass@tcp(host:port)/dbname?params
	Host         string `json:"host"`
	Port         int    `json:"port" default:"3306"`
	DBName       string `json:"dbname"`
	Params       string `json:"params"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	MaxOpenConns int    `json:"max_open_conns"`
	MaxIdleConns int    `json:"max_idle_conns"`
	MaxIdleTime  int    `json:"max_idle_time"` // 单位秒
	MaxLifetime  int    `json:"max_lifetime"`
	TablePrefix  string `json:"table_prefix"`
}

func ConnectDatabase(cfg *DatabaseConfig) (*sqlx.DB, error) {
	if cfg.DS == "" {
		if cfg.Host != "" {
			cfg.DS = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s", //
				cfg.Username, cfg.Password, //
				cfg.Host, cfg.Port, //
				cfg.DBName, cfg.Params, //
			)
		} else {
			return nil, errors.New("database dsn is empty, disable confx")
		}
	}
	// dbs, err := sql.Open("mysql", "")
	cds, err := sqlx.Connect(cfg.DN, cfg.DS)
	if err != nil {
		dsn := cfg.DS
		if idx := strings.Index(dsn, "@"); idx > 0 {
			dsn = dsn[idx:]
		}
		return nil, errors.New("database connect error [***" + dsn + "]" + err.Error())
	}
	// 设置数据库连接参数
	if cfg.MaxOpenConns > 0 {
		cds.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		cds.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.MaxIdleTime > 0 {
		cds.SetConnMaxIdleTime(time.Duration(cfg.MaxIdleTime) * time.Second)
	}
	if cfg.MaxLifetime > 0 {
		cds.SetConnMaxLifetime(time.Duration(cfg.MaxLifetime) * time.Second)
	}
	// 打印数据库连接信息
	{
		dsn := cfg.DS
		if idx := strings.Index(dsn, "@"); idx > 0 {
			dsn = dsn[idx+1:]
		}
		z.Println("[database]: connect ok,", dsn)
	}
	return cds, nil
}

// -----------------------------------------------------------------------
