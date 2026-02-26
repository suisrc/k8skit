package zdb

import (
	"database/sql"

	"github.com/suisrc/zgg/z/ze/sqlx"
)

// data object zrecord
type ZrecordDO struct {
	ID        int64          `db:"id"`
	ApiVer    sql.NullString `db:"apiversion"`
	Kind      sql.NullString `db:"kind"`
	Namespace sql.NullString `db:"namespace"`
	Name      sql.NullString `db:"name"`
	MetaUid   sql.NullString `db:"metauid"`
	MetaVer   sql.NullString `db:"metaver"`
	Template  sql.NullString `db:"template"`

	VBD
}

func (ZrecordDO) TableName() string {
	return C.Database.TablePrefix + "zrecord"
	// return sqlx.GetTableByEnv("zrecord", "zrecord")
}

// zrecord repository
type ZrecordRepo struct {
	sqlx.Repo[ZrecordDO]
}
