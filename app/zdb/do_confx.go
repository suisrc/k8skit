package zdb

import (
	"database/sql"

	"github.com/suisrc/zgg/z/ze/sqlx"
)

// data object for confx
type ConfxDO struct {
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

func (ConfxDO) TableName() string {
	return C.Database.TablePrefix + "confx"
	// return sqlx.GetTableByEnv("confx", "confx")
}

// confx repository
type ConfxRepo struct {
	sqlx.Repo[ConfxDO]
}
