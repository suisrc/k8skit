package iam

import (
	"k8skit/app"
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

		return nil
	})
}

type IamServeApi struct {
	DSC *sqlx.DB   `svckit:"auto"` // type, auto
	CAC zdb.CacheX `svckit:"auto"`
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
