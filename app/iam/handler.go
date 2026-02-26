package iam

import (
	"k8skit/app"
	"k8skit/app/cache"
	"k8skit/app/zdb"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/ze/sqlx"
)

func init() {
	z.Register("30-iamapi", func(zgg *z.Zgg) z.Closed {
		z.C.Server.ReqXrtd = "2" // 强制使用[ali]模式，保持前端兼容
		api := z.Inject(zgg.SvcKit, &IamServeApi{})
		zgg.AddRouter("a/odic/login", z.MergeFunc(api.Authx, api.AodicLogin))
		zgg.AddRouter("a/odic/logout", z.MergeFunc(api.Authx, api.AodicLogout))
		zgg.AddRouter("a/odic/user_info", z.MergeFunc(api.Authx, api.AodicLogin))
		zgg.AddRouter("a/odic/test_repo", api.TestRepo)
		return nil
	})
}

type IamServeApi struct {
	Dscdb *sqlx.DB       `svckit:"auto"` // type, auto
	Cache cache.CacheX   `svckit:"auto"`
	Authz *zdb.AuthzRepo `svckit:"auto"`
}

// 返回校验结果
func (s *IamServeApi) AodicLogin(zrc *z.Ctx) {
	zrc.JSON(&z.Result{Success: true, Data: "ok"})
}

// 返回校验结果
func (s *IamServeApi) AodicLogout(zrc *z.Ctx) {
	user := zrc.Caches["user"].(*app.User)
	user.UserInfo = app.UserInfo{} // 清空之前的信息
	user.IsLogin = false

	referer := zrc.Request.Referer() // z.HA{"delay": 1000, "location": referer}
	zrc.JSON(&z.Result{Success: false, Data: z.HA{"location": referer}, ErrShow: 9})
}

// 返回用户信息
func (s *IamServeApi) AodicUserInfo(zrc *z.Ctx) {
	zrc.JSON(&z.Result{Success: true, Data: "ok"})
}

// 测试数据库链接
func (s *IamServeApi) TestRepo(zrc *z.Ctx) {
	var rst []zdb.AuthzDO
	var err error
	switch zrc.Request.URL.Query().Get("t") {
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
