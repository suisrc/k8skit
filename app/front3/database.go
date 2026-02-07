package front3

import (
	"database/sql"
	"time"

	"github.com/suisrc/zgg/z"
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

type AuthzRepo struct {
	Database    *sqlx.DB
	TablePrefix string
}

func (aa *AuthzRepo) TableName() string {
	return aa.TablePrefix + "authz"
}

func (aa *AuthzRepo) SelectCols() string {
	return "SELECT id, appkey, secret, permiss, disable, deleted FROM " + aa.TableName()
}

// 通过 ak 获取令牌
func (aa *AuthzRepo) GetByAppKey(ak string) (*AuthzDO, error) {
	var ret AuthzDO
	err := aa.Database.Get(&ret, aa.SelectCols()+" WHERE appkey=? AND deleted=0", ak)
	return &ret, err
}

// ======================================================================================================================================================

// AppInfoDO ...
type AppInfoDO struct {
	ID       int64          `db:"id"`
	Tag      sql.NullString `db:"tag"`      // 标签
	Name     sql.NullString `db:"name"`     // 应用名称
	App      sql.NullString `db:"app"`      // 应用标识
	Vpp      sql.NullString `db:"vpp"`      // 版本名, 不存在，使用app代替
	Ver      sql.NullString `db:"ver"`      // 版本号
	Domain   sql.NullString `db:"domain"`   // 域名
	RootDir  sql.NullString `db:"rootdir"`  // 根目录
	Priority sql.NullString `db:"priority"` // 优先级
	Routers  sql.NullString `db:"routers"`  // 路由
	Disable  bool           `db:"disable"`  // 禁用
	Deleted  bool           `db:"deleted"`  // 删除
	// Updated sql.NullTime   `db:"updated"`
	// Updater sql.NullString `db:"updater"`
	// Created sql.NullTime   `db:"created"`
	// Creater sql.NullString `db:"creater"`
	// Version int            `db:"version"`
}

func (aa AppInfoDO) GVP() string {
	if aa.Vpp.String != "" {
		return aa.Vpp.String
	}
	return aa.App.String
}

type AppInfoRepo struct {
	Database    *sqlx.DB
	TablePrefix string
}

func (aa *AppInfoRepo) TableName() string {
	return aa.TablePrefix + "fronta"
}

func (aa *AppInfoRepo) SelectCols() string {
	return "SELECT id, tag, name, app, vpp, ver, domain, rootdir, priority, routers, disable, deleted FROM " + aa.TableName()
}

// 通过域名获取应用列表，排除删除的, 不排除禁用，以便于通知页面，应用被禁用
func (aa *AppInfoRepo) GetAllByDomain(domain string) ([]AppInfoDO, error) {
	var ret []AppInfoDO
	err := aa.Database.Select(&ret, aa.SelectCols()+" WHERE domain=? AND deleted=0", domain)
	return ret, err
}

// 通过应用编码查找应用
func (aa *AppInfoRepo) GetByApp(app string) (*AppInfoDO, error) {
	var ret AppInfoDO
	err := aa.Database.Get(&ret, aa.SelectCols()+" WHERE app=? AND deleted=0 LIMIT 1", app)
	return &ret, err
}

// 通过应用编码查找应用
func (aa *AppInfoRepo) GetByAppWithDelete(app string) (*AppInfoDO, error) {
	var ret AppInfoDO
	err := aa.Database.Get(&ret, aa.SelectCols()+" WHERE app=? LIMIT 1", app)
	return &ret, err
}

// 逻辑删除应用
func (aa *AppInfoRepo) DelByApp(app string) error {
	_, err := aa.Database.Exec("UPDATE "+aa.TableName()+" SET deleted=1, updated=?, updater=? WHERE app=?", time.Now(), z.AppName, app)
	return err
}

// 逻辑删除应用
func (aa *AppInfoRepo) DelByID(id int64) error {
	_, err := aa.Database.Exec("UPDATE "+aa.TableName()+" SET deleted=1, updated=?, updater=? WHERE id=?", time.Now(), z.AppName, id)
	return err
}

// ======================================================================================================================================================

// VersionDO ...
type VersionDO struct {
	ID        int64          `db:"id"`
	Tag       sql.NullString `db:"tag"`       // 标签
	Vpp       string         `db:"vpp"`       // 版本名
	Ver       string         `db:"ver"`       // 版本号
	Image     sql.NullString `db:"image"`     // 镜像
	TPRoot    sql.NullString `db:"tproot"`    // 替换根目录
	IndexPath sql.NullString `db:"indexpath"` // 索引文件
	Indexs    sql.NullString `db:"indexs"`    // 索引列表
	ImagePath sql.NullString `db:"imagepath"` // 输入文件
	ReCache   sql.NullBool   `db:"recache"`   // 重置缓存
	CdnCache  sql.NullBool   `db:"cdncache"`  // cdn 缓存 解决镜像重复加载问题
	CdnName   sql.NullString `db:"cdnname"`   // cdn 域
	CdnPath   sql.NullString `db:"cdnpath"`   // cdn 路径
	CdnPush   sql.NullBool   `db:"cdnpush"`   // cdn 使用
	CdnRenew  sql.NullBool   `db:"cdnrenew"`  // nil or true 启用cdn重写
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
	return `SELECT t1.id, t1.tag, t1.vpp, t1.ver, t1.image, t1.tproot, t1.indexpath, t1.indexs, t1.imagepath, t1.recache, t1.cdncache, t1.cdnname, t1.cdnpath, t1.cdnpush, t1.cdnrenew, t1.started, t1.indexhtml, t1.disable, t1.deleted FROM ` + aa.TableName() + " t1"
}

// 获取最新的版本， 排除禁用和删除和未生效的
func (aa *VersionRepo) GetTop1ByVppAndVer(vpp, ver string) (*VersionDO, error) {
	var ret VersionDO
	if ver == "" {
		err := aa.Database.Get(&ret, aa.SelectCols()+" WHERE t1.vpp=? AND (started<=NOW() OR started IS NULL) AND disable=0 AND deleted=0 ORDER BY ver DESC LIMIT 1", vpp)
		return &ret, err
	} else {
		err := aa.Database.Get(&ret, aa.SelectCols()+" WHERE t1.vpp=? AND t1.ver=? AND deleted=0", vpp, ver) // 忽略限制的条件, 除了deleted
		return &ret, err
	}
}

// 获取最新的版本
func (aa *VersionRepo) GetTop1ByVppAndVerWithDelete(vpp, ver string) (*VersionDO, error) {
	var ret VersionDO
	err := aa.Database.Get(&ret, aa.SelectCols()+" WHERE t1.vpp=? AND t1.ver=?", vpp, ver)
	return &ret, err
}

// 更新CDN信息， 更新 cdnname, cdnpath, cdnrew 字段
func (aa *VersionRepo) UpdateCdnInfo(data *VersionDO) error {
	_, err := aa.Database.Exec("UPDATE "+aa.TableName()+" SET cdnname=?, cdnpath=?, cdnrenew=? WHERE id=?", data.CdnName, data.CdnPath, data.CdnRenew, data.ID)
	return err
}

// 更新缓存信息
func (aa *VersionRepo) UpdateCacInfo(data *VersionDO) error {
	_, err := aa.Database.Exec("UPDATE "+aa.TableName()+" SET recache=? WHERE id=?", data.ReCache, data.ID)
	return err
}

// 通过镜像名称获取版本， 确定版本是否存在
func (aa *VersionRepo) GetByImage(image string) ([]VersionDO, error) {
	var ret []VersionDO
	err := aa.Database.Select(&ret, aa.SelectCols()+" WHERE t1.image=? AND deleted=0", image)
	return ret, err
}

// 通过 image 获取 ver 最大的数据，然后使用 vpp 去重, 考虑会有多个前端使用同一个镜像的问题
func (aa *VersionRepo) GetByImageName(name string) ([]VersionDO, error) {
	// SELECT * FROM (SELECT *, ROW_NUMBER() OVER (PARTITION BY vpp ORDER BY ver DESC) AS rn FROM frontv) AS sub WHERE rn = 1; 8.0.0+支持
	// SELECT * FROM frontv t1 WHERE NOT EXISTS (SELECT 1 FROM frontv t2 WHERE t2.vpp = t1.vpp AND t2.ver > t1.ver);
	var ret []VersionDO
	err := aa.Database.Select(&ret, aa.SelectCols()+" WHERE t1.image like ? AND t1.deleted=0 AND NOT EXISTS (SELECT 1 FROM frontv t2 WHERE t2.vpp = t1.vpp AND t2.ver > t1.ver);", name+":%")
	return ret, err
}

// 插入一条数据
func (aa *VersionRepo) Insert(data *VersionDO) error {
	ret, err := aa.Database.Exec("INSERT "+aa.TableName()+" SET tag=?, vpp=?, ver=?, image=?, tproot=?, indexpath=?, indexs=?, imagepath=?, cdncache=? cdnname=?, cdnpath=?, cdnpush=?, cdnrenew=?, started=?, indexhtml=?, disable=?, deleted=?", //
		data.Tag, data.Vpp, data.Ver, data.Image, data.TPRoot, data.IndexPath, data.Indexs, data.ImagePath, data.CdnCache, data.CdnName, data.CdnPath, data.CdnPush, data.CdnRenew, data.Started, data.IndexHtml, data.Disable, data.Deleted)
	if err == nil {
		data.ID, _ = ret.LastInsertId()
	}
	return err
}

// ======================================================================================================================================================

type IngressDO struct {
	ID       int64          `db:"id"`
	Ns       sql.NullString `db:"ns"`
	Name     sql.NullString `db:"name"`
	Clzz     sql.NullString `db:"clzz"`
	Host     sql.NullString `db:"host"`
	Template sql.NullString `db:"template"`
	Disable  bool           `db:"disable"`
	Deleted  bool           `db:"deleted"`
	Version  int            `db:"version"`
	Updated  sql.NullTime   `db:"updated"`
	Updater  sql.NullString `db:"updater"`
	Created  sql.NullTime   `db:"created"`
	Creater  sql.NullString `db:"creater"`
}

type IngressRepo struct {
	Database    *sqlx.DB
	TablePrefix string
}

func (aa *IngressRepo) TableName() string {
	return aa.TablePrefix + "ingress"
}

func (aa *IngressRepo) SelectCols() string {
	return `SELECT id, ns, name, clzz, host, htls, template, deleted, version ` + aa.TableName()
}

func (aa *IngressRepo) GetByNsAndName(ns, name string) (*IngressDO, error) {
	var ret IngressDO
	err := aa.Database.Get(&ret, aa.SelectCols()+" WHERE ns=? AND name=?", ns, name)
	return &ret, err
}

func (aa *IngressRepo) UpdateOne(data *IngressDO) error {
	_, err := aa.Database.Exec("UPDATE "+aa.TableName()+" SET ns=?, name=?, clzz=?, host=?, template=?, deleted=?, version=?, updated=?, updater=? WHERE id=?", //
		data.Ns, data.Name, data.Clzz, data.Host, data.Template, data.Deleted, data.Version+1, data.Updated, data.Updater, data.ID)
	return err
}

func (aa *IngressRepo) InsertOne(data *IngressDO) error {
	rst, err := aa.Database.Exec("INSERT "+aa.TableName()+" SET ns=?, name=?, clzz=?, host=?, template=?, deleted=?, version=?, created=?, creater=?", //
		data.Ns, data.Name, data.Clzz, data.Host, data.Template, data.Deleted, 0, data.Created, data.Creater)
	if err == nil {
		data.ID, _ = rst.LastInsertId()
	}
	return err
}

func (aa *IngressRepo) DeleteOne(data *IngressDO) error {
	// _, err := aa.Database.Exec("DELETE FROM "+aa.TableName()+" WHERE id=?", data.ID)
	_, err := aa.Database.Exec("UPDATE "+aa.TableName()+" SET deleted=1 WHERE id=?", data.ID)
	return err
}
