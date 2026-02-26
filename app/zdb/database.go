package zdb

// 数据库链接和管理

import (
	"database/sql"
	"embed"
	"flag"
	"os"
	"strings"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/ze/sqlx"
)

var (
	C = struct {
		Database sqlx.DatabaseConfig
	}{}
)

func init() {
	z.Config(&C)

	flag.StringVar(&C.Database.Driver, "dsd", "mysql", "数据库驱动")
	flag.StringVar(&C.Database.DataSource, "dsn", "", "数据库连接")

	// 激活 ksql 模板
	sqlx.C.Sqlx.KsqlTbl = true

	z.Register("20-database", func(zgg *z.Zgg) z.Closed {
		if sqlx.C.Sqlx.KsqlTbl {
			sqlx.RegKsqlEvalue("entity", sqlx.KsqlTblExt)
		}
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
		NewDsc = func() sqlx.Dsc { return &sqlx.Dsx{Ex: dsc} }
		if sqlx.C.Sqlx.KsqlDebug {
			ksgr = sqlx.Ksgr(os.DirFS("app/zdb/ksql"), "")
		}
		// 注册仓库 ---------------------------------------------------
		z.RegKey(zgg.SvcKit, false, "", sqlx.NewRepo[AuthzRepo](ksgr))

		// 清理函数 ---------------------------------------------------
		return func() { dsc.Close(); NewDsc = nil }
	})
}

// 生成数据库链接
var NewDsc func() sqlx.Dsc

//go:embed ksql/*
var ksfs embed.FS
var ksgr = sqlx.Ksgr(ksfs, "ksql/") // if sqlx.C.Sqlx.KsqlDebug { ksgr = sqlx.Ksgr(os.DirFS("ksql"), "") }

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
