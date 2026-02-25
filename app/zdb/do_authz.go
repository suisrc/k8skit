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

func NewAuthzRepo() *AuthzRepo {
	return sqlx.NewRepo[AuthzRepo]()
}

// authz repository
type AuthzRepo struct {
	sqlx.Repo[AuthzDO]
}

func (r *AuthzRepo) FindAll() ([]AuthzDO, error) {
	rst, _, err := Ksql[AuthzDO]("authz_find_all", nil, false)
	return rst, err
}
