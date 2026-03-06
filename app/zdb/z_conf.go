package zdb

import (
	"database/sql"

	"github.com/suisrc/zgg/z/ze/sqlx"
)

// data object for confx
type ZconfDO struct {
	ID   int64          `db:"id"`
	Tag  sql.NullString `db:"tag"`
	Name sql.NullString `db:"name"`
	Env  sql.NullString `db:"env"`
	App  sql.NullString `db:"app"`
	Ver  sql.NullString `db:"ver"`
	Kind sql.NullString `db:"kind"`
	Code sql.NullString `db:"code"`
	Data sql.NullString `db:"data"`
	Dkey sql.NullString `db:"dkey"`

	VBD
}

func (ZconfDO) TableName() string {
	return C.Database.TablePrefix + "zconf"
	// return sqlx.GetTableByEnv("confx", "confx")
}

// confx repository
type ZconfRepo struct {
	sqlx.Repo[ZconfDO]
}
