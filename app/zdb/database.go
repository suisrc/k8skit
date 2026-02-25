package zdb

// 数据库链接和管理

import (
	"database/sql"
	"embed"
	"flag"
	"strings"
	"time"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/ze/sqlx"
)

var (
	C = struct {
		Database sqlx.DatabaseConfig
		Cache    CacheConfig
	}{}
)

type CacheConfig struct {
	DefaultExpired  int `json:"expired" default:"7200"`
	CleanupInterval int `json:"interval" default:"7200"`
}

func init() {
	z.Config(&C)

	flag.StringVar(&C.Database.Driver, "dsd", "mysql", "数据库驱动")
	flag.StringVar(&C.Database.DataSource, "dsn", "", "数据库连接")

	z.Register("20-database", func(zgg *z.Zgg) z.Closed {
		// 数据库链接
		dsc, err := sqlx.ConnectDatabase(&C.Database)
		if err != nil {
			zgg.ServeStop(err.Error())
			return nil
		} else {
			dsn := C.Database.DataSource
			if idx := strings.Index(dsn, "@"); idx > 0 {
				dsn = dsn[idx+1:]
			}
			z.Println("[database]: connect ok,", dsn)
		}
		z.RegKey(zgg.SvcKit, false, "dsc", dsc)
		// 本地缓存
		cacde := time.Duration(C.Cache.DefaultExpired) * time.Second
		cacci := time.Duration(C.Cache.CleanupInterval) * time.Second
		cache := NewCacheMem(cacde, cacci)
		z.RegKey(zgg.SvcKit, false, "cache", cache)
		// 数据库链接
		NewDsc = func() sqlx.Dsc { return &sqlx.Dsx{Ex: dsc} }
		return func() { dsc.Close(); NewDsc = nil }
	})
}

// 生成数据库链接
var NewDsc func() sqlx.Dsc

// ===================================================================================

//go:embed ksql/*
var ksql embed.FS

// ksql cache map
var kmap = map[string]string{}

// ksql function
func Ksql[T any](name string, argm map[string]any, size bool) ([]T, int64, error) {
	str, ok := kmap[name]
	if !ok {
		if bts, err := ksql.ReadFile("ksql/" + name + ".sql"); err != nil {
			return nil, 0, err
		} else {
			str = string(bts)
			kmap[name] = str
		}
	}
	return sqlx.Ksql[T](NewDsc(), str, argm, size)
}

// ===================================================================================

// 基础数据对象
type BaseDO struct {
	Disable sql.NullBool   `db:"disable"`
	Deleted sql.NullBool   `db:"deleted"`
	Updated sql.NullTime   `db:"updated"`
	Updater sql.NullString `db:"updater"`
	Created sql.NullTime   `db:"created"`
	Creater sql.NullString `db:"creater"`
	Version sql.NullInt64  `db:"version"`
}
