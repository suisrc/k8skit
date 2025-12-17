package app

// 这是一个测试类， 需要屏蔽 init 函数

import (
	"kube-sidecar/z"
)

func init() {
	Init0()
}

func Init0() {
	z.Register("00-hello", func(svc z.SvcKit, enr z.Enroll) z.Closed {
		api := z.Inject(svc, &HelloApi{})
		enr("hello", api.hello)
		return func() {
			z.Println("api-hello closed")
		}
	})
	z.Register("zz-world", func(svc z.SvcKit, enr z.Enroll) z.Closed {
		api := svc.Get("HelloApi").(*HelloApi)
		enr(z.GET("world"), api.world)
		enr(z.GET("token"), z.TokenAuth(z.Ptr(""), api.token))
		return nil
	})
}

type HelloApi struct {
	FA any       // 标记不注入，默认
	FB any       `svckit:"-"`         // 标记不注入，默认
	CM z.IServer `svckit:"type"`      // 根据类型自动注入
	SK z.SvcKit  `svckit:"type"`      // 根据类型自动注入
	AH any       `svckit:"api-hello"` // 根据名称自动注入
	AW any       `svckit:"api-world"` // 根据名称自动注入
	TK z.TplKit  `svckit:"auto"`      // 根据名称自动注入
}

func (aa *HelloApi) hello(zrc *z.Ctx) bool {
	return zrc.JSON(&z.Result{Success: true, Data: "hello!"})
}
func (aa *HelloApi) world(zrc *z.Ctx) bool {
	return zrc.JSON(&z.Result{Success: true, Data: "world!"})
}
func (aa *HelloApi) token(zrc *z.Ctx) bool {
	return zrc.JSON(&z.Result{Success: true, Data: "token!"})
}
