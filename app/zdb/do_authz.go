package zdb

import (
	"database/sql"

	"github.com/suisrc/zgg/z/ze/sqlx"
)

// data object for authz
type AuthzDO struct {
	ID      int64          `db:"id"`
	Name    sql.NullString `db:"name"`
	AppKey  sql.NullString `db:"appkey"`
	Secret  sql.NullString `db:"secret"`
	Permiss sql.NullString `db:"permiss"`
	Remarks sql.NullString `db:"remarks"`

	Expired sql.NullTime   `db:"expired"`
	String1 sql.NullString `db:"string1"`
	String2 sql.NullString `db:"string2"`
	String3 sql.NullString `db:"string3"`

	VBD
}

func (AuthzDO) TableName() string {
	return C.Database.TablePrefix + "authz"
	// return sqlx.GetTableByEnv("authz", "authz")
}

// authz repository
type AuthzRepo struct {
	sqlx.Repo[AuthzDO]
}

// --------------------------------------------------------------------------
// 测试内容

func (r *AuthzRepo) Test1() ([]AuthzDO, error) {
	// 指定 ksql 文件
	rst, _, err := sqlx.Ksgs[AuthzDO](r.Dsc, r.Kgr, "authz_find_all", nil, nil)
	return rst, err
}

func (r *AuthzRepo) Test2() ([]AuthzDO, error) {
	// 通过 [结构体]_[方法名] 获取 ksql 文件
	rst, _, err := r.KsqlMap(r.Dsc, map[string]any{"id": 13}, nil)
	return rst, err
}

func (r *AuthzRepo) Test3() ([]AuthzDO, error) {
	// 指定不存在的 ksql 文件
	rst, _, err := r.KsqlMap(r.Dsc, map[string]any{"id": 14}, nil)
	return rst, err
}

// 测试内容
// --------------------------------------------------------------------------
