package zdb

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/ze/sqlx"
)

// data object for ActionDO
type ActionDO struct {
	ID    int64          `db:"id"`
	Sys   sql.NullString `db:"sys"`   // 系统
	Key   sql.NullString `db:"key"`   // 主键 [owner]_[metdho]; [GET|POST...]_[action]
	Name  sql.NullString `db:"name"`  // 名称
	Role  sql.NullString `db:"role"`  // 角色
	Ksql  sql.NullString `db:"ksql"`  // ksql
	Data  sql.NullString `db:"data"`  // 原始数据
	Html  sql.NullString `db:"html"`  // 描述信息
	Args  sql.NullString `db:"args"`  // 参数描述
	Mock  sql.NullString `db:"mock"`  // 模拟数据
	Ctype sql.NullString `db:"ctype"` // 响应类型, 默认 application/json
	Cstat sql.NullInt32  `db:"cstat"` // 响应状态, 默认 200
	Ckind sql.NullInt32  `db:"ckind"` // 响应方式，1. mock, 2. data, 3. ksql, 4. gateway ...

	VBD
}

func (ActionDO) TableName() string {
	return C.Database.TablePrefix + "action"
	// return sqlx.GetTableByEnv("action", "action")
}

// action repository
type ActionRepo struct {
	sqlx.Repo[ActionDO]
}

func (r *ActionRepo) Ksgr(key string) (string, error) {
	if !C.DBAction.Enable {
		return r.Kgr(key) // 未启用，直接使用本地处理器
	}
	act, err := r.GetBy(r.Dsc, r.ColsByInc("id", "sys", "key", "role", "ksql", "disable", "version"), nil, //
		"deleted=0 AND sys=? AND key=? ORDER BY version DESC LIMIT 1", C.DBAction.System, key)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}
	if act == nil || act.ID == 0 {
		ksql, err := r.Kgr(key) // 未找到，使用本地处理器
		if err != nil {
			return "", fmt.Errorf("ksql by database, not found, [%s] %s", C.DBAction.System, key)
		}
		z.Printf("ksql by database not found, use local, [%s] %s", C.DBAction.System, key)
		return ksql, nil
	}
	if act.Disable.Bool {
		return "", fmt.Errorf("ksql by database, is disable, [%s] %s", C.DBAction.System, key)
	}
	return act.Ksql.String, nil
}

func (r *ActionRepo) Request(zrc *z.Ctx) {
	method := zrc.Request.Method
	if method == "" {
		method = http.MethodGet
	}
	action := zrc.Action
	if C.DBAction.TpRoot != "" {
		action = strings.TrimPrefix(zrc.Action, C.DBAction.TpRoot)
	}
	key := method + "_" + action
	cond := "deleted=0 AND sys=? AND key=?"
	args := []any{C.DBAction.System, key}

	qry := zrc.Request.URL.Query()
	if version := qry.Get("version"); version != "" {
		cond += " AND version=?"
		args = append(args, version)
	}
	cond += " ORDER BY version DESC"
	act, err := r.GetBy(r.Dsc, r.ColsByExc("html", "args"), nil, cond, args...)
	// 执行逻辑
	if err != nil && err != sql.ErrNoRows {
		zrc.JERR(err, 0)
		return
	}
	if act == nil || act.ID == 0 {
		zrc.JERR(fmt.Errorf("no action from database for [%s]", key), 404)
		return
	}
	if act.Ctype.String == "" {
		act.Ctype = sqlx.NewString("application/json")
	}
	if act.Cstat.Int32 == 0 {
		act.Cstat = sqlx.NewInt32(200)
	}
	switch act.Ckind.Int32 {
	// ---------------------------------------------------------------------
	case 1: // mock 模式
		if act.Mock.String == "" {
			zrc.JERR(fmt.Errorf("no mock for [%s]", key), 404)
			return
		}
		btr := bytes.NewReader([]byte(act.Data.String))
		zrc.BYTE(btr, int(act.Cstat.Int32), act.Ctype.String)
		return
	// ---------------------------------------------------------------------
	case 2: // data 模式
		if act.Data.String == "" {
			zrc.JERR(fmt.Errorf("no data for [%s]", key), 404)
			return
		}
		if act.Data.String[0] == '{' {
			rst := map[string]any{}
			if err := json.Unmarshal([]byte(act.Data.String), &rst); err == nil {
				zrc.JSON(&z.Result{Success: true, Data: rst})
				return
			}
		}
		// 数据无法解析为json， 返回原始数据
		btr := bytes.NewReader([]byte(act.Data.String))
		zrc.BYTE(btr, int(act.Cstat.Int32), act.Ctype.String)
		return
	// ---------------------------------------------------------------------
	case 3: // ksql 模式
		if act.Ksql.String == "" {
			zrc.JERR(fmt.Errorf("no ksql for [%s]", key), 404)
			return
		}
		argv := map[string]any{} // 应该从 zrc 中获取参数
		{
			if method == http.MethodGet {
				for kk, vv := range qry {
					if len(vv) > 0 {
						argv[kk] = vv[0]
					} else {
						argv[kk] = nil
					}
				}
				// if _, err := z.ReadForm(zrc.Request, &argv); err != nil { zrc.JERR(err, 400); return }
			} else {
				if _, err := z.ReadBody(zrc.Request, &argv); err != nil {
					zrc.JERR(err, 400) // 获取参数错误
					return
				}
			}
		}
		page := &sqlx.Pagx{} // 应该从 req 中获取分页， pageNo, pageSize
		{
			if val := qry.Get("pageNo"); val != "" {
				if val, err := strconv.Atoi(val); err == nil {
					page.Page = int64(val)
				}
			}
			if val := qry.Get("pageSize"); val != "" {
				if val, err := strconv.Atoi(val); err == nil {
					page.Size = int64(val)
				}
			}
			if val := qry.Get("total"); val != "" {
				if val, err := strconv.Atoi(val); err == nil {
					page.IsTS = val == 1
				}
			}
			if page.Page == 0 && page.Size == 0 && !page.IsTS {
				page = nil // 忽略分页参数
			}
		}
		rst, siz, err := sqlx.Ksql[map[string]any](r.Dsc, act.Ksql.String, argv, page)
		if err != nil {
			zrc.JERR(err, 500)
			return
		}
		zrc.JSON(&z.Result{Success: true, Data: rst, Total: z.Ptr(int(siz))})
		return
	// ---------------------------------------------------------------------
	default:
		zrc.JERR(fmt.Errorf("unknow kind [%d] for [%s]", act.Ckind.Int32, key), 500)
		return
	}
}
