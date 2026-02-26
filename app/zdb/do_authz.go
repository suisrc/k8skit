package zdb

import (
	"database/sql"

	"github.com/suisrc/zgg/z/ze/sqlx"
)

// authz data object
type AuthzDO struct {
	ID      int64          `db:"id"`
	Name    sql.NullString `db:"name"`
	AppKey  sql.NullString `db:"appkey"`
	Secret  sql.NullString `db:"secret"`
	Permiss sql.NullString `db:"permiss"`
	Remarks sql.NullString `db:"remarks"`

	BaseDO

	Expired sql.NullTime   `db:"expired"`
	String1 sql.NullString `db:"string1"`
	String2 sql.NullString `db:"string2"`
	String3 sql.NullString `db:"string3"`
}

func (AuthzDO) TableName() string {
	return C.Database.TablePrefix + "authz"
}

// authz repository
type AuthzRepo struct {
	sqlx.Repo[AuthzDO]
}

func (r *AuthzRepo) Test1() ([]AuthzDO, error) {
	rst, _, err := sqlx.Ksgs[AuthzDO](NewDsc(), ksgr, "authz_find_all", nil, nil)
	return rst, err
}

func (r *AuthzRepo) Test2() ([]AuthzDO, error) {
	rst, _, err := r.KsqlMap(NewDsc(), map[string]any{"id": 13}, nil)
	return rst, err
}

func (r *AuthzRepo) Test3() ([]AuthzDO, error) {
	rst, _, err := r.KsqlMap(NewDsc(), map[string]any{"id": 14}, nil)
	return rst, err
}
