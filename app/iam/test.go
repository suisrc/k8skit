package iam

import (
	"k8skit/app/zdb"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/ze/sqlx"
)

// 测试数据库链接
func (s *IamServeApi) TestRepo(zrc *z.Ctx) {
	var rst []zdb.AuthzDO
	var err error
	switch zrc.Request.URL.Query().Get("t") {
	case "0":
		s.TestAllDB(zrc)
		return
	case "2":
		rst, err = s.Authz.Test2()
	case "3":
		rst, err = s.Authz.Test3()
	default:
		rst, err = s.Authz.Test1()
	}
	z.Println("[testrepo]:", z.ToStr(rst), err)
	if err != nil {
		z.Println("[tstcache]:", z.ToStr(sqlx.KsqlStmCache))
		zrc.JERR(err, 0)
	} else {
		zrc.JSON(&z.Result{Success: true, Data: rst})
	}
}

func (s *IamServeApi) TestAllDB(zrc *z.Ctx) {
	dsc := zdb.NewDsc()
	rst := []any{}
	if zdb, err := s.Authz.Get(dsc, 1); err != nil {
		rst = append(rst, "[authz error]: "+err.Error())
	} else {
		rst = append(rst, zdb)
	}
	if zdb, err := s.Confx.Get(dsc, 1); err != nil {
		rst = append(rst, "[confx error]: "+err.Error())
	} else {
		rst = append(rst, zdb)
	}
	if zdb, err := s.Fronta.Get(dsc, 1); err != nil {
		rst = append(rst, "[fronta error]: "+err.Error())
	} else {
		rst = append(rst, zdb)
	}
	if zdb, err := s.Frontv.Get(dsc, 1); err != nil {
		rst = append(rst, "[frontv error]: "+err.Error())
	} else {
		rst = append(rst, zdb)
	}
	if zdb, err := s.Ingress.Get(dsc, 1); err != nil {
		rst = append(rst, "[ingress error]: "+err.Error())
	} else {
		rst = append(rst, zdb)
	}
	if zdb, err := s.Service.Get(dsc, 1); err != nil {
		rst = append(rst, "[service error]: "+err.Error())
	} else {
		rst = append(rst, zdb)
	}
	if zdb, err := s.Zrecord.Get(dsc, 1); err != nil {
		rst = append(rst, "[zrecord error]: "+err.Error())
	} else {
		rst = append(rst, zdb)
	}
	zrc.JSON(&z.Result{Success: true, Data: rst})
}
