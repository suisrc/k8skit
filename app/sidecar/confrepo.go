package sidecar

import (
	"database/sql"
	"strings"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/ze/sqlx"
)

// ConfDO ...
type ConfDO struct {
	ID      int64          `db:"id"`
	Tag     sql.NullString `db:"tag"`
	Env     sql.NullString `db:"env"`
	App     sql.NullString `db:"app"`
	Ver     sql.NullString `db:"ver"`
	Kind    sql.NullString `db:"kind"`
	Code    sql.NullString `db:"code"`
	Data    sql.NullString `db:"data"`
	Dkey    sql.NullString `db:"dkey"`
	Disable bool           `db:"disable"`
	Deleted bool           `db:"deleted"`

	// Updated sql.NullTime   `db:"updated"`
	// Updater sql.NullString `db:"updater"`
	// Created sql.NullTime   `db:"created"`
	// Creater sql.NullString `db:"creater"`
	// Version int            `db:"version"`
}

func (aa ConfDO) TableName() string {
	return C.Sidecar.DB.TablePrefix + "confx"
}

type ConfRepo struct {
	sqlx.Repo[ConfDO]
}

// 获取一个配置
func (aa *ConfRepo) GetConfig1(id int64) *ConfDO {
	if aa.Dsc == nil {
		return nil
	}
	cfx, err := aa.GetBy(aa.Dsc, aa.Cols(), nil, "deleted=0 and id=?", id)
	if err != nil {
		return nil
	}
	if cfx.Disable || cfx.Deleted || cfx.Code.String == "" {
		return nil
	}
	return cfx
}

// 获取配置集合
func (aa *ConfRepo) GetConfigs(env, app, ver, kind string) []ConfDO {
	cfxs := []ConfDO{}
	aa._LoopConfig("", env, app, ver, kind, &cfxs, make(map[string]bool))
	return cfxs
}

// 递归获取配置
func (aa *ConfRepo) _LoopConfig(tag, env, app, ver, kind string, cfs *[]ConfDO, cfm map[string]bool) {
	if aa.Dsc == nil {
		return // pass
	}
	if kind == "" {
		return // pass
	}
	//
	cond := "deleted=0"
	args := map[string]any{}
	// tag
	if tag != "" {
		cond += " AND tag like :tag"
		args["tag"] = "%" + tag + "%" // 模糊匹配
	}
	// env
	if env != "" {
		cond += " AND env=:env"
		args["env"] = env
	} else {
		cond += " AND env is null"
	}
	// app
	if app != "" {
		cond += " AND app=:app"
		args["app"] = app
	}
	// ver
	if ver != "" {
		cond += " AND ver<=:ver"
		args["ver"] = ver
	}
	// kind
	cond += " AND kind like :kind"
	args["kind"] = kind + "%"
	// order by
	cond += " ORDER BY ver DESC"
	// query by named
	rows, err := aa.SelectByExc(aa.Dsc, nil, cond, args)
	if err != nil {
		z.Println("sql qry error:", err.Error())
		return
	}
	for _, cfx := range rows {
		// rows.StructScan(&cfx)
		if cfx.Disable || cfx.Deleted || cfx.Code.String == "" {
			continue // 忽略无效数据
		}
		if _, ok := cfm[cfx.Code.String]; ok {
			continue // 忽略重复数据
		}
		cfm[cfx.Code.String] = true
		if !strings.HasSuffix(cfx.Kind.String, "-ref") {
			*cfs = append(*cfs, cfx)
			continue // 数据加入返回列表
		}
		if tag != "" {
			continue // 忽略非默认标签数据
		}
		// 递归处理相关配置
		aa._LoopConfig(tag, env, cfx.Code.String, cfx.Data.String, kind, cfs, cfm)
	}
}
