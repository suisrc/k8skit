package repo

import (
	"database/sql"
	"strings"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/ze/sqlx"
)

// ConfxDO ...
type ConfxDO struct {
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

/*
CREATE TABLE `confx` (
  `id` int NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `tag` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '标签: 特殊匹配， 用于追加一些特殊的匹配规则， key=val形式',
  `env` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '环境: DEV(开发), FAT(功能验收), UAT(用户验收), PRO(生产);\r\nDevelopment Environment;\r\nFunctional Acceptance Testing;\r\nUser Acceptance Testing;\r\nProduction Environment;',
  `app` varchar(128) DEFAULT NULL COMMENT '应用名称',
  `ver` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '应用版本(高版本替换低版本，聚合低版本不同的配置)',
  `kind` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '类型: ref(引用), env(环境), json，prop, yaml, toml ...',
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL COMMENT '编码: 应用名称_版本/环境名称/文件名称（''/''开头abspath)',
  `data` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci COMMENT '内容',
  `dkey` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '秘钥，与应用秘钥合用，完成data加密',
  `disable` int DEFAULT '0' COMMENT '禁用(禁用后，低版本会被删除)',
  `deleted` int DEFAULT '0' COMMENT '删除(删除后，使用低版本替代)',
  `updated` datetime DEFAULT NULL COMMENT '更新时间',
  `updater` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '更新者',
  `created` datetime DEFAULT NULL COMMENT '创建时间',
  `creater` varchar(255) DEFAULT NULL COMMENT '创建者',
  `version` int DEFAULT '0' COMMENT '版本',
  PRIMARY KEY (`id`),
  KEY `conf_app` (`app`),
  KEY `conf_code` (`code`),
  KEY `conf_env` (`env`),
  KEY `conf_ver` (`ver` DESC) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
*/

//=========================================================================================================================

type ConfxRepo struct {
	DS *sqlx.DB
}

func (*ConfxRepo) TableName() string {
	return C.DB.TablePrefix + "confx"
}

func (aa *ConfxRepo) SelectCols() string {
	return "SELECT id, tag, env, app, ver, kind, code, data, dkey, disable, deleted FROM " + aa.TableName()
}

// 获取一个签名
func (aa *ConfxRepo) GetConfig1(id int64) *ConfxDO {
	cfx := &ConfxDO{}
	err := aa.DS.Get(cfx, aa.SelectCols()+" WHERE deleted=0 and id=?", id)
	if err != nil {
		return nil
	}
	if cfx.Disable || cfx.Deleted || cfx.Code.String == "" {
		return nil
	}
	return cfx
}

// 获取签名集合
func (aa *ConfxRepo) GetConfigs(env, app, ver, kind string) []ConfxDO {
	cfxs := []ConfxDO{}
	aa.ConfLoop("", env, app, ver, kind, &cfxs, make(map[string]bool))
	return cfxs
}

// 递归获取签名
func (aa *ConfxRepo) ConfLoop(tag, env, app, ver, kind string, cfs *[]ConfxDO, cfm map[string]bool) {
	if aa.DS == nil {
		return // pass
	}
	if kind == "" {
		return // pass
	}
	//
	sql_ := aa.SelectCols() + " WHERE deleted=0"
	args := map[string]any{}
	// tag
	if tag != "" {
		sql_ += " AND tag like :tag"
		args["tag"] = "%" + tag + "%" // 模糊匹配
	}
	// env
	if env != "" {
		sql_ += " AND env=:env"
		args["env"] = env
	} else {
		sql_ += " AND env is null"
	}
	// app
	if app != "" {
		sql_ += " AND app=:app"
		args["app"] = app
	}
	// ver
	if ver != "" {
		sql_ += " AND ver<=:ver"
		args["ver"] = ver
	}
	// kind
	sql_ += " AND kind like :kind"
	args["kind"] = kind + "%"
	// order by
	sql_ += " ORDER BY ver DESC"
	// z.Println("sql:", sql_)
	// query by named
	rows, err := aa.DS.NamedQuery(sql_, args)
	if err != nil {
		z.Println("sql qry error:", err.Error())
		return
	}
	defer rows.Close()
	for rows.Next() {
		cfx := ConfxDO{}
		if err := rows.StructScan(&cfx); err != nil {
			z.Println("sql row error:", err.Error())
			continue
		}
		if cfx.Disable || cfx.Deleted || cfx.Code.String == "" {
			continue
		}
		if _, ok := cfm[cfx.Code.String]; ok {
			continue
		}
		cfm[cfx.Code.String] = true
		if !strings.HasSuffix(cfx.Kind.String, "-ref") {
			*cfs = append(*cfs, cfx)
			continue
		}
		if tag != "" {
			continue // tag by ref not support
		}
		// 递归处理相关配置
		aa.ConfLoop(tag, env, cfx.Code.String, cfx.Data.String, kind, cfs, cfm)
	}
}
