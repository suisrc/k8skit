package front3

import (
	"database/sql"

	"github.com/suisrc/zgg/z/ze/sqlx"
)

type RecordDO struct {
	ID         int64          `db:"id"`
	ApiVersion sql.NullString `db:"apiversion"`
	Kind       sql.NullString `db:"kind"`
	Namespace  sql.NullString `db:"namespace"`
	Name       sql.NullString `db:"name"`
	MetaUid    sql.NullString `db:"metauid"`
	MetaVer    sql.NullString `db:"metaver"`
	Template   sql.NullString `db:"template"`
	Disable    bool           `db:"disable"`
	Deleted    bool           `db:"deleted"`
	Updated    sql.NullTime   `db:"updated"`
	Updater    sql.NullString `db:"updater"`
	Created    sql.NullTime   `db:"created"`
	Creater    sql.NullString `db:"creater"`
	Version    int            `db:"version"`
}

func (aa RecordDO) TableName() string {
	return C.Front3.DB.TablePrefix + "record"
}

// --------------------------------------------
type RecordRepo struct {
	sqlx.Repo[RecordDO]
}

func (aa RecordRepo) LstRecordBy(kind, apiv, ns, name string, delete bool) ([]RecordDO, error) {
	return aa.SelectBy(aa.Dsc, aa.Cols(), "kind=? AND apiversion=? AND namespace=? AND name=? AND deleted=?", kind, apiv, ns, name, delete)
}

func (aa RecordRepo) InsertOne(data *RecordDO) error {
	return aa.Insert(aa.Dsc, data)
}

func (aa RecordRepo) DeleteOne(data *RecordDO) error {
	stm := "UPDATE " + data.TableName() + " SET deleted=1, updated=?, updater=? WHERE id=?"
	_, err := aa.Dsc.Ext().Exec(stm, data.Updated, data.Updater, data.ID)
	return err
}
