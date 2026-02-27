package front3

import (
	"database/sql"
	"strings"
	"time"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/ze/sqlx"
)

// Frontv ...
type FrontvDO struct {
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

func (FrontvDO) TableName() string {
	return C.Front3.DB.TablePrefix + "frontv"
}

// ----------------------------------------------------
type FrontvRepo struct {
	sqlx.Repo[FrontvDO]
}

// 获取最新的版本， 排除禁用和删除和未生效的
func (aa *FrontvRepo) GetTop1ByVppAndVer(vpp, ver string) (*FrontvDO, error) {
	if ver == "" {
		return aa.GetBy(aa.Dsc, aa.Cols(), nil, "vpp=? AND (started<=NOW() OR started IS NULL) AND disable=0 AND deleted=0 ORDER BY ver DESC LIMIT 1", vpp)
	} else {
		return aa.GetBy(aa.Dsc, aa.Cols(), nil, "vpp=? AND ver=? AND deleted=0", vpp, ver) // 忽略限制的条件, 除了deleted
	}
}

// 获取最新的版本
func (aa *FrontvRepo) GetTop1ByVppAndVerWithDelete(vpp, ver string) (*FrontvDO, error) {
	return aa.GetBy(aa.Dsc, aa.Cols(), nil, "vpp=? AND ver=?", vpp, ver)
}

// 更新CDN信息， 更新 cdnname, cdnpath, cdnrew 字段
func (aa *FrontvRepo) UpdateCdnInfo(data *FrontvDO) error {
	return aa.UpdateByInc(aa.Dsc, data, "cdnname", "cdnpath", "cdnrenew")
}

// 更新缓存信息
func (aa *FrontvRepo) UpdateCacInfo(data *FrontvDO) error {
	return aa.UpdateByInc(aa.Dsc, data, "recache")
}

// 通过镜像名称获取版本， 确定版本是否存在
func (aa *FrontvRepo) GetByImage(image string) ([]FrontvDO, error) {
	return aa.SelectBy(aa.Dsc, aa.Cols(), "image=? AND deleted=0", image)
}

// 通过 image 获取 ver 最大的数据，然后使用 vpp 去重, 考虑会有多个前端使用同一个镜像的问题
func (aa *FrontvRepo) GetByImageName(name string) ([]FrontvDO, error) {
	// SELECT * FROM (SELECT *, ROW_NUMBER() OVER (PARTITION BY vpp ORDER BY ver DESC) AS rn FROM frontv) AS sub WHERE rn = 1 (8.0.0+支持)
	// SELECT * FROM frontv t1 WHERE NOT EXISTS (SELECT 1 FROM frontv t2 WHERE t2.vpp = t1.vpp AND t2.ver > t1.ver)
	return aa.SelectBy(aa.Dsc, aa.ColsByExc("t1."), "t1.image like ? AND t1.deleted=0 AND NOT EXISTS (SELECT 1 FROM frontv t2 WHERE t2.vpp = t1.vpp AND t2.ver > t1.ver)", name+":%")
}

// 插入一条数据
func (aa *FrontvRepo) Insert(data *FrontvDO) error {
	return aa.Insert(data)
}

func (aa *FrontvRepo) ModifyByInfo(info *FrontvDO, vpp, ver, img string, annos map[string]string) error {
	asql := "updated=?, updater=?, deleted=0, disable=0, vpp=?, ver=?, image=?"
	args := []any{time.Now(), z.AppName, vpp, ver, img}
	pre_ := "frontend/db.frontv."
	len_ := len(pre_)
	for anno, data := range annos {
		if anno == pre_+"image" || anno == pre_+"vpp" || anno == pre_+"ver" {
			continue
		}
		if strings.HasPrefix(anno, pre_) {
			key := anno[len_:]
			switch data {
			case "true":
				asql += "," + key + "=1"
			case "false":
				asql += "," + key + "=0"
			default:
				asql += "," + key + "=?"
				args = append(args, data)
			}
		}
	}
	if info.ID > 0 {
		args = append(args, info.ID)
		_, err := aa.Dsc.Ext().Exec("UPDATE "+info.TableName()+" SET "+asql+" WHERE id=?", args...)
		if err != nil {
			return err // 更新数据库发生异常
		}
	} else {
		asql += ", created=?, creater=?"
		args = append(args, time.Now(), z.AppName)
		ret, err := aa.Dsc.Ext().Exec("INSERT "+info.TableName()+" SET "+asql, args...)
		if err != nil {
			return err // 插入数据库发生异常
		}
		info.ID, _ = ret.LastInsertId()
		args = append(args, info.ID)
	}
	z.Println("[_mutate_]:", "update/insert app version info into database,", asql, z.ToStr(args))
	return nil
}
