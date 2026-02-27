package front3

import (
	"database/sql"
	"strings"
	"time"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/ze/sqlx"
)

// Fronta ...
type FrontaDO struct {
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

func (aa FrontaDO) TableName() string {
	return C.Front3.DB.TablePrefix + "fronta"
}

func (aa FrontaDO) GVP() string {
	if aa.Vpp.String != "" {
		return aa.Vpp.String
	}
	return aa.App.String
}

// ----------------------------------------------------
type FrontaRepo struct {
	sqlx.Repo[FrontaDO]
}

// 通过域名获取应用列表，排除删除的, 不排除禁用，以便于通知页面，应用被禁用
func (aa *FrontaRepo) GetAllByDomain(domain string) ([]FrontaDO, error) {
	return aa.SelectBy(aa.Dsc, aa.Cols(), "domain=? AND deleted=0", domain)
}

// 通过应用编码查找应用
func (aa *FrontaRepo) GetByApp(app string) (*FrontaDO, error) {
	return aa.GetBy(aa.Dsc, aa.Cols(), nil, "app=? AND deleted=0 LIMIT 1", app)
}

// 通过应用编码查找应用
func (aa *FrontaRepo) GetByAppWithDelete(app string) (*FrontaDO, error) {
	return aa.GetBy(aa.Dsc, aa.Cols(), nil, "app=? LIMIT 1", app)
}

// 逻辑删除应用
func (aa *FrontaRepo) DelByApp(app string) error {
	tbl := FrontaDO{}.TableName()
	_, err := aa.Dsc.Ext().Exec("UPDATE "+tbl+" SET deleted=1, updated=?, updater=? WHERE app=?", time.Now(), z.AppName, app)
	return err
}

// 逻辑删除应用
func (aa *FrontaRepo) DelByID(id int64) error {
	tbl := FrontaDO{}.TableName()
	_, err := aa.Dsc.Ext().Exec("UPDATE "+tbl+" SET deleted=1, updated=?, updater=? WHERE id=?", time.Now(), z.AppName, id)
	return err
}

// 变动应用信息
func (aa *FrontaRepo) ModifyByInfo(info *FrontaDO, app, ver, domain, rootdir string, annos map[string]string) error {
	asql := "updated=?, updater=?, deleted=0, disable=0, app=?, ver=?, domain=?, rootdir=?"
	args := []any{time.Now(), z.AppName, app, ver, domain, rootdir}
	pre_ := "frontend/db.fronta."
	len_ := len(pre_)
	for anno, data := range annos {
		if anno == pre_+"app" || anno == pre_+"ver" || anno == pre_+"domain" || anno == pre_+"rootdir" {
			continue
		}
		if strings.HasPrefix(anno, pre_) {
			key := anno[len_:]
			if key == "vpp" {
				info.Vpp.String = data // 更新应用名，后面需要使用最新的
			}
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
	z.Println("[_mutate_]:", "update/insert appinfo into database,", asql, z.ToStr(args))
	return nil
}
