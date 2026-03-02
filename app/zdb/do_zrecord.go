package zdb

import (
	"database/sql"

	"github.com/suisrc/zgg/z/ze/sqlx"
)

// data object zrecord
type RecordDO struct {
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

func (RecordDO) TableName() string {
	return C.Database.TablePrefix + "zrecord"
	// return sqlx.GetTableByEnv("zrecord", "zrecord")
}

// record repository
type RecordRepo struct {
	sqlx.Repo[RecordDO]
}
