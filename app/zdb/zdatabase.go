package zdb

// 数据库链接和管理

import (
	"database/sql"
	"embed"
	"flag"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/ze/sqlx"
)

var (
	C = struct {
		Database sqlx.DatabaseConfig
		DBAction DbActionConfig
	}{}

	//go:embed ksql/*
	ksfs embed.FS
	ksgr = sqlx.Ksgr(ksfs, "ksql/") // if sqlx.C.Sqlx.KsqlDebug { ksgr = sqlx.Ksgr(os.DirFS("ksql"), "") }
)

type DbActionConfig struct {
	System string `json:"system"`
	Enable bool   `json:"enable"`
	TpRoot string `json:"tproot"`
}

func init() {
	z.Config(&C)

	flag.StringVar(&C.Database.Driver, "dsd", "mysql", "数据库驱动")
	flag.StringVar(&C.Database.DataSource, "dsn", "", "数据库连接")

	// 激活 ksql 模板 --> sqlx.RegKsqlEvalue("entity", sqlx.KsqlTblExt)
	sqlx.C.Sqlx.KsqlTbl = true

	z.Register("20-database", func(zgg *z.Zgg) z.Closed {
		if C.Database.Driver == "disable" {
			z.Println("[database]: disabled, skip")
			return nil // 禁用数据库
		}
		if sqlx.C.Sqlx.KsqlTbl {
			sqlx.RegKsqlEvalue("entity", sqlx.KsqlTblExt)
		}
		// 创建数据库 -------------------------------------------------
		dsc, err := sqlx.ConnectDB(&C.Database, z.Println)
		if err != nil {
			zgg.ServeStop(err.Error())
			return nil
		}
		z.RegKey(zgg.SvcKit, false, "dsc", dsc)
		if sqlx.C.Sqlx.KsqlDebug {
			ksgr = sqlx.Ksgr(os.DirFS("app/zdb/ksql"), "")
		}
		dsx := &sqlx.Dsx{Ex: dsc}
		act := sqlx.NewRepox[ActionRepo](dsx, ksgr) // 提供基于 db 的 ksql 操作
		// 注册数据仓 -------------------------------------------------
		z.RegKey(zgg.SvcKit, false, "", sqlx.NewRepox[AuthzRepo](dsx, act.Ksgr))
		z.RegKey(zgg.SvcKit, false, "", sqlx.NewRepox[ConfxRepo](dsx, act.Ksgr))
		z.RegKey(zgg.SvcKit, false, "", sqlx.NewRepox[FrontaRepo](dsx, act.Ksgr))
		z.RegKey(zgg.SvcKit, false, "", sqlx.NewRepox[FrontvRepo](dsx, act.Ksgr))
		z.RegKey(zgg.SvcKit, false, "", sqlx.NewRepox[IngressRepo](dsx, act.Ksgr))
		z.RegKey(zgg.SvcKit, false, "", sqlx.NewRepox[ServiceRepo](dsx, act.Ksgr))
		z.RegKey(zgg.SvcKit, false, "", sqlx.NewRepox[RecordRepo](dsx, act.Ksgr))

		// 清理函数 ---------------------------------------------------
		return func() { dsc.Close() }
	})
}

// ===================================================================================

// 基础数据对象
type VBD struct {
	Disable sql.NullBool   `db:"disable"`
	Deleted sql.NullBool   `db:"deleted"`
	Updated sql.NullTime   `db:"updated"`
	Updater sql.NullString `db:"updater"`
	Created sql.NullTime   `db:"created"`
	Creater sql.NullString `db:"creater"`
	Version sql.NullInt64  `db:"version"`
}
