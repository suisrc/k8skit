package zdb

import (
	"database/sql"

	"github.com/suisrc/zgg/z/ze/sqlx"
)

// data object ingress
type IngressDO struct {
	ID        int64          `db:"id"`
	Namespace sql.NullString `db:"ns"`
	Name      sql.NullString `db:"name"`
	Clzz      sql.NullString `db:"clzz"`
	Host      sql.NullString `db:"host"`
	MetaUid   sql.NullString `db:"metauid"`
	MetaVer   sql.NullString `db:"metaver"`
	Template  sql.NullString `db:"template"`

	VBD
}

func (IngressDO) TableName() string {
	return C.Database.TablePrefix + "ingress"
	// return sqlx.GetTableByEnv("ingress", "ingress")
}

// ingress repository
type IngressRepo struct {
	sqlx.Repo[IngressDO]
}
