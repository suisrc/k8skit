package iam

import (
	"k8skit/app/cache"
	"k8skit/app/zdb"
	"k8skit/usr"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/ze/sqlx"
)

func init() {
	z.Register("30-iam", func(zgg *z.Zgg) z.Closed {
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
	Dscdb *sqlx.DB     `svckit:"auto"` // type, auto
	Cache cache.CacheX `svckit:"auto"`

	Authz   *zdb.AuthzRepo   `svckit:"auto"`
	Confx   *zdb.ConfxRepo   `svckit:"auto"`
	Fronta  *zdb.FrontaRepo  `svckit:"auto"`
	Frontv  *zdb.FrontvRepo  `svckit:"auto"`
	Ingress *zdb.IngressRepo `svckit:"auto"`
	Service *zdb.ServiceRepo `svckit:"auto"`
	Zrecord *zdb.ZrecordRepo `svckit:"auto"`
}

// 返回校验结果
func (s *IamServeApi) AodicLogin(zrc *z.Ctx) {
	zrc.JSON(&z.Result{Success: true, Data: "ok"})
}

// 返回校验结果
func (s *IamServeApi) AodicLogout(zrc *z.Ctx) {
	user := zrc.Caches["user"].(*usr.User)
	user.UserInfo = usr.UserInfo{} // 清空之前的信息
	user.IsLogin = false

	referer := zrc.Request.Referer() // z.HA{"delay": 1000, "location": referer}
	zrc.JSON(&z.Result{Success: false, Data: z.HA{"location": referer}, ErrShow: 9})
}

// 返回用户信息
func (s *IamServeApi) AodicUserInfo(zrc *z.Ctx) {
	zrc.JSON(&z.Result{Success: true, Data: "ok"})
}
