package front3

import (
	"database/sql"

	"github.com/suisrc/zgg/z/ze/sqlx"
)

// AppInfoDO ...
type AppInfoDO struct {
	ID       int64          `db:"id"`
	Tag      sql.NullString `db:"tag"`      // 标签
	Name     sql.NullString `db:"name"`     // 应用名称
	App      sql.NullString `db:"app"`      // 应用标识
	Domain   sql.NullString `db:"domain"`   // 域名
	RootDir  sql.NullString `db:"rootdir"`  // 根目录
	Priority sql.NullString `db:"priority"` // 优先级
	Disable  bool           `db:"disable"`  // 禁用
	Deleted  bool           `db:"deleted"`  // 删除
	// Updated sql.NullTime   `db:"updated"`
	// Updater sql.NullString `db:"updater"`
	// Created sql.NullTime   `db:"created"`
	// Creater sql.NullString `db:"creater"`
	// Version int            `db:"version"`
}

type AppInfoRepo struct {
	Database    *sqlx.DB
	TablePrefix string
}

func (aa *AppInfoRepo) TableName() string {
	return aa.TablePrefix + "fronta"
}

func (aa *AppInfoRepo) SelectCols() string {
	return "SELECT id, tag, name, app, domain, rootdir, priority, disable, deleted FROM " + aa.TableName()
}

// 通过域名获取应用列表，排除删除的, 不排除禁用，以便于通知页面，应用被禁用
func (aa *AppInfoRepo) GetAllByDomain(domain string) ([]AppInfoDO, error) {
	var ret []AppInfoDO
	err := aa.Database.Select(&ret, aa.SelectCols()+" WHERE domain=? AND deleted=0", domain)
	return ret, err
}

// VersionDO ...
type VersionDO struct {
	ID        int64          `db:"id"`
	Tag       sql.NullString `db:"tag"`       // 标签
	Aid       int64          `db:"aid"`       // 应用ID
	Ver       sql.NullString `db:"ver"`       // 版本
	Image     sql.NullString `db:"image"`     // 镜像
	TPRoot    sql.NullString `db:"tproot"`    // 替换根目录
	IndexPath sql.NullString `db:"indexpath"` // 索引文件
	Indexs    sql.NullString `db:"indexs"`    // 索引列表
	ImagePath sql.NullString `db:"imagepath"` // 输入文件
	CdnName   sql.NullString `db:"cdnname"`   // cdn 域
	CdnPath   sql.NullString `db:"cdnpath"`   // cdn 路径
	CdnUse    sql.NullBool   `db:"cdnuse"`    // cdn 使用
	CdnRew    sql.NullBool   `db:"cdnrew"`    // nil or true 启用cdn重写
	Started   sql.NullTime   `db:"started"`   // 生效时间
	IndexHtml sql.NullString `db:"indexhtml"` // 索引文件内容
	Disable   bool           `db:"disable"`   // 禁用
	Deleted   bool           `db:"deleted"`   // 删除
	// Updated sql.NullTime   `db:"updated"`
	// Updater sql.NullString `db:"updater"`
	// Created sql.NullTime   `db:"created"`
	// Creater sql.NullString `db:"creater"`
	// Version int            `db:"version"`
}

type VersionRepo struct {
	Database    *sqlx.DB
	TablePrefix string
}

func (aa *VersionRepo) TableName() string {
	return aa.TablePrefix + "frontv"
}

func (aa *VersionRepo) SelectCols() string {
	return "SELECT id, tag, aid, ver, image, tproot, indexpath, indexs, imagepath, cdnname, cdnpath, cdnuse, cdnrew, started, indexhtml, disable, deleted FROM " + aa.TableName()
}

// 获取最新的版本， 排除禁用和删除和未生效的
func (aa *VersionRepo) GetTop1ByAidAndVer(aid int64, ver string) (*VersionDO, error) {
	var ret VersionDO
	var err error
	if ver == "" {
		err = aa.Database.Get(&ret, aa.SelectCols()+" WHERE aid=? AND (started<=NOW() OR started IS NULL) AND disable=0 AND deleted=0 ORDER BY ver DESC LIMIT 1", aid)
	} else {
		// 忽略限制的条件, 除了deleted
		err = aa.Database.Get(&ret, aa.SelectCols()+" WHERE aid=? AND ver=? AND deleted=0", aid, ver)
	}
	return &ret, err
}

// 更新CDN信息， 更新 cdnname, cdnpath, cdnrew 字段
func (aa *VersionRepo) UpdateCdnInfo(data *VersionDO) error {
	_, err := aa.Database.Exec("UPDATE "+aa.TableName()+" SET cdnname=?, cdnpath=?, cdnrew=? WHERE id=?", data.CdnName, data.CdnPath, data.CdnRew, data.ID)
	return err
}
