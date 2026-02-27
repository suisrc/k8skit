package front3

import (
	"database/sql"

	"github.com/suisrc/zgg/z/ze/sqlx"
)

// AuthzDO ...
type AuthzDO struct {
	ID      int64          `db:"id"`
	AppKey  sql.NullString `db:"appkey"`  // 标识
	Secret  sql.NullString `db:"secret"`  // 秘钥
	Permiss sql.NullString `db:"permiss"` // 权限
	Disable bool           `db:"disable"` // 禁用
	Deleted bool           `db:"deleted"` // 删除

	// Remarks sql.NullString `db:"remarks"`
	// Updated sql.NullTime   `db:"updated"`
	// Updater sql.NullString `db:"updater"`
	// Created sql.NullTime   `db:"created"`
	// Creater sql.NullString `db:"creater"`
	// Version int            `db:"version"`
}

func (aa AuthzDO) TableName() string {
	return C.Front3.DB.TablePrefix + "authz"
}

// ----------------------------------------------------
type AuthzRepo struct {
	sqlx.Repo[AuthzDO]
}

// 通过 ak 获取令牌
func (aa *AuthzRepo) GetByAppKey(ak string) (*AuthzDO, error) {
	return aa.GetBy(aa.Dsc, aa.Cols(), nil, "appkey=? AND deleted=0", ak)
}
