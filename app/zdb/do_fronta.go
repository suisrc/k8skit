package zdb

import (
	"database/sql"

	"github.com/suisrc/zgg/z/ze/sqlx"
)

// data object for fronta
type FrontaDO struct {
	ID       int64          `db:"id"`
	Tag      sql.NullString `db:"tag"`
	Name     sql.NullString `db:"name"`
	App      sql.NullString `db:"app"`
	Vpp      sql.NullString `db:"vpp"`
	Ver      sql.NullString `db:"ver"`
	Domain   sql.NullString `db:"domain"`
	Rootdir  sql.NullString `db:"rootdir"`
	Priority sql.NullString `db:"priority"`
	Routers  sql.NullString `db:"routers"`

	VBD
}

func (FrontaDO) TableName() string {
	return C.Database.TablePrefix + "fronta"
	// return sqlx.GetTableByEnv("fronta", "fronta")
}

// fronta repository
type FrontaRepo struct {
	sqlx.Repo[FrontaDO]
}
