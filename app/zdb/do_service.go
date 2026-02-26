package zdb

import (
	"database/sql"

	"github.com/suisrc/zgg/z/ze/sqlx"
)

// data object service
type ServiceDO struct {
	ID        int64          `db:"id"`
	Tag       sql.NullString `db:"tag"`
	Namespace sql.NullString `db:"ns"`
	App       sql.NullString `db:"app"`
	Ver       sql.NullString `db:"ver"`
	Ports     sql.NullString `db:"ports"`
	Confx     sql.NullString `db:"confx"`
	Kwdog     sql.NullString `db:"kwdog"`
	Image     sql.NullString `db:"image"`
	Replicas  sql.NullInt64  `db:"replicas"`
	MetaUid   sql.NullString `db:"metauid"`
	MetaVer   sql.NullString `db:"metaver"`
	Template  sql.NullString `db:"template"`
	Async     sql.NullBool   `db:"async"`
	Atime     sql.NullTime   `db:"atime"`
	Error     sql.NullString `db:"error"`

	VBD
}

func (ServiceDO) TableName() string {
	return C.Database.TablePrefix + "service"
	// return sqlx.GetTableByEnv("service", "service")
}

// service repository
type ServiceRepo struct {
	sqlx.Repo[ServiceDO]
}
