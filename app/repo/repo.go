package repo

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/ze/sqlx"
)

var (
	C = struct {
		DB Config `json:"database"`
	}{}
)

type Config struct {
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

func init() {
	z.Config(&C)

	flag.StringVar(&C.DB.DN, "driver", "mysql", "sqlx driver name")
	flag.StringVar(&C.DB.DS, "dsn", "", "sqlx data source name")

	z.Register("11-repo.init", func(zgg *z.Zgg) z.Closed {
		if C.DB.DS == "" {
			if C.DB.Host != "" {
				C.DB.DS = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s", //
					C.DB.Username, C.DB.Password, //
					C.DB.Host, C.DB.Port, //
					C.DB.DBName, C.DB.Params, //
				)
			} else {
				// zgg.ServeStop("database dsn is empty")
				z.RegKey(zgg.SvcKit, false, "repoconfx", &ConfxRepo{})
				z.Println("database dsn is empty, disable confx")
				return nil
			}
		}
		// dbs, err := sql.Open("mysql", "")
		cds, err := sqlx.Connect(C.DB.DN, C.DB.DS)
		if err != nil {
			dsn := C.DB.DS
			if idx := strings.Index(dsn, "@"); idx > 0 {
				dsn = dsn[idx:]
			}
			zgg.ServeStop("database connect error [", "***"+dsn, "]", err.Error())
			return nil
		}
		// 设置数据库连接参数
		if C.DB.MaxOpenConns > 0 {
			cds.SetMaxOpenConns(C.DB.MaxOpenConns)
		}
		if C.DB.MaxIdleConns > 0 {
			cds.SetMaxIdleConns(C.DB.MaxIdleConns)
		}
		if C.DB.MaxIdleTime > 0 {
			cds.SetConnMaxIdleTime(time.Duration(C.DB.MaxIdleTime) * time.Second)
		}
		if C.DB.MaxLifetime > 0 {
			cds.SetConnMaxLifetime(time.Duration(C.DB.MaxLifetime) * time.Second)
		}
		// 注册数据库链接
		z.RegSvc(zgg.SvcKit, cds)
		z.RegKey(zgg.SvcKit, false, "repoconfx", &ConfxRepo{DS: cds})

		{
			dsn := C.DB.DS
			if idx := strings.Index(dsn, "@"); idx > 0 {
				dsn = dsn[idx+1:]
			}
			z.Println("[database]: connect ok,", dsn)
		}
		return func() { cds.Close() }
	})
}
