# zgg

## 框架介绍

这是一极精简的 基于 基于 net/http web 服务框架， 没有 router ，为单接口而生。  
通过 query.action(query参数) 或者 path=action(path就是action) 两种方式确定 handle 函数。  


这是一极精简的 web 服务框架， 默认没有 routing， 使用 map 进行 action 检索 ，为单接口而生。  
通过 query.action(query参数) 或者 path=action(path就是action) 两种方式确定 handle 函数。  
但是，可以通过 '-mux' 参数切换使用 net/http.mux 标准库的 http.router 完成标准 web 服务切换。  
当前框架不依赖任何第三方库， 纯原生实现。  

  
为什么需要它？  
在很多项目中，可能只需要几个接口， 而为这些接口无论使用 gin, echo, iris, fasthttp...我认为都是不值当的。因此它就诞生了。  

  
自动注入wire?  
wire 是一个依赖注入框架， 但是考虑到框架本身就比较小，本身不依赖任何第三方库，所以不会集成wire， 如果需要，可以考虑自行增加。  
但是，实现了一个简单的注入封装，`svckit:"auto"`, 可以自动注入依赖。例如：

```go
package app

import (
	"kube-sidecar/z"
)

func init() {
	z.Register("00-hello", func(svc z.SvcKit, enr z.Enroll) z.Closed {
		api := z.RegSvc(svc, "api-hello", &HelloApi{})
		enr("hello", api.hello)
		return func() {
			z.Println("api-hello closed")
		}
	})
	z.Register("zz-world", func(svc z.SvcKit, enr z.Enroll) z.Closed {
		api := svc.Get("api-hello").(*HelloApi)
		enr(z.GET("world"), api.world)
		enr(z.GET("token"), z.TokenAuth(z.Ptr(""), api.token))
		return nil
	})
}

type HelloApi struct {
	FA any                           // 标记不注入，默认
	FB any      `svckit:"-"`         // 标记不注入，默认
	CM z.Module `svckit:"type"`      // 根据【类型】自动注入
	SK z.SvcKit `svckit:"type"`      // 根据【类型】自动注入
	AH any      `svckit:"api-hello"` // 根据【名称】自动注入
	AW any      `svckit:"api-world"` // 根据【名称】自动注入
	TK z.TplKit `svckit:"auto"`      // 根据【类型】自动注入
	tt z.TplKit `svckit:"auto"`      // 私有【属性】不能注入
}

```



这是一极精简的 web 服务框架， 默认没有 routing， 使用 map 进行 action 检索 ，为单接口而生。  
通过 query.action(query参数) 或者 path=action(path就是action) 两种方式确定 handle 函数。  
但是，可以通过 '-mux' 参数切换使用 net/http.mux 标准库的 http.router 完成标准 web 服务切换。  
  
[z]包为 zgg(z? google golang) 框架抽象层，尽量不要修改，以免出现兼容性问题  
  
## 快速开始

```sh
# 命令
xxx [command] [arguments]

xxx web (default)
  -debug bool # debug mode
        debug mode
  -local bool # local mode， addr = 127.0.0.1
        http server local mode
  -addr string # 服务绑定的ip
        http server addr (default "0.0.0.0")
  -port int # 服务绑定的 port
        http server Port (default 80)
  -crtPath string # 服务绑定的 cer file，https 模式
        http server cer file
  -keyPath string # 服务绑定的 key file，https 模式
        http server key file
  -mux   bool # http server with mux， 默认为 false， 使用 net/http.mux 作为服务
        http server with mux
  -apiPath string # 服务绑定的 api path
        http server api path
  -token string # 服务绑定的 api token
        http server api token

xxx version # 查看应用版本

xxx cert # 生成证书

xxx hello # 测试 hello world

xxx -h # 查看帮助(仅限web模式)


# 示例, 默认 web 模式
xxx -debug -local

```

## 其他说明

核心包 [z]， 其他都可以删除， [z/serve.go] 是一个 server 的标准实现，可以自行修改，增加启动参数。[app] 增加 web 服务， [cmd] 增加 命令行服务。