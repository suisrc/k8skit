package zdb

import (
	"database/sql"

	"github.com/suisrc/zgg/z/ze/sqlx"
)

// data object frontv
type FrontvDO struct {
	ID        int64          `db:"id"`
	Tag       sql.NullString `db:"tag"`
	Vpp       sql.NullString `db:"vpp"`
	Ver       sql.NullString `db:"ver"`
	Image     sql.NullString `db:"image"`
	Tproot    sql.NullString `db:"tproot"`
	IndexPath sql.NullString `db:"indexpath"`
	Indexs    sql.NullString `db:"indexs"`
	ImagePath sql.NullString `db:"imagepath"`
	Recache   sql.NullBool   `db:"recache"`
	CdnCache  sql.NullBool   `db:"cdncache"`
	CdnName   sql.NullString `db:"cdnname"`
	CdnPath   sql.NullString `db:"cdnpath"`
	CdnPush   sql.NullBool   `db:"cdnpath"`
	CdnRenew  sql.NullBool   `db:"cdnrenew"`
	Started   sql.NullTime   `db:"started"`
	IndexHtml sql.NullString `db:"indexhtml"`

	VBD
}

func (FrontvDO) TableName() string {
	return C.Database.TablePrefix + "frontv"
	// return sqlx.GetTableByEnv("frontv", "frontv")
}

// frontv repository
type FrontvRepo struct {
	sqlx.Repo[FrontvDO]
}
