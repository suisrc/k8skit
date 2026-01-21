# k8skit

高度集成， 封装了一个前端服务

```
  -addr string
        http server addr (default "0.0.0.0")
  -api string
        http server api path
  -c string
        config file path
  -debug
        debug mode
  -dual
        running http and https server
  -eng string
        http server router engine (default "map")
  -f2folder value
        static folder (default /)
  -f2rmap value
        router path replace (default /api1/=http://127.0.0.1:8081/api2/)
  -f2rp value
        root dir parts list (default /zgg,/demo1/demo2)
  -f2show string
        show www resource uri
  -fxser
        http header flag xser-*
  -index string
        index file name (default "index.html")
  -indexs value
        index file name (default /zgg=index.htm)
  -local
        http server local mode
  -logtty
        logger to tty
  -native
        use native file server
  -port int
        http server Port (default 80)
  -print
        print mode
  -ptls int
        https server Port (default 443)
  -s3access string
        S3 账号
  -s3addrport string
        CND索引监听端口 (default "0.0.0.0:88")
  -s3bucket string
        S3 存储桶
  -s3domain string
        S3 CDN 域名
  -s3endpoint string
        S3 接口
  -s3region string
        S3 区域
  -s3rewrite
        S3 是否覆盖
  -s3rootdir string
        S3 根目录
  -s3secret string
        S3 秘钥
  -s3signer int
        S3 签名, 0: def, 1: v4(default), 2: v2, 3: v4stream, 4: anonymous (default 1)
  -s3ttoken value
        S3 临时令牌
  -suff value
        replace tmpl file suffix (default .html,.htm,.css,.map,.js)
  -syslog value
        logger to syslog server
  -tmpl string
        root router path (default "ROOT_PATH")
  -tpl string
        templates folder path
  -xrt string
        X-Request-Rt default value
```

## 使用说明


 k8s集群工具箱

[zgg](https://github.com/suisrc/zgg.git) Web服务框架

[k8skit](https://github.com/suisrc/k8skit.git) k8s工具包
- [ksidecar](https://github.com/suisrc/k8skit/tree/sidecar): k8s 边车注入服务
- [wgetar]: k8s 边车注入服务中，对于配置文件的获取服务， 基于busybox 的 wget+tar
- [front2]: 前端部署服务， 取代 nginx 作为前端容器，提供灵活的根路径配置等
- [kwdog2]: 由 kwdog2 + proxy2 组成的服务， 提供了 k8s 容器日志、监控、鉴权服务
- [kwlog2]: fluentbit 日志HTTP接受服务, 提供简单的日志存储和查询服务
- [front2s3](https://github.com/suisrc/k8skit/tree/front2s3): 扩展前端部署服务，提供将前端部署到S3CDN的服务