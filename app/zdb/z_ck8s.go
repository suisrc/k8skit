package zdb

import (
	"database/sql"

	"github.com/suisrc/zgg/z/ze/sqlx"
)

// data object
type Zck8sDO struct {
	ID        int64          `db:"id"`
	Label     sql.NullString `db:"label"`
	Kind      sql.NullString `db:"kind"`
	Namespace sql.NullString `db:"namespace"`
	Name      sql.NullString `db:"name"`
	Data      sql.NullString `db:"data"`  // 原始请求数据
	Yaml      sql.NullString `db:"yaml"`  // 原始模版数据
	Yaml2     sql.NullString `db:"yaml2"` // 修改模版数据
	Json      sql.NullString `db:"json"`  // 原始模版数据
	Json2     sql.NullString `db:"json2"` // 修改模版数据

	VBD
}

func (Zck8sDO) TableName() string {
	return C.Database.TablePrefix + "zck8s"
	// return sqlx.GetTableByEnv("zrecord", "zrecord")
}

// record repository
type Zck8sRepo struct {
	sqlx.Repo[Zck8sDO]
}
