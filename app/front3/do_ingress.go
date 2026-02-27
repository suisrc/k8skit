package front3

import (
	"database/sql"
	"time"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/ze/sqlx"
)

type IngressDO struct {
	ID       int64          `db:"id"`
	Ns       sql.NullString `db:"ns"`
	Name     sql.NullString `db:"name"`
	Clzz     sql.NullString `db:"clzz"`
	Host     sql.NullString `db:"host"`
	MetaUid  sql.NullString `db:"metauid"`
	MetaVer  sql.NullString `db:"metaver"`
	Template sql.NullString `db:"template"`
	Disable  bool           `db:"disable"`
	Deleted  bool           `db:"deleted"`
	Updated  sql.NullTime   `db:"updated"`
	Updater  sql.NullString `db:"updater"`
	Created  sql.NullTime   `db:"created"`
	Creater  sql.NullString `db:"creater"`
	Version  int            `db:"version"`
}

func (IngressDO) TableName() string {
	return C.Front3.DB.TablePrefix + "ingress"
}

// ---------------------------------------------
type IngressRepo struct {
	sqlx.Repo[IngressDO]
}

func (aa *IngressRepo) GetBySpaceAndName(space, name string) (*IngressDO, error) {
	return aa.GetBy(aa.Dsc, aa.Cols(), nil, "ns=? AND name=? AND deleted=0 LIMIT 1 ORDER BY id DESC", space, name)
}

func (aa *IngressRepo) GetBySpaceAndNames(space, name string) ([]IngressDO, error) {
	return aa.SelectBy(aa.Dsc, aa.Cols(), "ns=? AND name=? AND deleted=0", space, name)
}

func (aa *IngressRepo) UpdateOne(data *IngressDO) error {
	return aa.Update(aa.Dsc, data)
}

func (aa *IngressRepo) InsertOne(data *IngressDO) error {
	return aa.Insert(aa.Dsc, data)
}

func (aa *IngressRepo) DeleteOne(data *IngressDO) error {
	_, err := aa.Dsc.Ext().Exec("UPDATE "+data.TableName()+" SET deleted=1, updated=?, updater=? WHERE id=?", time.Now(), z.AppName, data.ID)
	return err
}
